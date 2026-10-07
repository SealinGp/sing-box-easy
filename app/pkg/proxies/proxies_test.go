package proxies

import (
	"context"
	"encoding/json"
	"errors"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/SealinGp/sing-box-easy/app/pkg/fault"
	"github.com/SealinGp/sing-box-easy/app/pkg/singbox/clashapi"
	"github.com/SealinGp/sing-box-easy/app/pkg/singbox/config"
)

func sample() []clashapi.Proxy {
	return []clashapi.Proxy{
		{Name: "Media", Type: "Selector", Now: "流媒体", All: []string{"流媒体", "hk-01", "ghost"}, Delay: 999},
		{Name: "流媒体", Type: "URLTest", Now: "sg-01", All: []string{"sg-01", "hk-01"}},
		{Name: "backup", Type: "Fallback", Now: "", All: []string{}},
		{Name: "sg-01", Type: "VLESS", Delay: 74},
		{Name: "hk-01", Type: "AnyTLS", Delay: 0},
		{Name: "direct", Type: "Direct"},
	}
}

func TestBuildViewListsOnlyGroupsInOrder(t *testing.T) {
	view := BuildView(sample(), true)
	if len(view.Groups) != 3 {
		t.Fatalf("groups = %d, want 3", len(view.Groups))
	}
	if view.Groups[0].Name != "Media" || view.Groups[1].Name != "流媒体" || view.Groups[2].Name != "backup" {
		t.Errorf("order = %s, %s, %s", view.Groups[0].Name, view.Groups[1].Name, view.Groups[2].Name)
	}
	if !view.SelectionPersisted {
		t.Error("selection_persisted was dropped")
	}
}

func TestBuildViewOnlySelectorsAreSwitchable(t *testing.T) {
	view := BuildView(sample(), false)
	if !view.Groups[0].Switchable {
		t.Error("a Selector must be switchable")
	}
	if view.Groups[1].Switchable || view.Groups[2].Switchable {
		t.Error("URLTest and Fallback groups must not be switchable")
	}
}

// A selector's own history describes whichever member was probed through it
// last. The delay shown must be the node it resolves to NOW.
func TestBuildViewResolvesDelayThroughNestedGroups(t *testing.T) {
	media := BuildView(sample(), false).Groups[0]
	if media.Delay != 74 {
		t.Errorf("group delay = %d, want 74 (sg-01 via 流媒体), not the selector's own 999", media.Delay)
	}
	if media.Members[0].Delay != 74 || !media.Members[0].Group {
		t.Errorf("nested group member = %+v", media.Members[0])
	}
	if media.Members[1].Delay != 0 || media.Members[1].Type != "AnyTLS" {
		t.Errorf("untested node member = %+v", media.Members[1])
	}
}

func TestBuildViewKeepsMembersMissingFromTheRunningConfig(t *testing.T) {
	media := BuildView(sample(), false).Groups[0]
	if len(media.Members) != 3 || media.Members[2].Name != "ghost" || media.Members[2].Type != "" {
		t.Errorf("members = %+v", media.Members)
	}
}

func TestBuildViewSurvivesAGroupCycle(t *testing.T) {
	view := BuildView([]clashapi.Proxy{
		{Name: "a", Type: "Selector", Now: "b", All: []string{"b"}},
		{Name: "b", Type: "Selector", Now: "a", All: []string{"a"}},
	}, false)
	if view.Groups[0].Delay != 0 {
		t.Errorf("delay = %d", view.Groups[0].Delay)
	}
}

func TestBuildViewEmptyListIsAnEmptyArrayNotNull(t *testing.T) {
	encoded, err := json.Marshal(BuildView(nil, false))
	if err != nil {
		t.Fatal(err)
	}
	if string(encoded) != `{"groups":[],"selection_persisted":false}` {
		t.Errorf("json = %s", encoded)
	}
}

/* ── Service ────────────────────────────────────────────────────────────── */

type fakeSettings struct {
	experimental string
}

func (f fakeSettings) GetClashAPISettings() (config.ClashAPISettings, error) {
	return config.ClashAPISettings{ExternalController: "127.0.0.1:9090"}, nil
}

