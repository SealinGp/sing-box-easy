package repo

import "github.com/SealinGp/sing-box-easy/app/pkg/subscription/model"

// Repository persists subscription records. Delete only removes the database
// row; callers must use the subscription business service for node cleanup.
type Repository interface {
	Init() error
	List() ([]*model.Subscription, error)
	Get(id string) (*model.Subscription, error)
	Add(model.Subscription) error
	Update(string, model.Subscription) error
	Delete(string) error
	UpdateLastUpdate(string) error
	UpdateInfo(string, []model.SubInfo) error
	UpdateOfficialURL(string, string) error
}

var _ Repository = (*Store)(nil)
