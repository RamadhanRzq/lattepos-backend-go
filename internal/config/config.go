package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

// DBConfig menyimpan konfigurasi koneksi PostgreSQL.
type DBConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Name     string
	SSLMode  string
}

// AppConfig menyimpan konfigurasi aplikasi.
type AppConfig struct {
	Port          string
	JWTSecret     string
	JWTExpiration int // dalam menit
}

// Config menyimpan seluruh konfigurasi aplikasi.
type Config struct {
	DB  DBConfig
	App AppConfig
}

// Load membaca konfigurasi dari environment. File .env (jika ada) dimuat
// dulu dengan godotenv; variabel yang sudah ada di environment tidak dioverride.
func Load() (*Config, error) {
	if err := godotenv.Load(".env"); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("config: gagal memuat .env: %w", err)
	}

	cfg := &Config{
		DB: DBConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnv("DB_PORT", "5432"),
			User:     getEnv("DB_USER", "postgres"),
			Password: getEnv("DB_PASSWORD", ""),
			Name:     getEnv("DB_NAME", "lattepos"),
			SSLMode:  getEnv("DB_SSLMODE", "disable"),
		},
		App: AppConfig{
			Port:          getEnv("PORT", "8000"),
			JWTSecret:     getEnv("JWT_SECRET", ""),
			JWTExpiration: getEnvInt("JWT_EXPIRATION_MINUTES", 60),
		},
	}

	if cfg.App.JWTSecret == "" {
		return nil, fmt.Errorf("config: JWT_SECRET wajib diisi (cek .env)")
	}

	return cfg, nil
}

// DSN mengembalikan connection string PostgreSQL.
func (c *Config) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		c.DB.Host, c.DB.Port, c.DB.User, c.DB.Password, c.DB.Name, c.DB.SSLMode,
	)
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	raw := getEnv(key, "")
	if raw == "" {
		return fallback
	}

	value, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}

	return value
}
