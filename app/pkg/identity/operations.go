package user

import (
	"github.com/SealinGp/sing-box-easy/app/pkg/fault"
	"github.com/SealinGp/sing-box-easy/app/pkg/identity/repo"
)

type User = repo.User

// UpdateAs applies resource authorization for every caller, including non-HTTP callers.
func UpdateAs(manager UserManager, actor *User, id int64, username, password, role string) (*User, error) {
	if actor == nil {
		return nil, ErrUnauthorized
	}
	if actor.Role != "admin" {
		if actor.ID != id {
			return nil, fault.New(fault.Forbidden, "You can only update your own profile")
		}
		role = ""
	}
	return manager.UpdateUser(id, username, password, role)
}
func DeleteAs(manager UserManager, actor *User, id int64) error {
	if actor == nil {
		return ErrUnauthorized
	}
	if actor.Role != "admin" {
		return fault.New(fault.Forbidden, "Administrator access is required")
	}
	if actor.ID == id {
		return fault.New(fault.Forbidden, "You cannot delete your own account")
	}
	return manager.DeleteUser(id)
}
