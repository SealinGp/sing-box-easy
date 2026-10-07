package proxies

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/SealinGp/sing-box-easy/app/pkg/fault"
	"github.com/SealinGp/sing-box-easy/app/pkg/singbox/clashapi"
	"github.com/SealinGp/sing-box-easy/app/pkg/singbox/config"
)

// testTimeout bounds ONE node's latency test. It must stay below the Clash API
// client's own 10s request timeout, or the transport gives up before sing-box
// can answer and every slow node becomes an unexplained error.
const testTimeout = 5 * time.Second

// groupTestConcurrency is how many members are tested at once — sing-box's own
// batch size for the same job, and few enough not to starve a router that is
// also routing traffic.
const groupTestConcurrency = 10

// groupTestCeiling bounds a whole group test. Worst case is every member dead:
// ceil(n/10) waves of testTimeout, so 90s covers about 180 dead nodes; past
// that the members not reached are simply absent from the result.
const groupTestCeiling = 90 * time.Second

// Runtime is the slice of the Clash API this package uses. `*clashapi.Client`
// satisfies it; tests use a fake.
type Runtime interface {
	Proxies(ctx context.Context) ([]clashapi.Proxy, error)
	SelectProxy(ctx context.Context, group, name string) error
	Delay(ctx context.Context, tag, testURL string, timeout time.Duration) (int, error)
}

// Settings is what the service reads from the sing-box config.
type Settings interface {
	GetClashAPISettings() (config.ClashAPISettings, error)
	GetConfigSection(name string) (json.RawMessage, bool, error)
}

// Service serves the runtime proxies page.
type Service struct {
	settings Settings
	// connect builds the Clash API client. A field so tests can substitute it.
	connect func(externalController, secret string) (Runtime, error)
}

// NewService builds the service over the config manager.
func NewService(settings Settings) *Service {
	return &Service{
		settings: settings,
		connect: func(externalController, secret string) (Runtime, error) {
			return clashapi.NewFromValues(externalController, secret)
		},
	}
}

// SelectCommand is the body of `PUT /runtime/proxies/selection`.
//
// Names travel in the body, never in the path: an outbound tag is free text
// that routinely carries spaces, emoji, "|" and occasionally "/", and a path
// segment survives none of those reliably through a router and a reverse proxy.
type SelectCommand struct {
	Group string `json:"group"`
	Name  string `json:"name"`
}

// TestCommand is the body of the two delay endpoints.
type TestCommand struct {
	Name string `json:"name"`
}

// NodeDelay is the result of testing one outbound.
type NodeDelay struct {
	Name string `json:"name"`
	// Delay is 0 when the test failed — a failed test is a result, not an error.
	Delay int `json:"delay"`
}

// GroupDelays is the result of testing a group. Members that did not answer
// are absent from Delays.
type GroupDelays struct {
	Group  string         `json:"group"`
	Delays map[string]int `json:"delays"`
}

// View returns every group of the running sing-box.
func (s *Service) View(ctx context.Context) (View, error) {
	runtime, err := s.runtime()
	if err != nil {
		return View{}, err
	}
	list, err := runtime.Proxies(ctx)
	if err != nil {
		return View{}, unavailable(err)
	}
	return BuildView(list, s.selectionPersisted()), nil
}

// Select switches a selector to one of its members, in the running process.
func (s *Service) Select(ctx context.Context, command SelectCommand) error {
	group, name := strings.TrimSpace(command.Group), strings.TrimSpace(command.Name)
	if group == "" || name == "" {
		return fault.New(fault.Input, "group and name are required")
	}
	runtime, err := s.runtime()
	if err != nil {
		return err
	}
	err = runtime.SelectProxy(ctx, group, name)
	switch {
	case err == nil:
		return nil
	case errors.Is(err, clashapi.ErrSelectRejected):
		return &fault.Error{Kind: fault.Invalid, Message: err.Error(), Cause: err}
	case errors.Is(err, clashapi.ErrProxyNotFound):
		return notRunning(group, err)
	default:
		return unavailable(err)
	}
}

// TestNode URL-tests one outbound through the running sing-box.
func (s *Service) TestNode(ctx context.Context, command TestCommand) (NodeDelay, error) {
	name := strings.TrimSpace(command.Name)
	if name == "" {
		return NodeDelay{}, fault.New(fault.Input, "name is required")
	}
	runtime, err := s.runtime()
	if err != nil {
		return NodeDelay{}, err
	}
	delay, err := runtime.Delay(ctx, name, clashapi.DefaultDelayURL, testTimeout)
	switch {
	case err == nil:
		return NodeDelay{Name: name, Delay: delay}, nil
	case errors.Is(err, clashapi.ErrDelayFailed):
		return NodeDelay{Name: name}, nil
	case errors.Is(err, clashapi.ErrProxyNotFound):
		return NodeDelay{}, notRunning(name, err)
	default:
		return NodeDelay{}, unavailable(err)
	}
}

