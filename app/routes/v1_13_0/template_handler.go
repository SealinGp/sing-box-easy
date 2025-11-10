package v1_13_0

import (
	"context"

	"github.com/SealinGp/sing-box-easy/app/pkg/config"
	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/common/utils"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
)

// GetDefaultRuleSets returns default rule sets template
func (h *Handler) GetDefaultRuleSets(ctx context.Context, c *app.RequestContext) {
	ruleSets := []config.RuleSet{
		{
			Tag:            "geosite-cn",
			Type:           "remote",
			Format:         "binary",
			URL:            "https://gh-proxy.com/https://raw.githubusercontent.com/MetaCubeX/meta-rules-dat/sing/geo/geosite/cn.srs",
			DownloadDetour: "➡️ 直连",
		},
		{
			Tag:            "geosite-private",
			Type:           "remote",
			Format:         "binary",
			URL:            "https://gh-proxy.com/https://raw.githubusercontent.com/MetaCubeX/meta-rules-dat/sing/geo/geosite/private.srs",
			DownloadDetour: "➡️ 直连",
		},
		{
			Tag:            "geosite-geolocation-!cn",
			Type:           "remote",
			Format:         "binary",
			URL:            "https://gh-proxy.com/https://raw.githubusercontent.com/MetaCubeX/meta-rules-dat/sing/geo/geosite/geolocation-!cn.srs",
			DownloadDetour: "➡️ 直连",
		},
		{
			Tag:            "geosite-google",
			Type:           "remote",
			Format:         "binary",
			URL:            "https://gh-proxy.com/https://raw.githubusercontent.com/MetaCubeX/meta-rules-dat/sing/geo/geosite/google.srs",
			DownloadDetour: "➡️ 直连",
		},
		{
			Tag:            "geosite-github",
			Type:           "remote",
			Format:         "binary",
			URL:            "https://gh-proxy.com/https://raw.githubusercontent.com/MetaCubeX/meta-rules-dat/sing/geo/geosite/github.srs",
			DownloadDetour: "➡️ 直连",
		},
		{
			Tag:            "geosite-telegram",
			Type:           "remote",
			Format:         "binary",
			URL:            "https://gh-proxy.com/https://raw.githubusercontent.com/MetaCubeX/meta-rules-dat/sing/geo/geosite/telegram.srs",
			DownloadDetour: "➡️ 直连",
		},
		{
			Tag:            "geosite-youtube",
			Type:           "remote",
			Format:         "binary",
			URL:            "https://gh-proxy.com/https://raw.githubusercontent.com/MetaCubeX/meta-rules-dat/sing/geo/geosite/youtube.srs",
			DownloadDetour: "➡️ 直连",
		},
		{
			Tag:            "geosite-netflix",
			Type:           "remote",
			Format:         "binary",
			URL:            "https://gh-proxy.com/https://raw.githubusercontent.com/MetaCubeX/meta-rules-dat/sing/geo/geosite/netflix.srs",
			DownloadDetour: "➡️ 直连",
		},
		{
			Tag:            "geosite-apple",
			Type:           "remote",
			Format:         "binary",
			URL:            "https://gh-proxy.com/https://raw.githubusercontent.com/MetaCubeX/meta-rules-dat/sing/geo/geosite/apple.srs",
			DownloadDetour: "➡️ 直连",
		},
		{
			Tag:            "geosite-microsoft",
			Type:           "remote",
			Format:         "binary",
			URL:            "https://gh-proxy.com/https://raw.githubusercontent.com/MetaCubeX/meta-rules-dat/sing/geo/geosite/microsoft.srs",
			DownloadDetour: "➡️ 直连",
		},
		{
			Tag:            "geosite-tiktok",
			Type:           "remote",
			Format:         "binary",
			URL:            "https://gh-proxy.com/https://raw.githubusercontent.com/MetaCubeX/meta-rules-dat/sing/geo/geosite/tiktok.srs",
			DownloadDetour: "➡️ 直连",
		},
		{
			Tag:            "geoip-cn",
			Type:           "remote",
			Format:         "binary",
			URL:            "https://gh-proxy.com/https://raw.githubusercontent.com/MetaCubeX/meta-rules-dat/sing/geo/geoip/cn.srs",
			DownloadDetour: "➡️ 直连",
		},
		{
			Tag:            "geoip-google",
			Type:           "remote",
			Format:         "binary",
			URL:            "https://gh-proxy.com/https://raw.githubusercontent.com/MetaCubeX/meta-rules-dat/sing/geo/geoip/google.srs",
			DownloadDetour: "➡️ 直连",
		},
		{
			Tag:            "geoip-telegram",
			Type:           "remote",
			Format:         "binary",
			URL:            "https://gh-proxy.com/https://raw.githubusercontent.com/MetaCubeX/meta-rules-dat/sing/geo/geoip/telegram.srs",
			DownloadDetour: "➡️ 直连",
		},
		{
			Tag:            "geoip-netflix",
			Type:           "remote",
			Format:         "binary",
			URL:            "https://gh-proxy.com/https://raw.githubusercontent.com/MetaCubeX/meta-rules-dat/sing/geo/geoip/netflix.srs",
			DownloadDetour: "➡️ 直连",
		},
		{
			Tag:            "geoip-apple",
			Type:           "remote",
			Format:         "binary",
			URL:            "https://gh-proxy.com/https://raw.githubusercontent.com/MetaCubeX/meta-rules-dat/sing/geo-lite/geoip/apple.srs",
			DownloadDetour: "➡️ 直连",
		},
	}

	c.JSON(consts.StatusOK, utils.H{
		"rule_sets": ruleSets,
	})
}
