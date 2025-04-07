package config

import (
	"chrononewsapi/internal/entity"
	"context"
	"fmt"
	slogGorm "github.com/orandin/slog-gorm"
	"github.com/spf13/viper"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"log"
	"time"
)

func NewDatabase(config *viper.Viper) *gorm.DB {
	username := config.GetString("db.username")
	password := config.GetString("db.password")
	host := config.GetString("db.host")
	port := config.GetInt("db.port")
	database := config.GetString("db.name")

	dsn := fmt.Sprintf("postgresql://%s:%s@%s:%d/%s?sslmode=disable&lock_timeout=5000", username, password, host, port, database)

	logger := slogGorm.New(
		slogGorm.WithRecordNotFoundError(),
		slogGorm.WithSlowThreshold(500*time.Millisecond),
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger})
	if err != nil {
		log.Fatalln(err)
	}

	ctx := context.Context(context.Background())
	if err := Migrate(ctx, db); err != nil {
		log.Fatalln(err)
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
            CREATE TYPE user_type AS ENUM ('admin','journalist','user');
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
