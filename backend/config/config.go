package config

import (
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	Addr          string
	Env           string
	UploadDir     string
	MaxUploadMB   int64
	ReadTimeoutS  int
	WriteTimeoutS int
	IdleTimeoutS  int

	DBName string
	DBUser string
	DBPass string
	DBHost string
	DBPort string

	RESEND_API_KEY string
	REDIS_HOST     string
	REDIS_PORT     string
	REDIS_PASS     string
	REDIS_DB       int
	REDIS_SECRET   string

	APP_NAME           string
	APP_URL            string
	BACKEND_PUBLIC_URL string

	SUPPORT_EMAIL   string
	DEVELOPER_EMAIL string

	WORKOS_API_KEY       string
	WORKOS_CLIENT_ID     string
	WORKOS_COOKIE_SECRET string

	PASSWORD_MIN_LENGTH     int
	PASSWORD_MAX_LENGTH     int
	PASSWORD_REQUIRE_UPPER  bool
	PASSWORD_REQUIRE_LOWER  bool
	PASSWORD_REQUIRE_NUMBER bool
	PASSWORD_REQUIRE_SYMBOL bool
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func getenvStrict(k string) string {
	if v := os.Getenv(k); v != "" {
		return v
	} else {
		log.Fatalf("env variable %s is not set", k)
	}

	return ""
}

func getint(k string, def int) int {
	v := getenv(k, "")
	if v == "" {
		return def
	}
	i, err := strconv.Atoi(v)
	if err != nil {
		log.Printf("invalid int for %s: %v, using %d", k, err, def)
		return def
	}
	return i
}

// func getint64(k string, def int64) int64 {
// 	v := getenv(k, "")
// 	if v == "" {
// 		return def
// 	}
// 	i, err := strconv.ParseInt(v, 10, 64)
// 	if err != nil {
// 		log.Printf("invalid int64 for %s: %v, using %d", k, err, def)
// 		return def
// 	}
// 	return i
// }

func Load() Config {
	err := godotenv.Load()
	if err != nil {
		err = godotenv.Load(".env.local")
		if err != nil {
			log.Fatalf("failed to load env variables: %v", err)
		}
	}
	return Config{
		Addr:          getenv("BACKEND_ADDR", ":8080"),
		Env:           getenv("APP_ENV", "dev"),
		ReadTimeoutS:  getint("APP_READ_TIMEOUT_S", 15),
		WriteTimeoutS: getint("APP_WRITE_TIMEOUT_S", 30),
		IdleTimeoutS:  getint("APP_IDLE_TIMEOUT_S", 60),

		DBName: getenvStrict("DB_NAME"),
		DBUser: getenvStrict("DB_USER"),
		DBPass: getenvStrict("DB_PASS"),
		DBHost: getenvStrict("DB_HOST"),
		DBPort: getenv("DB_PORT", "5432"),

		RESEND_API_KEY: getenvStrict("RESEND_API_KEY"),

		REDIS_HOST:   getenv("REDIS_HOST", "localhost"),
		REDIS_PORT:   getenv("REDIS_PORT", "6379"),
		REDIS_PASS:   getenv("REDIS_PASS", ""),
		REDIS_DB:     getint("REDIS_DB", 0),
		REDIS_SECRET: getenvStrict("REDIS_SECRET"),

		APP_NAME:           getenvStrict("APP_NAME"),
		APP_URL:            getenvStrict("APP_URL"),
		BACKEND_PUBLIC_URL: getenvStrict("BACKEND_PUBLIC_URL"),

		SUPPORT_EMAIL:   fmt.Sprintf("support@%s", getenvStrict("APP_URL")),
		DEVELOPER_EMAIL: getenv("DEVELOPER_EMAIL", ""),

		WORKOS_API_KEY:       getenvStrict("WORKOS_API_KEY"),
		WORKOS_CLIENT_ID:     getenvStrict("WORKOS_CLIENT_ID"),
		WORKOS_COOKIE_SECRET: getenvStrict("WORKOS_COOKIE_SECRET"),

		PASSWORD_MIN_LENGTH:     8,
		PASSWORD_MAX_LENGTH:     128,
		PASSWORD_REQUIRE_UPPER:  true,
		PASSWORD_REQUIRE_LOWER:  true,
		PASSWORD_REQUIRE_NUMBER: true,
		PASSWORD_REQUIRE_SYMBOL: true,
	}
}
