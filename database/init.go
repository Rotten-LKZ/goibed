package database

import (
	"goibed/config"
	"log/slog"
	"os"
	"path/filepath"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var DB *gorm.DB

func init() {
	if err := os.MkdirAll(filepath.Join(config.Config.BasePath, "data"), 0755); err != nil {
		slog.Error("failed to create data folder to save datas")
		os.Exit(1)
	}
	if err := os.MkdirAll(filepath.Join(config.Config.BasePath, "data", "uploads"), 0755); err != nil {
		slog.Error("failed to create uploads folder to save datas")
		os.Exit(1)
	}
	db, err := gorm.Open(sqlite.Open("./data/goibed.db"), &gorm.Config{})
	if err != nil {
		slog.Error("failed to connect to the database")
		os.Exit(1)
	}
	db.AutoMigrate(&Images{})
	DB = db
}
