package database

import (
	"fmt"
	"log/slog"

	"github.com/glebarez/sqlite"
	"github.com/jacinli/sky-guardwall/internal/config"
	"github.com/jacinli/sky-guardwall/internal/model"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func Init(cfg *config.Config) *gorm.DB {
	var dialector gorm.Dialector

	switch cfg.DBType {
	case "mysql":
		dialector = mysql.Open(cfg.DBDSN)
	case "postgres":
		dialector = postgres.Open(cfg.DBDSN)
	default:
		slog.Info("using SQLite database", "path", cfg.DBDSN)
		dialector = sqlite.Open(cfg.DBDSN)
	}

	db, err := gorm.Open(dialector, &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		panic(fmt.Sprintf("failed to open database: %v", err))
	}

	if err := db.AutoMigrate(
		&model.IptablesRule{},
		&model.ManagedRule{},
		&model.SyncMeta{},
	); err != nil {
		panic(fmt.Sprintf("automigrate failed: %v", err))
	}

	slog.Info("database ready")
	return db
}
