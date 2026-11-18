package subscription

// syncNodeHistory uses the committed config, including on an idempotent retry
// where the latest diff no longer contains the previously deleted tags.
func (s *Service) syncNodeHistory(id string) error {
	if s.nodeHistory == nil {
		return nil
	}
	cfg, err := s.configManager.GetOutboundsConfig()
	if err != nil {
		return err
	}
	var active []string
	for _, outbound := range cfg.Outbounds {
		if TagBelongsToSubscription(outbound.Tag, id) {
			active = append(active, outbound.Tag)
		}
	}
	return s.nodeHistory.SyncSubscription(id, active)
}
