package v1_13_0

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/SealinGp/sing-box-easy/app/pkg/identity"

	"github.com/cloudwego/hertz/pkg/app"
)

type preferenceManager struct {
	user.UserManager
	calledID int64
}

func (m *preferenceManager) GetPreferences(id int64) (*user.Preferences, error) {
	m.calledID = id
	return &user.Preferences{OverviewOrder: []string{"dns-probe"}}, nil
}
func (m *preferenceManager) UpdatePreferences(id int64, p user.Preferences) (*user.Preferences, error) {
	m.calledID = id
	return &p, nil
}

func TestPreferenceHandlersAccountBoundary(t *testing.T) {
	for _, method := range []string{"get", "put"} {
		for _, id := range []int64{-1, 0, 42} {
			m := &preferenceManager{}
			h := testHandler(&Handler{userManager: m})
			c := app.NewContext(0)
			if id >= 0 {
				c.Set(UserContextKey, &user.User{ID: id, Role: "viewer"})
			}
			c.Request.Header.Set("Content-Type", "application/json")
			// Client-provided identity must never override the authenticated viewer.
			c.Request.SetBody([]byte(`{"id":99,"user_id":99,"overview_order":["dns-probe"]}`))
			if method == "get" {
				h.GetPreferences(context.Background(), c)
			} else {
				h.UpdatePreferences(context.Background(), c)
			}
			var response BasicResponse[any]
			if err := json.Unmarshal(c.Response.Body(), &response); err != nil {
				t.Fatal(err)
			}
			if id == 42 {
				if response.Code != CodeSuccess || m.calledID != 42 {
					t.Fatalf("%s: viewer preference failed: %+v, id %d", method, response, m.calledID)
				}
			} else if response.Code != CodeUnauthorized || m.calledID != 0 {
				t.Fatalf("%s: anonymous request reached storage: %+v", method, response)
			}
		}
	}
}

func TestPreferenceHandlerRejectsMalformedOrder(t *testing.T) {
	for _, body := range []string{`{}`, `null`, `{"overview_order":null}`, `{"overview_order":"dns-probe"}`, `{"overview_order":[3]}`} {
		m := &preferenceManager{}
		h := testHandler(&Handler{userManager: m})
		c := app.NewContext(0)
		c.Set(UserContextKey, &user.User{ID: 42, Role: "viewer"})
		c.Request.Header.Set("Content-Type", "application/json")
		c.Request.SetBody([]byte(body))
		h.UpdatePreferences(context.Background(), c)
		var response BasicResponse[any]
		if err := json.Unmarshal(c.Response.Body(), &response); err != nil {
			t.Fatal(err)
		}
		if response.Code != CodeBadRequest || m.calledID != 0 {
			t.Fatalf("accepted %s: %+v", body, response)
		}
	}
}
