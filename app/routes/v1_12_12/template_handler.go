package v1_13_0

import (
	"context"

	"github.com/SealinGp/sing-box-easy/app/pkg/config"
	"github.com/cloudwego/hertz/pkg/app"
	C "github.com/sagernet/sing-box/constant"
	"github.com/sagernet/sing-box/option"
)

// createRemoteRuleSet creates a remote rule set with proper structure
func createRemoteRuleSet(tag, url, detour string) config.RuleSet {
	return config.RuleSet{
		Type:   C.RuleSetTypeRemote,
		Tag:    tag,
		Format: C.RuleSetFormatBinary,
		RemoteOptions: option.RemoteRuleSet{
			URL:            url,
			DownloadDetour: detour,
		},
	}
}

// GetDefaultRuleSets returns default rule sets template
func (h *Handler) GetDefaultRuleSets(ctx context.Context, c *app.RequestContext) {
	ruleSets := []config.RuleSet{
		createRemoteRuleSet(
			"geosite-cn",
			"https://gh-proxy.com/https://raw.githubusercontent.com/MetaCubeX/meta-rules-dat/sing/geo/geosite/cn.srs",
			"➡️ 直连",
		),
		createRemoteRuleSet(
			"geosite-private",
			"https://gh-proxy.com/https://raw.githubusercontent.com/MetaCubeX/meta-rules-dat/sing/geo/geosite/private.srs",
			"➡️ 直连",
		),
		createRemoteRuleSet(
			"geosite-geolocation-!cn",
			"https://gh-proxy.com/https://raw.githubusercontent.com/MetaCubeX/meta-rules-dat/sing/geo/geosite/geolocation-!cn.srs",
			"➡️ 直连",
		),
		createRemoteRuleSet(
			"geosite-google",
			"https://gh-proxy.com/https://raw.githubusercontent.com/MetaCubeX/meta-rules-dat/sing/geo/geosite/google.srs",
			"➡️ 直连",
		),
		createRemoteRuleSet(
			"geosite-github",
			"https://gh-proxy.com/https://raw.githubusercontent.com/MetaCubeX/meta-rules-dat/sing/geo/geosite/github.srs",
			"➡️ 直连",
		),
		createRemoteRuleSet(
			"geosite-telegram",
			"https://gh-proxy.com/https://raw.githubusercontent.com/MetaCubeX/meta-rules-dat/sing/geo/geosite/telegram.srs",
			"➡️ 直连",
		),
		createRemoteRuleSet(
			"geosite-youtube",
			"https://gh-proxy.com/https://raw.githubusercontent.com/MetaCubeX/meta-rules-dat/sing/geo/geosite/youtube.srs",
			"➡️ 直连",
		),
		createRemoteRuleSet(
			"geosite-netflix",
			"https://gh-proxy.com/https://raw.githubusercontent.com/MetaCubeX/meta-rules-dat/sing/geo/geosite/netflix.srs",
			"➡️ 直连",
		),
		createRemoteRuleSet(
			"geosite-apple",
			"https://gh-proxy.com/https://raw.githubusercontent.com/MetaCubeX/meta-rules-dat/sing/geo/geosite/apple.srs",
			"➡️ 直连",
		),
		createRemoteRuleSet(
			"geosite-microsoft",
			"https://gh-proxy.com/https://raw.githubusercontent.com/MetaCubeX/meta-rules-dat/sing/geo/geosite/microsoft.srs",
			"➡️ 直连",
		),
		createRemoteRuleSet(
			"geosite-tiktok",
			"https://gh-proxy.com/https://raw.githubusercontent.com/MetaCubeX/meta-rules-dat/sing/geo/geosite/tiktok.srs",
			"➡️ 直连",
		),
		createRemoteRuleSet(
			"geoip-cn",
			"https://gh-proxy.com/https://raw.githubusercontent.com/MetaCubeX/meta-rules-dat/sing/geo/geoip/cn.srs",
			"➡️ 直连",
		),
		createRemoteRuleSet(
			"geoip-google",
			"https://gh-proxy.com/https://raw.githubusercontent.com/MetaCubeX/meta-rules-dat/sing/geo/geoip/google.srs",
			"➡️ 直连",
		),
		createRemoteRuleSet(
			"geoip-telegram",
			"https://gh-proxy.com/https://raw.githubusercontent.com/MetaCubeX/meta-rules-dat/sing/geo/geoip/telegram.srs",
			"➡️ 直连",
		),
		createRemoteRuleSet(
			"geoip-netflix",
			"https://gh-proxy.com/https://raw.githubusercontent.com/MetaCubeX/meta-rules-dat/sing/geo/geoip/netflix.srs",
			"➡️ 直连",
		),
		createRemoteRuleSet(
			"geoip-apple",
			"https://gh-proxy.com/https://raw.githubusercontent.com/MetaCubeX/meta-rules-dat/sing/geo-lite/geoip/apple.srs",
			"➡️ 直连",
		),
	}

	respOK(ctx, c, map[string]any{"rule_sets": ruleSets})
}
