package model

import "time"

// ManagedRule is a firewall rule created by the user through the UI.
// IptablesArgs stores the JSON-encoded arg slice (without -I/-D flag)
// so we can reconstruct both add and delete commands.
type ManagedRule struct {
	ID           uint   `gorm:"primaryKey;autoIncrement" json:"id"`
	Description  string `gorm:"type:varchar(256)" json:"description"`
	Chain        string `gorm:"type:varchar(64);not null" json:"chain"`
	SrcIP        string `gorm:"type:varchar(128)" json:"src_ip"`
	Protocol     string `gorm:"type:varchar(16)" json:"protocol"`
	DstPort      int    `json:"dst_port"`
	Target       string `gorm:"type:varchar(16);not null" json:"target"`
	IptablesArgs string `gorm:"type:text" json:"iptables_args"` // JSON []string
	IsApplied    bool   `gorm:"default:false" json:"is_applied"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
