package storage

import (
	"fmt"

	"github.com/glebarez/sqlite"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Open initializes a GORM database connection and runs auto-migration.
func Open(driver, dsn string) (*gorm.DB, error) {
	var dialector gorm.Dialector
	switch driver {
	case "sqlite":
		dialector = sqlite.Open(dsn)
	case "postgres":
		dialector = postgres.Open(dsn)
	case "mysql":
		dialector = mysql.Open(dsn)
	default:
		return nil, fmt.Errorf("unsupported database driver: %s", driver)
	}

	db, err := gorm.Open(dialector, &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	if err := db.AutoMigrate(&Scenario{}, &Prompt{}); err != nil {
		return nil, fmt.Errorf("auto-migrate: %w", err)
	}

	// Scenarios are now hard-deleted; purge any rows soft-deleted by earlier
	// versions so their names stop blocking recreation via the unique index.
	if err := db.Unscoped().Where("deleted_at IS NOT NULL").Delete(&Scenario{}).Error; err != nil {
		return nil, fmt.Errorf("purge soft-deleted scenarios: %w", err)
	}

	return db, nil
}
