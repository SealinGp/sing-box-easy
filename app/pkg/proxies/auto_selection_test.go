package proxies

import (
	"context"
	"errors"
	"github.com/SealinGp/sing-box-easy/app/pkg/autoselect"
	"github.com/SealinGp/sing-box-easy/app/pkg/settings"
	"github.com/SealinGp/sing-box-easy/app/pkg/singbox/clashapi"
	"testing"
	"time"
)

type policyStore struct {
	raw  string
	fail bool
}

func (s *policyStore) Get(string) (string, error) {
	if s.raw == "" {
		return "", settings.ErrSettingNotFound
	}
	return s.raw, nil
}
func (s *policyStore) Set(_, v string) error {
	if s.fail {
		return errors.New("disk full")
	}
	s.raw = v
	return nil
}
func (s *policyStore) GetNodeQuality() settings.NodeQuality { return settings.DefaultNodeQuality() }
func (s *policyStore) GetProbeInterval() time.Duration      { return 10 * time.Minute }
func TestManualSelectionPausesAutoBeforeSwitching(t *testing.T) {
	runtime := &fakeRuntime{list: []clashapi.Proxy{{Name: "g", Type: "Selector", Now: "a", All: []string{"a", "b"}}, {Name: "a", Type: "VLESS"}, {Name: "b", Type: "VLESS"}}}
	s := newService(runtime, "")
	store := &policyStore{}
	if err := s.ConfigureAutoSelection(store); err != nil {
		t.Fatal(err)
	}
	if _, err := s.SetAutoSelection(context.Background(), "g", autoselect.Policy{Enabled: true, IntervalSeconds: 300}); err != nil {
		t.Fatal(err)
	}
	if err := s.Select(context.Background(), SelectCommand{"g", "b"}); err != nil {
		t.Fatal(err)
	}
	if runtime.selected != [2]string{"g", "b"} || s.automatic.Statuses()["g"].Enabled {
		t.Fatal("manual choice did not pause scheduler")
	}
	_, _ = s.SetAutoSelection(context.Background(), "g", autoselect.Policy{Enabled: true, IntervalSeconds: 300})
	store.fail = true
	runtime.selected = [2]string{}
	if err := s.Select(context.Background(), SelectCommand{"g", "a"}); err == nil {
		t.Fatal("expected persistence failure")
	}
	if runtime.selected != [2]string{} {
		t.Fatal("manual switch applied without persisting scheduler pause")
	}
}