func (f fakeSettings) GetConfigSection(string) (json.RawMessage, bool, error) {
	if f.experimental == "" {
		return nil, false, nil
	}
	return json.RawMessage(f.experimental), true, nil
}

type fakeRuntime struct {
	list      []clashapi.Proxy
	err       error
	selected  [2]string
	delay     int
	connected bool

	mutex sync.Mutex
	// perNode overrides delay/err for the named outbound.
	perNode map[string]error
	tested  []string
}

func (f *fakeRuntime) Proxies(context.Context) ([]clashapi.Proxy, error) { return f.list, f.err }
func (f *fakeRuntime) SelectProxy(_ context.Context, group, name string) error {
	f.selected = [2]string{group, name}
	return f.err
}
func (f *fakeRuntime) Delay(_ context.Context, tag, _ string, _ time.Duration) (int, error) {
	f.mutex.Lock()
	defer f.mutex.Unlock()
	f.tested = append(f.tested, tag)
	if err, ok := f.perNode[tag]; ok {
		return 0, err
	}
	return f.delay, f.err
}

func newService(runtime *fakeRuntime, experimental string) *Service {
	service := NewService(fakeSettings{experimental: experimental})
	service.connect = func(string, string) (Runtime, error) {
		runtime.connected = true
		return runtime, nil
	}
	return service
}

func faultKind(t *testing.T, err error) fault.Kind {
	t.Helper()
	var e *fault.Error
	if !errors.As(err, &e) {
		t.Fatalf("err = %v, want a *fault.Error", err)
	}
	return e.Kind
}

func TestViewReportsCacheFile(t *testing.T) {
	runtime := &fakeRuntime{list: sample()}
	on, err := newService(runtime, `{"cache_file":{"enabled":true}}`).View(context.Background())
	if err != nil || !on.SelectionPersisted {
		t.Fatalf("view = %+v, err = %v", on, err)
	}
	off, _ := newService(runtime, `{"clash_api":{}}`).View(context.Background())
	missing, _ := newService(runtime, ``).View(context.Background())
	if off.SelectionPersisted || missing.SelectionPersisted {
		t.Error("cache_file absent or disabled must read as not persisted")
	}
}

func TestViewUnreachableIsUnavailable(t *testing.T) {
	_, err := newService(&fakeRuntime{err: errors.New("connection refused")}, ``).View(context.Background())
	if faultKind(t, err) != fault.Unavailable {
		t.Errorf("err = %v", err)
	}
}

func TestSelectValidatesAndTrims(t *testing.T) {
	runtime := &fakeRuntime{}
	service := newService(runtime, ``)
	if err := service.Select(context.Background(), SelectCommand{Group: " ", Name: "x"}); faultKind(t, err) != fault.Input {
		t.Errorf("blank group: %v", err)
	}
	if runtime.connected {
		t.Error("an invalid command must not reach sing-box")
	}
	if err := service.Select(context.Background(), SelectCommand{Group: " Media ", Name: " hk-01 "}); err != nil {
		t.Fatal(err)
	}
	if runtime.selected != [2]string{"Media", "hk-01"} {
		t.Errorf("selected = %v", runtime.selected)
	}
}

func TestSelectMapsSingBoxRefusals(t *testing.T) {
	rejected := newService(&fakeRuntime{err: clashapi.ErrSelectRejected}, ``).Select(context.Background(), SelectCommand{Group: "g", Name: "n"})
	if faultKind(t, rejected) != fault.Invalid {
		t.Errorf("rejected: %v", rejected)
	}
	missing := newService(&fakeRuntime{err: clashapi.ErrProxyNotFound}, ``).Select(context.Background(), SelectCommand{Group: "g", Name: "n"})
	if faultKind(t, missing) != fault.Missing {
		t.Errorf("missing: %v", missing)
	}
}

// A node that fails its test is DOWN, which is an answer; only "sing-box does
// not have this outbound" and "sing-box cannot be asked" are errors.
func TestTestNodeFailureIsAResult(t *testing.T) {
	result, err := newService(&fakeRuntime{err: clashapi.ErrDelayFailed}, ``).TestNode(context.Background(), TestCommand{Name: "hk-01"})
	if err != nil || result.Delay != 0 || result.Name != "hk-01" {
		t.Fatalf("result = %+v, err = %v", result, err)
	}
	ok, err := newService(&fakeRuntime{delay: 88}, ``).TestNode(context.Background(), TestCommand{Name: "hk-01"})
	if err != nil || ok.Delay != 88 {
		t.Fatalf("result = %+v, err = %v", ok, err)
	}
	_, err = newService(&fakeRuntime{err: clashapi.ErrProxyNotFound}, ``).TestNode(context.Background(), TestCommand{Name: "hk-01"})
	if faultKind(t, err) != fault.Missing {
		t.Errorf("not found: %v", err)
	}
}

