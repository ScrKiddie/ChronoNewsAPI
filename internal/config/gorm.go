package config

import (
	"chrononewsapi/internal/entity"
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	slogGorm "github.com/orandin/slog-gorm"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func NewDatabase(config *Config) *gorm.DB {
	username := config.DB.Username
	password := config.DB.Password
	host := config.DB.Host
	port := config.DB.Port
	database := config.DB.Name

	dsn := fmt.Sprintf("postgresql://%s:%s@%s:%d/%s?sslmode=disable&lock_timeout=5000", username, password, host, port, database)

	logger := slogGorm.New(
		slogGorm.WithRecordNotFoundError(),
		slogGorm.WithSlowThreshold(500*time.Millisecond),
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger})
	if err != nil {
		slog.Error("Failed to connect to database", "err", err)
		os.Exit(1)
	}

	ctx := context.Context(context.Background())
	if err := Migrate(ctx, db); err != nil {
		slog.Error("Failed to migrate database", "err", err)
		os.Exit(1)
	}

	return db
}

func Migrate(ctx context.Context, db *gorm.DB) error {
	tx := db.WithContext(ctx).Begin()
	defer tx.Rollback()

	if err := tx.Exec(`
    DO $$ 
    BEGIN 
        IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'user_type') THEN 
            CREATE TYPE user_type AS ENUM ('admin','journalist');
        END IF;
    END $$;
	`).Error; err != nil {
		return err
	}

	if err := tx.Exec(`
    DO $$
    BEGIN
        IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'file_type') THEN
            CREATE TYPE file_type AS ENUM ('thumbnail','attachment');
        END IF;
    END $$;
	`).Error; err != nil {
		return err
	}

	if err := tx.Exec(`
    DO $$
    BEGIN
        IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'file_status') THEN
            CREATE TYPE file_status AS ENUM ('pending','processing','compressed','failed');
        END IF;
    END $$;
	`).Error; err != nil {
		return err
	}

	entities := []interface{}{
		&entity.User{},
		&entity.Post{},
		&entity.File{},
		&entity.Category{},
		&entity.Reset{},
		&entity.DeadLetterQueue{},
		&entity.SourceFileToDelete{},
	}

	for _, e := range entities {
		if err := tx.AutoMigrate(e); err != nil {
			return err
		}
	}

	if err := tx.Commit().Error; err != nil {
		return err
	}

	return nil
}
