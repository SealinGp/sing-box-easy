package sublink

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/SealinGp/sing-box-easy/app/pkg/logger"
	"github.com/SealinGp/sing-box-easy/app/pkg/sublink/clash"
	"github.com/SealinGp/sing-box-easy/app/pkg/sublink/node"
	"go.uber.org/zap"
)

// parseNonBase64Body handles the two subscription shapes that are not the
// canonical base64 URI list:
//
//  1. a Clash / Mihomo YAML profile — served by Clash-only endpoints and by
//     panels that content-negotiate on the User-Agent;
//  2. a plain-text URI list — the canonical format with the base64 layer
//     simply omitted, which a few panels do.
//
// It is only reached after base64 decoding has failed, so it never changes the
// handling of a well-formed subscription. `source` is used for logging only.
func (l *SubLink) parseNonBase64Body(body []byte, source string) ([]*node.SubNode, error) {
	if clash.Detect(body) {
		nodes, skipped, err := clash.Parse(body)
		if err != nil {
			return nil, err
		}
		// Unsupported proxies are reported, not swallowed: a subscription that
		// silently imports 40 of 50 nodes looks like a bug in the panel.
		if len(skipped) > 0 {
			reasons := make([]string, 0, len(skipped))
			for _, s := range skipped {
				reasons = append(reasons, s.String())
			}
			logger.Warn("clash profile: some proxies were skipped",
				zap.String("url", source),
				zap.Int("imported", len(nodes)),
				zap.Int("skipped", len(skipped)),
				zap.Strings("reasons", reasons))
		}
		logger.Info("imported subscription as a clash profile",
			zap.String("url", source),
			zap.Int("nodes", len(nodes)))
		return nodes, nil
	}

	// Plain-text URI list.
	if strings.Contains(string(body), "://") {
		if nodes := l.parseBody(body); len(nodes) > 0 {
			logger.Info("imported subscription as a plain-text uri list",
				zap.String("url", source),
				zap.Int("nodes", len(nodes)))
			return nodes, nil
		}
	}

	return nil, fmt.Errorf("body is neither base64, a clash profile, nor a uri list")
}

// shouldRetryWithClientUA decides whether a response deserves a second attempt
// under a client-style User-Agent.
//
// Only 4xx qualifies: a UA-gating panel answers 403/404 (some answer 401), all
// of which are indistinguishable from a genuinely bad URL until the retry is
// tried. A 5xx is the server failing at a request it accepted, so repeating it
// with a different UA only doubles the load on an already-struggling panel.
//
// 429 is excluded for the same reason turned around: it is the panel asking to
// be left alone, and retrying it wearing a different client's name is exactly
// the behaviour that gets a subscriber's IP blocked outright.
func shouldRetryWithClientUA(statusCode int) bool {
	if statusCode == http.StatusTooManyRequests {
		return false
	}
	return statusCode >= 400 && statusCode < 500
}
