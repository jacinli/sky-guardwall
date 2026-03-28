---
name: database
description: Use when writing GORM model definitions, database initialization, queries, or anything touching internal/model/ or internal/database/. Critical: enforces the no-foreign-key rule.
---

# Database Conventions

## THE ONE RULE: No Foreign Keys

The database must work identically across SQLite, MySQL, and PostgreSQL.
**Never add FK constraints.** This means:

```go
// FORBIDDEN
type PortEntry struct {
    ScanRecord   ScanRecord `gorm:"foreignKey:ScanRecordID"`  // NO
    ScanRecordID uint       `gorm:"constraint:OnDelete:CASCADE"` // NO
}

// CORRECT — plain integer reference, no constraint
type PortEntry struct {
    ScanRecordID uint `gorm:"not null;index"`  // index OK, FK NOT OK
}
```

## Multi-DB Initialization

```go
// internal/database/db.go
func Init(dbType, dsn string) (*gorm.DB, error) {
    var dialector gorm.Dialector
    switch dbType {
    case "mysql":
        dialector = mysql.Open(dsn)
    case "postgres":
        dialector = postgres.Open(dsn)
    default: // "sqlite"
        dialector = sqlite.Open(dsn)  // modernc.org/sqlite — no CGO
    }
    db, err := gorm.Open(dialector, &gorm.Config{})
    if err != nil {
        return nil, fmt.Errorf("db open: %w", err)
    }
    return db, db.AutoMigrate(
        &model.ScanRecord{},
        &model.PortEntry{},
        &model.FirewallRule{},
    )
}
```

Use `modernc.org/sqlite` (pure-Go) as SQLite driver. Enables `CGO_ENABLED=0` in Docker.

## Model Definitions

### ScanRecord

```go
type ScanRecord struct {
    ID          uint      `gorm:"primaryKey;autoIncrement"`
    ScannedAt   time.Time `gorm:"not null"`
    RawSST      string    `gorm:"type:text;column:raw_ss_tcp"`
    RawSSU      string    `gorm:"type:text;column:raw_ss_udp"`
    RawIptables string    `gorm:"type:text"`
    RawNft      string    `gorm:"type:text"`
    CreatedAt   time.Time
}
```

### PortEntry

```go
type PortEntry struct {
    ID            uint      `gorm:"primaryKey;autoIncrement"`
    ScanRecordID  uint      `gorm:"not null;index"`            // NO FK
    Protocol      string    `gorm:"type:varchar(8);not null"`  // tcp/udp
    LocalAddr     string    `gorm:"type:varchar(64);not null"`
    Port          int       `gorm:"not null"`
    ProcessName   string    `gorm:"type:varchar(128)"`
    PID           int
    ExposureLevel string    `gorm:"type:varchar(16);not null"` // public/private/loopback/specific
    SourceType    string    `gorm:"type:varchar(16);not null"` // docker/system/user
    CreatedAt     time.Time
}
```

### FirewallRule

```go
type FirewallRule struct {
    ID        uint      `gorm:"primaryKey;autoIncrement"`
    RuleType  string    `gorm:"type:varchar(16);not null"` // allow/deny
    Direction string    `gorm:"type:varchar(8);not null"`  // in/out
    Protocol  string    `gorm:"type:varchar(8);not null"`  // tcp/udp/all
    SrcIP     string    `gorm:"type:varchar(64)"`          // empty=any
    DstPort   int                                          // 0=any
    Action    string    `gorm:"type:varchar(16);not null"` // ACCEPT/DROP/REJECT
    Chain     string    `gorm:"type:varchar(32);not null"` // INPUT/OUTPUT/...
    Comment   string    `gorm:"type:varchar(256)"`
    IsActive  bool      `gorm:"default:false"`
    CreatedAt time.Time
    UpdatedAt time.Time
}
```

## Query Patterns

```go
// Filter by scan + exposure
db.Where("scan_record_id = ? AND exposure_level = ?", scanID, level).Find(&entries)

// Latest scan
var latest model.ScanRecord
db.Order("scanned_at DESC").First(&latest)

// Paginate
db.Offset((page-1)*size).Limit(size).Find(&rules)

// Toggle active
db.Model(&rule).Update("is_active", true)
```

## Indexes to Add

- `port_entries.scan_record_id` — list by scan
- `port_entries.exposure_level` — filter by exposure
- `firewall_rules.is_active` — list active rules

## Rules

- Never write raw SQL — GORM methods only
- Never add `REFERENCES` or `constraint:` tags
- Use `AutoMigrate` — never hand-write DDL
- No soft delete (`DeletedAt`) — use `IsActive` for firewall rules only
