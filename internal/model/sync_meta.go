package model

import "time"

// SyncMeta records each iptables sync attempt.
type SyncMeta struct {
	ID        uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	SyncedAt  time.Time `gorm:"not null" json:"synced_at"`
	RuleCount int       `json:"rule_count"`
	HasError  bool      `json:"has_error"`
	ErrorMsg  string    `gorm:"type:text" json:"error_msg"`
	CreatedAt time.Time `json:"created_at"`
}
