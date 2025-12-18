package db

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func Connect() *pgxpool.Pool {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("DATABASE_URL environment variable not set")
	}

	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		log.Fatal("Failed to parse DB config:", err)
	}

	config.MaxConns = int32(getEnvAsInt("DB_MAX_CONNS", 10))
	config.MinConns = int32(getEnvAsInt("DB_MIN_CONNS", 2))
	idleTime := getEnvAsDuration("DB_MAX_CONN_IDLE_TIME", "5m")
	config.MaxConnIdleTime = idleTime

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	DB, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		log.Fatal("Unable to connect to database:", err)
	}

	if err := DB.Ping(ctx); err != nil {
		log.Fatal("Database ping failed:", err)
	}

	log.Println("Connected to PostgreSQL")

	// Apply schema on startup unless explicitly disabled
	if !strings.EqualFold(os.Getenv("APPLY_SCHEMA_ON_START"), "false") {
		schemaCtx, cancelSchema := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancelSchema()
		if err := ApplySchema(schemaCtx, DB); err != nil {
			log.Fatal("Failed to apply schema:", err)
		}
	}

	return DB
}

func getEnvAsInt(key string, defaultValue int) int {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}
	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return defaultValue
	}
	return value
}

func getEnvAsDuration(key, defaultValue string) time.Duration {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		valueStr = defaultValue
	}
	duration, err := time.ParseDuration(valueStr)
	if err != nil {
		log.Printf("Invalid duration for %s, using default: %s", key, defaultValue)
		duration, _ = time.ParseDuration(defaultValue)
	}
	return duration
}

// ApplySchema reads the SQL schema file and executes it against the provided pool.
// Default schema path: pkg/db/schema.sql. Override with SCHEMA_PATH.
func ApplySchema(ctx context.Context, pool *pgxpool.Pool) error {
	schemaPath := os.Getenv("SCHEMA_PATH")
	if schemaPath == "" {
		schemaPath = "pkg/db/schema.sql"
	}

	bytes, err := os.ReadFile(schemaPath)
	if err != nil {
		return fmt.Errorf("read schema file: %w", err)
	}

	sql := strings.TrimSpace(string(bytes))
	if sql == "" {
		return fmt.Errorf("schema file is empty: %s", schemaPath)
	}

	if _, err := pool.Exec(ctx, sql); err != nil {
		return fmt.Errorf("execute schema: %w", err)
	}

	log.Println("Schema applied from", schemaPath)
	return nil
}
