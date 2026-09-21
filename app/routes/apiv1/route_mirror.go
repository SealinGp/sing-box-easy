package apiv1

// Registering one route on several prefixes.
//
// The alternative — writing every route twice, once per prefix — is how the
// existing partial migration went wrong: five endpoints were mirrored onto
// `/api/v1` by hand and the other 143 were not, so the two prefixes silently
// disagreed about what the API contained. A route declared once cannot drift.

import (
	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/route"
)

// mirror is a set of router groups that receive the same registrations.
type mirror struct {
	groups []*route.RouterGroup
}

// newMirror opens one group per API prefix, with the supplied middleware.
func newMirror(h *server.Hertz, middleware ...app.HandlerFunc) mirror {
	groups := make([]*route.RouterGroup, 0, len(apiPrefixes))
	for _, prefix := range apiPrefixes {
		groups = append(groups, h.Group(prefix, middleware...))
	}
	return mirror{groups: groups}
}

// with derives a mirror that adds middleware to every prefix, so an
// authenticated or admin-only subtree is declared once.
func (m mirror) with(middleware ...app.HandlerFunc) mirror {
	groups := make([]*route.RouterGroup, 0, len(m.groups))
	for _, group := range m.groups {
		groups = append(groups, group.Group("", middleware...))
	}
	return mirror{groups: groups}
}

func (m mirror) GET(path string, handlers ...app.HandlerFunc) {
	for _, group := range m.groups {
		group.GET(path, handlers...)
	}
}

func (m mirror) POST(path string, handlers ...app.HandlerFunc) {
	for _, group := range m.groups {
		group.POST(path, handlers...)
	}
}

func (m mirror) PUT(path string, handlers ...app.HandlerFunc) {
	for _, group := range m.groups {
		group.PUT(path, handlers...)
	}
}

func (m mirror) DELETE(path string, handlers ...app.HandlerFunc) {
	for _, group := range m.groups {
		group.DELETE(path, handlers...)
	}
}
