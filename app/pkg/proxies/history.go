package proxies

import (
	"strings"
	"time"

	"github.com/SealinGp/sing-box-easy/app/pkg/fault"
	"github.com/SealinGp/sing-box-easy/app/pkg/nodehistory"
	"github.com/SealinGp/sing-box-easy/app/pkg/singbox/clashapi"
)

func nodeMeasurement(name string, delay int, success bool, source string) nodehistory.Measurement {
	return nodehistory.Measurement{Tag: name, SubID: nodehistory.SubscriptionID(name), TestURL: clashapi.DefaultDelayURL, At: time.Now(), Delay: delay, Success: success, Source: source}
}

func (s *Service) recordNode(name string, delay int, success bool, source string) error {
	if s.history == nil {
		return nil
	}
	if err := s.history.RecordBatch([]nodehistory.Measurement{nodeMeasurement(name, delay, success, source)}); err != nil {
		return fault.New(fault.Failed, "latency measured but could not save node history: "+err.Error())
	}
	return nil
}

func (s *Service) attachHistory(view *View) error {
	if s.history == nil {
		return nil
	}
	seen := make(map[string]bool)
	var tags []string
	for _, group := range view.Groups {
		for _, member := range group.Members {
			if !member.Group && !seen[member.Name] {
				seen[member.Name] = true
				tags = append(tags, member.Name)
			}
		}
	}
	// Quality criteria allow at most 100 observations; full charts load lazily.
	recent, err := s.history.Recent(tags, 100)
	if err != nil {
		return err
	}
	for i := range view.Groups {
		for j := range view.Groups[i].Members {
			member := &view.Groups[i].Members[j]
			member.History = recent[member.Name]
		}
	}
	return nil
}

func (s *Service) NodeHistory(name string) ([]nodehistory.Point, error) {
	if strings.TrimSpace(name) == "" {
		return nil, fault.New(fault.Input, "name is required")
	}
	if s.history == nil {
		return nil, fault.New(fault.Unavailable, "node history is not configured")
	}
	points, err := s.history.Series(name)
	if err != nil {
		return nil, fault.New(fault.Failed, err.Error())
	}
	return points, nil
}
