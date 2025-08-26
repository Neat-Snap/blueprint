package config

import (
	"log"
	"os"
	"strconv"
)

type Config struct {
	Addr          string
	Env           string
	UploadDir     string
	MaxUploadMB   int64
	ReadTimeoutS  int
	WriteTimeoutS int
	IdleTimeoutS  int
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
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

func getint64(k string, def int64) int64 {
	v := getenv(k, "")
	if v == "" {
		return def
	}
	i, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		log.Printf("invalid int64 for %s: %v, using %d", k, err, def)
		return def
	}
	return i
}

func Load() Config {
	return Config{
		Addr:          getenv("APP_ADDR", ":8080"),
		Env:           getenv("APP_ENV", "dev"),
		UploadDir:     getenv("APP_UPLOAD_DIR", "./uploads"),
		MaxUploadMB:   getint64("APP_MAX_UPLOAD_MB", 50),
		ReadTimeoutS:  getint("APP_READ_TIMEOUT_S", 15),
		WriteTimeoutS: getint("APP_WRITE_TIMEOUT_S", 30),
		IdleTimeoutS:  getint("APP_IDLE_TIMEOUT_S", 60),
	}
}