func TestTestGroupTestsEachNodeMemberOnce(t *testing.T) {
	runtime := &fakeRuntime{delay: 60, list: []clashapi.Proxy{
		{Name: "Media", Type: "Selector", Now: "inner", All: []string{"inner", "a", "b", "a"}},
		{Name: "inner", Type: "URLTest", Now: "c", All: []string{"c"}},
		{Name: "a", Type: "VLESS"}, {Name: "b", Type: "VLESS"}, {Name: "c", Type: "VLESS"},
	}}
	result, err := newService(runtime, ``).TestGroup(context.Background(), TestCommand{Name: "Media"})
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(runtime.tested)
	// The nested group is skipped and the duplicate member is tested once.
	if strings.Join(runtime.tested, ",") != "a,b" {
		t.Errorf("tested = %v", runtime.tested)
	}
	if len(result.Delays) != 2 || result.Delays["a"] != 60 || result.Group != "Media" {
		t.Errorf("result = %+v", result)
	}
}

// A dead member must cost only itself: the others are still reported.
func TestTestGroupDeadMembersAreAbsentNotFatal(t *testing.T) {
	runtime := &fakeRuntime{delay: 40, perNode: map[string]error{
		"dead": clashapi.ErrDelayFailed, "gone": clashapi.ErrProxyNotFound,
	}, list: []clashapi.Proxy{
		{Name: "auto", Type: "URLTest", All: []string{"dead", "ok", "gone"}},
		{Name: "dead", Type: "VLESS"}, {Name: "ok", Type: "VLESS"},
	}}
	result, err := newService(runtime, ``).TestGroup(context.Background(), TestCommand{Name: "auto"})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Delays) != 1 || result.Delays["ok"] != 40 {
		t.Errorf("delays = %v", result.Delays)
	}
}

func TestTestGroupEveryMemberDeadIsAnEmptyMapNotNull(t *testing.T) {
	runtime := &fakeRuntime{perNode: map[string]error{"a": clashapi.ErrDelayFailed}, list: []clashapi.Proxy{
		{Name: "auto", Type: "URLTest", All: []string{"a"}}, {Name: "a", Type: "VLESS"},
	}}
	result, err := newService(runtime, ``).TestGroup(context.Background(), TestCommand{Name: "auto"})
	if err != nil || result.Delays == nil || len(result.Delays) != 0 {
		t.Fatalf("result = %+v, err = %v", result, err)
	}
}

// Losing the controller mid-test is the PANEL's failure. Reporting it as a
// group of dead nodes would blame the provider for it.
func TestTestGroupControllerFailureIsAnErrorNotDeadNodes(t *testing.T) {
	runtime := &fakeRuntime{perNode: map[string]error{"a": errors.New("connection refused")}, list: []clashapi.Proxy{
		{Name: "auto", Type: "URLTest", All: []string{"a"}}, {Name: "a", Type: "VLESS"},
	}}
	_, err := newService(runtime, ``).TestGroup(context.Background(), TestCommand{Name: "auto"})
	if faultKind(t, err) != fault.Unavailable {
		t.Errorf("err = %v", err)
	}
}

func TestTestGroupUnknownOrNonGroupIsMissing(t *testing.T) {
	runtime := &fakeRuntime{list: []clashapi.Proxy{{Name: "a", Type: "VLESS"}}}
	for _, name := range []string{"nope", "a"} {
		_, err := newService(runtime, ``).TestGroup(context.Background(), TestCommand{Name: name})
		if faultKind(t, err) != fault.Missing {
			t.Errorf("%s: err = %v", name, err)
		}
	}
	if len(runtime.tested) != 0 {
		t.Errorf("nothing should have been dialled: %v", runtime.tested)
	}
}
