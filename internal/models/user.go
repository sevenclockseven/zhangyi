package models

import "time"

// User 用户
type User struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	Username  string    `json:"username" gorm:"uniqueIndex;size:50;not null"`
	Password  string    `json:"-" gorm:"size:100;not null"` // bcrypt hash
	RealName  string    `json:"real_name" gorm:"size:50"`
	Role      string    `json:"role" gorm:"size:20;default:user"` // admin / user
	Status    string    `json:"status" gorm:"size:20;default:active"`
	LastLogin time.Time `json:"last_login"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
