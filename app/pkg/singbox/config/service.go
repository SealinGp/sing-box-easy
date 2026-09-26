package config

import (
	"context"
	"errors"

	"github.com/SealinGp/sing-box-easy/app/pkg/fault"
	"github.com/SealinGp/sing-box-easy/app/pkg/singbox/core"
)

// Service is the whole-document and version-history surface the HTTP layer
// uses, in the same role sections.Service and outbounds.Service play for their
// parts of the config.
//
// It owns the rules the config handlers used to apply inline — which version
// ids are acceptable, which store failures mean "not found" — and states them
// as fault kinds, so every route maps failures through one table
// (respondOperationError) instead of a hand-written branch per handler.
//
// Validation failures pass through unwrapped: *ValidationError carries the
// stage and the core version the UI shows beside the message.
type Service struct{ manager *Manager }

func NewService(manager *Manager) *Service { return &Service{manager: manager} }

func internal(err error) error {
	return &fault.Error{Kind: fault.Internal, Message: err.Error(), Cause: err}
}

// classifyVersion marks a missing version as Missing and anything else as an
// internal failure, keeping the store's own message.
func classifyVersion(err error) error {
	if errors.Is(err, ErrVersionNotFound) {
		return &fault.Error{Kind: fault.Missing, Message: err.Error(), Cause: err}
	}
	return internal(err)
}

func validVersionID(id int64) error {
	if id <= 0 {
		return fault.New(fault.Input, "invalid version id")
	}
	return nil
}

// Current returns the active document exactly as it is on disk.
func (s *Service) Current() (ConfigDocument, error) {
	document, err := s.manager.GetConfigDocument()
	if err != nil {
		return ConfigDocument{}, internal(err)
	}
	return document, nil
}

// Save validates raw against the installed core and promotes it atomically.
func (s *Service) Save(ctx context.Context, raw []byte) error {
	return s.manager.SaveRawConfig(ctx, raw)
}

// Validate checks raw against the installed core without touching the active file.
func (s *Service) Validate(ctx context.Context, raw []byte) error {
	return s.manager.ValidateRawConfig(ctx, raw)
}

// Backup returns the newest historical version.
func (s *Service) Backup() (ConfigDocument, error) {
	document, err := s.manager.GetBackupDocument()
	if err != nil {
		return ConfigDocument{}, internal(err)
	}
	return document, nil
}

// RestoreLatest restores the newest historical version.
func (s *Service) RestoreLatest() error {
	if err := s.manager.Rollback(); err != nil {
		return internal(err)
	}
	return nil
}

// ListHistory returns version metadata, newest first.
func (s *Service) ListHistory() ([]VersionInfo, error) {
	versions, err := s.manager.ListVersions()
	if err != nil {
		return nil, internal(err)
	}
	return versions, nil
}

// HistoryVersion returns one historical document.
func (s *Service) HistoryVersion(id int64) (ConfigDocument, error) {
	if err := validVersionID(id); err != nil {
		return ConfigDocument{}, err
	}
	document, err := s.manager.GetVersionDocument(id)
	if err != nil {
		return ConfigDocument{}, classifyVersion(err)
	}
	return document, nil
}

// DeleteHistory removes one historical version; the live config is untouched.
func (s *Service) DeleteHistory(id int64) error {
	if err := validVersionID(id); err != nil {
		return err
	}
	if err := s.manager.DeleteVersion(id); err != nil {
		return classifyVersion(err)
	}
	return nil
}

// DeleteHistoryBatch removes several historical versions and reports how many
// were deleted. Every id is checked before anything is removed.
func (s *Service) DeleteHistoryBatch(ids []int64) (int64, error) {
	if len(ids) == 0 {
		return 0, fault.New(fault.Input, "ids array is required and cannot be empty")
	}
	for _, id := range ids {
		if id <= 0 {
			return 0, fault.New(fault.Input, "ids must be positive integers")
		}
	}
	deleted, err := s.manager.DeleteVersions(ids)
	if err != nil {
		return 0, internal(err)
	}
	return deleted, nil
}

// RestoreVersion makes a historical version the live config.
//
// A missing id is reported as an internal failure, not Missing: that is what
// this endpoint has always returned, and changing a response code is an API
// change this refactor does not make.
func (s *Service) RestoreVersion(id int64) error {
	if err := validVersionID(id); err != nil {
		return err
	}
	if err := s.manager.RollbackToVersion(id); err != nil {
		return internal(err)
	}
	return nil
}

// CoreInfo reports the installed core version and its feature gates.
func (s *Service) CoreInfo(ctx context.Context) (core.CoreCapabilities, error) {
	capabilities, err := s.manager.CoreCapabilities(ctx)
	if err != nil {
		return core.CoreCapabilities{}, &fault.Error{Kind: fault.Unavailable, Message: err.Error(), Cause: err}
	}
	return capabilities, nil
}
