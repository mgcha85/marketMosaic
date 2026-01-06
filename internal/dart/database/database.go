package database

import (
	"log"
	"os"
	"path/filepath"

	"dx-unified/internal/dart/models"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

func InitDB(dbPath string) error {
	if dbPath == "" {
		dbPath = "./data/dart.db"
	}

	// Ensure directory exists
	dir := filepath.Dir(dbPath)
	if dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}

	var err error
	DB, err = gorm.Open(sqlite.Open(dbPath), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		return err
	}

	log.Println("[DART] Database connected successfully")

	// Auto Migrate
	err = DB.AutoMigrate(
		&models.Corp{},
		&models.Filing{},
		&models.FilingDocument{},
		&models.ExtractedEvent{},
	)
	if err != nil {
		return err
	}

	log.Println("[DART] Database migration completed")
	return nil
}

func GetDB() *gorm.DB {
	return DB
}
