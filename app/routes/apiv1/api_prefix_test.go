package apiv1

import (
	"context"
	"testing"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/common/ut"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
)

// TestEveryPrefixServesTheSameRoutes is the regression this migration needs.
//
// The previous state of this file mirrored five endpoints onto /api/v1 by hand
// and left 143 behind. Nothing failed, because nothing compared the two — a
// client on the new prefix simply got a 404 for most of the API. This asserts
// that a route registered through the mirror reaches every prefix.
func TestEveryPrefixServesTheSameRoutes(t *testing.T) {
	engine := server.New(server.WithHostPorts("127.0.0.1:0"))

	reached := map[string]int{}
	v1 := newMirror(engine)
	v1.GET("/probe-target", func(_ context.Context, c *app.RequestContext) {
		c.String(consts.StatusOK, "ok")
	})

	for _, prefix := range apiPrefixes {
		response := ut.PerformRequest(engine.Engine, "GET", prefix+"/probe-target", nil)
		if code := response.Result().StatusCode(); code != consts.StatusOK {
			t.Errorf("%s/probe-target returned %d, want 200", prefix, code)
		}
		reached[prefix]++
	}

	if len(reached) != len(apiPrefixes) {
		t.Fatalf("reached %d prefixes, want %d", len(reached), len(apiPrefixes))
	}
}

// TestStablePrefixIsFirst pins the ordering, because the first entry is the
// one the frontend and the docs are expected to use. A reordering that put the
// legacy path first would quietly make it the canonical one again.
func TestStablePrefixIsFirst(t *testing.T) {
	if len(apiPrefixes) == 0 || apiPrefixes[0] != "/api/v1" {
		t.Fatalf("apiPrefixes = %v, want /api/v1 first", apiPrefixes)
	}
}

// TestLegacyPrefixIsStillServed states the compatibility promise as a test, so
// removing it is a deliberate act rather than a cleanup.
func TestLegacyPrefixIsStillServed(t *testing.T) {
	found := false
	for _, prefix := range apiPrefixes {
		if prefix == "/api/1.12.12" {
			found = true
		}
	}
	if !found {
		t.Fatal("the legacy /api/1.12.12 prefix must stay served: operator bookmarks, scripts and LuCI point at it")
	}
}

