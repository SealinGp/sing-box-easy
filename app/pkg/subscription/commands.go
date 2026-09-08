package subscription

import (
	"crypto/rand"
	"encoding/hex"
	"github.com/SealinGp/sing-box-easy/app/pkg/fault"
)

// EditCommand preserves absent fields so updates cannot silently reset values.
type EditCommand struct {
	ID             string  `json:"id"`
	Name           *string `json:"name"`
	URL            *string `json:"url"`
	AutoUpdate     *bool   `json:"auto_update"`
	UpdateInterval *string `json:"update_interval"`
	FetchMode      *string `json:"fetch_mode"`
	ProxyURL       *string `json:"proxy_url"`
	OfficialURL    *string `json:"official_url"`
	ProbeEnabled   *bool   `json:"probe_enabled"`
	ProbeURL       *string `json:"probe_url"`
}

func (c EditCommand) apply(s *Subscription) error {
	if c.Name != nil {
		s.Name = *c.Name
	}
	if c.URL != nil {
		s.URL = *c.URL
	}
	if c.AutoUpdate != nil {
		s.AutoUpdate = *c.AutoUpdate
	}
	if c.UpdateInterval != nil {
		s.UpdateInterval = *c.UpdateInterval
	}
	if c.FetchMode != nil {
		s.FetchMode = *c.FetchMode
	}
	if c.ProxyURL != nil {
		s.ProxyURL = *c.ProxyURL
	}
	if c.OfficialURL != nil {
		s.OfficialURL = *c.OfficialURL
	}
	if c.ProbeURL != nil {
		s.ProbeURL = *c.ProbeURL
	}
	if c.ProbeEnabled != nil {
		s.ProbeEnabled = *c.ProbeEnabled
	}
	if s.Name == "" || s.URL == "" {
		return fault.New(fault.Input, "name and url are required")
	}
	if s.OfficialURL != "" {
		s.OfficialURL = NormalizeOfficialURL(s.OfficialURL)
		if s.OfficialURL == "" {
			return fault.New(fault.Input, "official_url must be an http(s) link")
		}
	}
	var err error
	s.ProbeURL, err = NormalizeProbeURL(s.ProbeURL)
	if err != nil {
		return fault.New(fault.Input, err.Error())
	}
	return nil
}
func (s *Service) Create(c EditCommand) (*Subscription, error) {
	id := c.ID
	if id == "" {
		var b [16]byte
		if _, err := rand.Read(b[:]); err != nil {
			return nil, err
		}
		id = "sub_" + hex.EncodeToString(b[:])
	}
	sub := &Subscription{ID: id, ProbeEnabled: true, UpdateInterval: "24h"}
	if err := c.apply(sub); err != nil {
		return nil, err
	}
	if err := s.add(*sub); err != nil {
		return nil, err
	}
	return s.Get(id)
}
func (s *Service) Edit(id string, c EditCommand) (*Subscription, error) {
	unlock := s.lockSubscription(id)
	defer unlock()
	if err := s.checkActive(id); err != nil {
		return nil, err
	}
	sub, err := s.repository.Get(id)
	if err != nil {
		return nil, err
	}
	if err := c.apply(sub); err != nil {
		return nil, err
	}
	if err := s.repository.Update(id, *sub); err != nil {
		return nil, err
	}
	return s.repository.Get(id)
}
