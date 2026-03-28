package model

import "time"

// IptablesRule holds a single parsed line from `iptables -S`.
// All rows are replaced on each sync; no foreign keys.
type IptablesRule struct {
	ID       uint   `gorm:"primaryKey;autoIncrement" json:"id"`
	LineType string `gorm:"type:varchar(16);not null" json:"line_type"` // policy / chain / rule
	Chain    string `gorm:"type:varchar(64);not null;index" json:"chain"`
	Target   string `gorm:"type:varchar(64)" json:"target"`
	Protocol string `gorm:"type:varchar(16)" json:"protocol"`
	Source   string `gorm:"type:varchar(128)" json:"source"`
	Dest     string `gorm:"type:varchar(128)" json:"dest"`
	SrcPort  string `gorm:"type:varchar(32)" json:"src_port"`
	DstPort  string `gorm:"type:varchar(32)" json:"dst_port"`
	InIface  string `gorm:"type:varchar(32)" json:"in_iface"`
	OutIface string `gorm:"type:varchar(32)" json:"out_iface"`
	Extra    string `gorm:"type:text" json:"extra"`
	RawLine  string `gorm:"type:text;not null" json:"raw_line"`

	SyncedAt  time.Time `gorm:"not null;index" json:"synced_at"`
	CreatedAt time.Time `json:"created_at"`
}
