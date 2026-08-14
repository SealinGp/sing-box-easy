package settings

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"unicode"

	"github.com/SealinGp/sing-box-easy/app/pkg/logger"
	"go.uber.org/zap"
)

// Bounds for the subscription info-keyword override. They exist to keep one bad
// PUT from bloating the settings row or slowing every node match down; real
// lists are a couple of dozen short words.
const (
	MaxInfoKeywords   = 200
	MaxInfoKeywordLen = 64
)

// GetSubscriptionInfoKeywords returns the operator's override list, or nil when
// none is stored. Callers resolve nil to their own defaults (see
// subscription.EffectiveInfoKeywords) — this layer only owns persistence.
//
// A corrupt value is logged and treated as "unset" rather than propagated: a bad
// row must never stop subscriptions from updating.
func (m *ManagerXORM) GetSubscriptionInfoKeywords() []string {
	raw, err := m.Get(KeySubscriptionInfoKeywords)
	if err != nil {
		if !errors.Is(err, ErrSettingNotFound) {
			logger.Warn("failed to read subscription_info_keywords, using defaults", zap.Error(err))
		}
		return nil
	}
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}

	var out []string
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		logger.Warn("subscription_info_keywords is not a JSON array, using defaults",
			zap.String("value", raw), zap.Error(err))
		return nil
	}
	return out
}

// SetSubscriptionInfoKeywords validates and stores the override. An empty list
// clears it, which restores the built-in defaults. Callers are expected to have
// normalized (trim/lowercase/dedupe) the list first; validation here is the
// boundary check that protects the database.
func (m *ManagerXORM) SetSubscriptionInfoKeywords(keywords []string) error {
	if err := validateInfoKeywords(keywords); err != nil {
		return err
	}
	if len(keywords) == 0 {
		return m.Set(KeySubscriptionInfoKeywords, "")
	}
	encoded, err := json.Marshal(keywords)
	if err != nil {
		return fmt.Errorf("failed to encode subscription_info_keywords: %w", err)
	}
	return m.Set(KeySubscriptionInfoKeywords, string(encoded))
}

// validateInfoKeywords rejects lists that are too long, entries that are empty,
// oversized, or carry control characters (which can only come from a mangled
// paste and would never match a display name anyway).
func validateInfoKeywords(keywords []string) error {
	if len(keywords) > MaxInfoKeywords {
		return fmt.Errorf("at most %d keywords are allowed, got %d", MaxInfoKeywords, len(keywords))
	}
	for _, kw := range keywords {
		if strings.TrimSpace(kw) == "" {
			return errors.New("keywords must not be empty")
		}
		if len([]rune(kw)) > MaxInfoKeywordLen {
			return fmt.Errorf("keyword %q exceeds %d characters", kw, MaxInfoKeywordLen)
		}
		for _, r := range kw {
			if unicode.IsControl(r) {
				return fmt.Errorf("keyword %q contains control characters", kw)
			}
		}
	}
	return nil
}
