package subscription

import (
	"github.com/SealinGp/sing-box-easy/app/pkg/subscription/model"
)

type SubscriptionManager interface {
	List() ([]*Subscription, error)
	Get(string) (*Subscription, error)
	Create(EditCommand) (*Subscription, error)
	Edit(string, EditCommand) (*Subscription, error)
	Delete(string) error
}

// ServiceRestarter applies a freshly-written sing-box configuration to the
// running service. Keeping this as a one-method interface lets subscription
// refreshes trigger the required lifecycle action without importing the
// concrete service package (which already depends on config).
type ServiceRestarter interface {
	Restart() error
}

const (
	DefaultSubscriptionPath = "/etc/sing-box/subscriptions.json"
)

// Subscription and SubInfo are shared with the persistence layer.
type Subscription = model.Subscription
type SubInfo = model.SubInfo

const (
	FetchModeDirect   = model.FetchModeDirect
	FetchModeCleanDNS = model.FetchModeCleanDNS
	FetchModeProxy    = model.FetchModeProxy
)
