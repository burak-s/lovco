package config

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

var DB *pgxpool.Pool

func InitDB(logger *slog.Logger) {
	err := godotenv.Load()
	if err != nil {
		logger.Error("error loading .env file", "error", err)
		os.Exit(1)
	}

	dsn := fmt.Sprintf(
		"user=%s password=%s host=%s port=%s dbname=%s sslmode=disable",
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_HOST"),
		os.Getenv("DB_PORT"),
		os.Getenv("DB_NAME"),
	)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	dbpool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		logger.Error("Unable to connect to database", "error", err)
		os.Exit(1)
	}

	DB = dbpool
	logger.Info("Connected to the database")
}

func InitTestDB(logger *slog.Logger) {
	err := godotenv.Load()
	if err != nil {
		logger.Error("error loading .env file", "error", err)
	}

	dsn := fmt.Sprintf(
		"user=%s password=%s host=%s port=%s dbname=%s sslmode=disable",
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_HOST_TEST"),
		os.Getenv("DB_PORT_TEST"),
		os.Getenv("DB_NAME_TEST"),
	)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	dbpool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		logger.Error("Unable to connect to database", "error", err)
		os.Exit(1)
	}

	DB = dbpool
	logger.Info("Connected to the database")
}

func CloseDB(logger *slog.Logger) {
	if DB != nil {
		DB.Close()
		logger.Info("Database connection closed")
	}
}
