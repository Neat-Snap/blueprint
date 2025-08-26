package db

import (
	"fmt"

	"github.com/Neat-Snap/blueprint-backend/config"
	"github.com/Neat-Snap/blueprint-backend/logger"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func Connect(cfg *config.Config, logger *logger.MultiLogger) (*gorm.DB, error) {
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s", cfg.DBHost, cfg.DBUser, cfg.DBPass, cfg.DBName, cfg.DBPort)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	if err := db.SetupJoinTable(&User{}, "WorkSpaces", &UserWorkspace{}); err != nil {
		logger.Error("failed to setup join table for User and WorkSpaces", "error", err)
		return nil, err
	}
	if err := db.SetupJoinTable(&WorkSpace{}, "Users", &UserWorkspace{}); err != nil {
		logger.Error("failed to setup join table for WorkSpace and Users", "error", err)
		return nil, err
	}

	logger.Info("successfully connected to database", "db_name", cfg.DBName)

	return db, nil
}
