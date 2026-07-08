package repo

import "time"

// User represents a user in the system
type User struct {
	ID           int64     `xorm:"pk autoincr 'id'" json:"id"`
	Username     string    `xorm:"unique notnull 'username'" json:"username"`
	PasswordHash string    `xorm:"notnull 'password_hash'" json:"-"` // Hidden from JSON responses
	Role         string    `xorm:"notnull default('viewer') 'role'" json:"role"` // "admin" or "viewer"
	CreatedAt    time.Time `xorm:"created 'created_at'" json:"created_at"`
	UpdatedAt    time.Time `xorm:"updated 'updated_at'" json:"updated_at"`
}

// TableName specifies the table name for User
func (User) TableName() string {
	return "users"
}

// UserSession represents a logged-in user session
type UserSession struct {
	Token     string    `xorm:"pk 'token'" json:"token"`
	UserID    int64     `xorm:"index notnull 'user_id'" json:"user_id"`
	ExpiresAt time.Time `xorm:"notnull 'expires_at'" json:"expires_at"`
	CreatedAt time.Time `xorm:"created 'created_at'" json:"created_at"`
}

// TableName specifies the table name for UserSession
func (UserSession) TableName() string {
	return "user_sessions"
}
