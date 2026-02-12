package migrate

import (
	"fmt"

	"gorm.io/gorm"
)

func init() {
	RegisterBeforeAutoMigration(Migration{
		Version: 3,
		Up:      setDefaultAuthType,
	})
}

// 003: set default auth_type for existing channels
func setDefaultAuthType(db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("db is nil")
	}

	dialect := db.Dialector.Name()

	// Check if auth_type column exists
	hasColumn := func(table, column string) bool {
		switch dialect {
		case "sqlite":
			var name string
			db.Raw("SELECT name FROM pragma_table_info(?) WHERE name = ? LIMIT 1", table, column).Scan(&name)
			return name == column
		case "mysql":
			var count int64
			db.Raw("SELECT COUNT(*) FROM information_schema.columns WHERE table_schema = DATABASE() AND table_name = ? AND column_name = ?", table, column).Scan(&count)
			return count > 0
		case "postgres":
			var count int64
			db.Raw("SELECT COUNT(*) FROM information_schema.columns WHERE table_name = ? AND column_name = ?", table, column).Scan(&count)
			return count > 0
		default:
			return db.Migrator().HasColumn(table, column)
		}
	}

	// Only set default if column exists (after AutoMigrate creates it)
	if hasColumn("channels", "auth_type") {
		// Update any NULL or empty auth_type to 'api_key'
		result := db.Exec("UPDATE channels SET auth_type = 'api_key' WHERE auth_type IS NULL OR auth_type = ''")
		if result.Error != nil {
			return fmt.Errorf("failed to set default auth_type: %w", result.Error)
		}
	}

	return nil
}