// TestGroup URL-tests every member of a group.
//
// The members are tested ONE REQUEST EACH from here, not through sing-box's
// own `/group/{name}/delay`. That endpoint shares a single deadline across the
// whole group while running ten at a time, so every dead node in an early wave
// eats the budget of the nodes queued behind it: measured on a 94-member
// group, 51 members answered inside 5s and 79 inside 8s — the rest were
// reported as timed out without ever having been dialled. A per-member
// deadline costs a longer total (bounded by groupTestCeiling) and reports only
// what was actually measured.
//
// Members that are themselves groups are skipped: their latency is that of
// the node they resolve to, which the caller's refetch supplies.
func (s *Service) TestGroup(ctx context.Context, command TestCommand) (GroupDelays, error) {
	name := strings.TrimSpace(command.Name)
	if name == "" {
		return GroupDelays{}, fault.New(fault.Input, "name is required")
	}
	runtime, err := s.runtime()
	if err != nil {
		return GroupDelays{}, err
	}
	list, err := runtime.Proxies(ctx)
	if err != nil {
		return GroupDelays{}, unavailable(err)
	}
	nodes, found := groupNodes(list, name)
	if !found {
		return GroupDelays{}, notRunning(name, clashapi.ErrProxyNotFound)
	}

	ctx, cancel := context.WithTimeout(ctx, groupTestCeiling)
	defer cancel()

	var (
		mutex   sync.Mutex
		delays  = make(map[string]int, len(nodes))
		failure error
	)
	queue := make(chan string)
	var workers sync.WaitGroup
	for range min(groupTestConcurrency, len(nodes)) {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for node := range queue {
				delay, err := runtime.Delay(ctx, node, clashapi.DefaultDelayURL, testTimeout)
				mutex.Lock()
				switch {
				case err == nil:
					delays[node] = delay
				case errors.Is(err, clashapi.ErrDelayFailed), errors.Is(err, clashapi.ErrProxyNotFound):
					// Down, or gone from the running config: absent from the
					// result, which is how "no answer" is reported.
				default:
					// sing-box itself cannot be asked. Remember the first such
					// error and stop: recording every node as down because the
					// PANEL lost the controller would blame the provider.
					if failure == nil {
						failure = err
						cancel()
					}
				}
				mutex.Unlock()
			}
		}()
	}
feed:
	for _, node := range nodes {
		select {
		case queue <- node:
		case <-ctx.Done():
			break feed
		}
	}
	close(queue)
	workers.Wait()

	if failure != nil {
		return GroupDelays{}, unavailable(failure)
	}
	return GroupDelays{Group: name, Delays: delays}, nil
}

// groupNodes returns the plain-node members of a group, de-duplicated and in
// member order. The bool is false when no such group is running.
func groupNodes(list []clashapi.Proxy, group string) ([]string, bool) {
	byName := make(map[string]clashapi.Proxy, len(list))
	for _, proxy := range list {
		byName[proxy.Name] = proxy
	}
	target, ok := byName[group]
	if !ok || !target.IsGroup() {
		return nil, false
	}
	seen := make(map[string]bool, len(target.All))
	nodes := make([]string, 0, len(target.All))
	for _, member := range target.All {
		if seen[member] || byName[member].IsGroup() {
			continue
		}
		seen[member] = true
		nodes = append(nodes, member)
	}
	return nodes, true
}

func (s *Service) runtime() (Runtime, error) {
	settings, err := s.settings.GetClashAPISettings()
	if err != nil {
		return nil, err
	}
	runtime, err := s.connect(settings.ExternalController, settings.Secret)
	if err != nil {
		if errors.Is(err, clashapi.ErrDisabled) {
			return nil, fault.New(fault.Unavailable, "the proxies page needs experimental.clash_api.external_controller to be set")
		}
		return nil, err
	}
	return runtime, nil
}

// selectionPersisted reads `experimental.cache_file.enabled`. Any failure reads
// as "not persisted": the notice it drives is a caution, and a caution shown
// needlessly costs less than one withheld.
func (s *Service) selectionPersisted() bool {
	raw, ok, err := s.settings.GetConfigSection("experimental")
	if err != nil || !ok {
		return false
	}
	var experimental struct {
		CacheFile struct {
			Enabled bool `json:"enabled"`
		} `json:"cache_file"`
	}
	if err := json.Unmarshal(raw, &experimental); err != nil {
		return false
	}
	return experimental.CacheFile.Enabled
}

func unavailable(err error) error {
	message := err.Error()
	if !errors.Is(err, clashapi.ErrUnauthorized) {
		message = "sing-box is not reachable: " + message
	}
	return &fault.Error{Kind: fault.Unavailable, Message: message, Cause: err}
}

// notRunning explains a 404: the tag exists in the page the operator is
// looking at but not in the process, which means the config changed underneath.
func notRunning(name string, err error) error {
	return &fault.Error{
		Kind:    fault.Missing,
		Message: "\"" + name + "\" is not in the running sing-box; reload the page, or restart sing-box to apply the config",
		Cause:   err,
	}
}
