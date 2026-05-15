package models

import (
	"time"

	"gorm.io/gorm"
)

type User struct {
	ID           uint           `gorm:"primaryKey" json:"id"`
	Username     string         `gorm:"uniqueIndex;not null" json:"username"`
	PasswordHash string         `gorm:"not null" json:"-"`
	Apps         []App          `gorm:"foreignKey:UserID" json:"apps,omitempty"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
}

type App struct {
	ID           string         `gorm:"primaryKey;type:uuid" json:"id"`
	Name         string         `gorm:"not null" json:"name"`
	Description  string         `json:"description"`
	Icon         string         `json:"icon"`
	UserID       uint           `gorm:"index;not null" json:"user_id"`
	Releases     []Release      `gorm:"foreignKey:AppID" json:"releases,omitempty"`
	MinVersion   string         `gorm:"default:'0.0.0'" json:"min_version"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
}

type Release struct {
	ID          uint           `gorm:"primaryKey" json:"id"`
	AppID       string         `gorm:"index;not null" json:"app_id"`
	Version     string         `gorm:"not null" json:"version"`
	OS          string         `gorm:"not null" json:"os"`
	Arch        string         `gorm:"not null" json:"arch"`
	SHA256      string         `gorm:"not null" json:"sha256"`
	Size        int64          `gorm:"not null" json:"size"`
	ScheduledAt *time.Time      `gorm:"index" json:"scheduled_at"`
	IsActive    bool           `gorm:"default:true" json:"is_active"`
	CreatedAt   time.Time      `json:"created_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

type Analytics struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	AppID        string    `gorm:"index;not null" json:"app_id"`
	ReleaseID    uint      `gorm:"index" json:"release_id"`
	NodeID       string    `gorm:"index" json:"node_id"`
	IP           string    `json:"ip,omitempty"`
	Country      string    `json:"country,omitempty"`
	OS           string    `json:"os"`
	Arch         string    `json:"arch"`
	Version      string    `json:"version"`
	Action       string    `json:"action"` // e.g., "check", "download"
	Status       string    `json:"status"` // e.g., "success", "fail"
	ErrorMessage string    `json:"error_message,omitempty"`
	CreatedAt    time.Time `gorm:"index" json:"created_at"`
	Node         *Node     `gorm:"foreignKey:NodeID,AppID;references:ID,AppID" json:"node,omitempty"`
}

type Node struct {
	ID        string    `gorm:"primaryKey" json:"id"`
	AppID     string    `gorm:"primaryKey" json:"app_id"`
	FirstSeen time.Time `json:"first_seen"`
	LastSeen  time.Time `json:"last_seen"`
	OS        string    `json:"os"`
	Arch      string    `json:"arch"`
	Version   string    `json:"version"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
