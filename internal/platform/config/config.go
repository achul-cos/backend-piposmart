package config

import (
	"bufio"
	"errors"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	drivermysql "github.com/go-sql-driver/mysql"
)

const (
	EnvironmentLocal       = "local"
	EnvironmentDevelopment = "development"
	EnvironmentTest        = "test"
	EnvironmentStaging     = "staging"
	EnvironmentProduction  = "production"
)

type Config struct {
	App       AppConfig
	HTTP      HTTPConfig
	Log       LogConfig
	Database  DatabaseConfig
	CORS      CORSConfig
	Storage   StorageConfig
	Worker    WorkerConfig
	Migration MigrationConfig
	Auth      AuthConfig
	Bootstrap BootstrapConfig
}

type AppConfig struct {
	Name            string
	Environment     string
	Host            string
	Port            int
	BasePath        string
	Timezone        *time.Location
	TimezoneName    string
	ShutdownTimeout time.Duration
}

func (c AppConfig) Address() string {
	return net.JoinHostPort(c.Host, strconv.Itoa(c.Port))
}

func (c AppConfig) IsProduction() bool {
	return c.Environment == EnvironmentProduction
}

type HTTPConfig struct {
	ReadTimeout       time.Duration
	ReadHeaderTimeout time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
}

type LogConfig struct {
	Level  string
	Format string
}

type DatabaseConfig struct {
	Host            string
	Port            int
	Name            string
	User            string
	Password        string
	Charset         string
	ParseTime       bool
	Location        string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	ConnectTimeout  time.Duration
}

func (c DatabaseConfig) DSN() string {
	location, _ := time.LoadLocation(c.Location)
	driverConfig := drivermysql.Config{
		User:      c.User,
		Passwd:    c.Password,
		Net:       "tcp",
		Addr:      net.JoinHostPort(c.Host, strconv.Itoa(c.Port)),
		DBName:    c.Name,
		Params:    map[string]string{"charset": c.Charset},
		ParseTime: c.ParseTime,
		Loc:       location,
		Timeout:   c.ConnectTimeout,
	}
	return driverConfig.FormatDSN()
}

type CORSConfig struct {
	AllowedOrigins   []string
	AllowCredentials bool
}

type StorageConfig struct {
	UploadDirectory string
	ExportDirectory string
	MaxUploadSizeMB int64
}

type WorkerConfig struct {
	PollInterval    time.Duration
	MaxAttempts     int
	StaleJobTimeout time.Duration
}

type MigrationConfig struct {
	Directory string
}

type AuthConfig struct {
	Issuer         string
	AccessSecret   string
	AccessTTL      time.Duration
	RefreshTTL     time.Duration
	CookieName     string
	CookieDomain   string
	CookieSecure   bool
	CookieSameSite string
}

type BootstrapConfig struct {
	AdminName     string
	AdminEmail    string
	AdminPhone    string
	AdminPassword string
}

func Load() (Config, error) {
	if err := LoadDotEnv(".env"); err != nil && !errors.Is(err, os.ErrNotExist) {
		return Config{}, fmt.Errorf("load .env: %w", err)
	}

	return LoadFromEnvironment()
}

func LoadFromEnvironment() (Config, error) {
	timezoneName := getEnv("APP_TIMEZONE", "Asia/Jakarta")
	timezone, err := time.LoadLocation(timezoneName)
	if err != nil {
		return Config{}, fmt.Errorf("APP_TIMEZONE tidak valid: %w", err)
	}

	cfg := Config{
		App: AppConfig{
			Name:            getEnv("APP_NAME", "crm-piposmart-backend"),
			Environment:     strings.ToLower(getEnv("APP_ENV", EnvironmentLocal)),
			Host:            getEnv("APP_HOST", "0.0.0.0"),
			Port:            getInt("APP_PORT", 8080),
			BasePath:        getEnv("APP_BASE_PATH", "/api/v1"),
			Timezone:        timezone,
			TimezoneName:    timezoneName,
			ShutdownTimeout: getDuration("APP_SHUTDOWN_TIMEOUT", 10*time.Second),
		},
		HTTP: HTTPConfig{
			ReadTimeout:       getDuration("HTTP_READ_TIMEOUT", 15*time.Second),
			ReadHeaderTimeout: getDuration("HTTP_READ_HEADER_TIMEOUT", 5*time.Second),
			WriteTimeout:      getDuration("HTTP_WRITE_TIMEOUT", 30*time.Second),
			IdleTimeout:       getDuration("HTTP_IDLE_TIMEOUT", 60*time.Second),
		},
		Log: LogConfig{
			Level:  strings.ToLower(getEnv("LOG_LEVEL", "info")),
			Format: strings.ToLower(getEnv("LOG_FORMAT", "json")),
		},
		Database: DatabaseConfig{
			Host:            getEnv("DB_HOST", ""),
			Port:            getInt("DB_PORT", 3306),
			Name:            getEnv("DB_NAME", ""),
			User:            getEnv("DB_USER", ""),
			Password:        getEnv("DB_PASSWORD", ""),
			Charset:         getEnv("DB_CHARSET", "utf8mb4"),
			ParseTime:       getBool("DB_PARSE_TIME", true),
			Location:        getEnv("DB_LOCATION", "UTC"),
			MaxOpenConns:    getInt("DB_MAX_OPEN_CONNS", 25),
			MaxIdleConns:    getInt("DB_MAX_IDLE_CONNS", 10),
			ConnMaxLifetime: getDuration("DB_CONN_MAX_LIFETIME", 30*time.Minute),
			ConnectTimeout:  getDuration("DB_CONNECT_TIMEOUT", 10*time.Second),
		},
		CORS: CORSConfig{
			AllowedOrigins:   getCSV("CORS_ALLOWED_ORIGINS", []string{"http://localhost:3000"}),
			AllowCredentials: getBool("CORS_ALLOW_CREDENTIALS", true),
		},
		Storage: StorageConfig{
			UploadDirectory: getEnv("UPLOAD_DIR", "./storage/uploads"),
			ExportDirectory: getEnv("EXPORT_DIR", "./storage/exports"),
			MaxUploadSizeMB: int64(getInt("MAX_UPLOAD_SIZE_MB", 20)),
		},
		Worker: WorkerConfig{
			PollInterval:    getDuration("WORKER_POLL_INTERVAL", 3*time.Second),
			MaxAttempts:     getInt("WORKER_MAX_ATTEMPTS", 5),
			StaleJobTimeout: getDuration("WORKER_STALE_JOB_TIMEOUT", 15*time.Minute),
		},
		Migration: MigrationConfig{
			Directory: getEnv("MIGRATION_DIR", "./migrations"),
		},
		Auth: AuthConfig{
			Issuer:         getEnv("JWT_ISSUER", "crm-piposmart"),
			AccessSecret:   getEnv("JWT_ACCESS_SECRET", ""),
			AccessTTL:      getDuration("JWT_ACCESS_TTL", 15*time.Minute),
			RefreshTTL:     getDuration("JWT_REFRESH_TTL", 168*time.Hour),
			CookieName:     getEnv("AUTH_COOKIE_NAME", "piposmart_refresh"),
			CookieDomain:   getEnv("AUTH_COOKIE_DOMAIN", ""),
			CookieSecure:   getBool("AUTH_COOKIE_SECURE", false),
			CookieSameSite: strings.ToLower(getEnv("AUTH_COOKIE_SAME_SITE", "lax")),
		},
		Bootstrap: BootstrapConfig{
			AdminName:     getEnv("BOOTSTRAP_ADMIN_NAME", "Initial Admin"),
			AdminEmail:    getEnv("BOOTSTRAP_ADMIN_EMAIL", ""),
			AdminPhone:    getEnv("BOOTSTRAP_ADMIN_PHONE", ""),
			AdminPassword: getEnv("BOOTSTRAP_ADMIN_PASSWORD", ""),
		},
	}

	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

func (c Config) Validate() error {
	var problems []string

	validEnvironments := map[string]bool{
		EnvironmentLocal:       true,
		EnvironmentDevelopment: true,
		EnvironmentTest:        true,
		EnvironmentStaging:     true,
		EnvironmentProduction:  true,
	}
	if !validEnvironments[c.App.Environment] {
		problems = append(problems, "APP_ENV harus local, development, test, staging, atau production")
	}
	if strings.TrimSpace(c.App.Name) == "" {
		problems = append(problems, "APP_NAME wajib diisi")
	}
	if c.App.Port < 1 || c.App.Port > 65535 {
		problems = append(problems, "APP_PORT harus berada di antara 1 dan 65535")
	}
	if !strings.HasPrefix(c.App.BasePath, "/") {
		problems = append(problems, "APP_BASE_PATH harus diawali /")
	}
	if c.App.ShutdownTimeout <= 0 {
		problems = append(problems, "APP_SHUTDOWN_TIMEOUT harus lebih besar dari 0")
	}

	if strings.TrimSpace(c.Database.Host) == "" {
		problems = append(problems, "DB_HOST wajib diisi")
	}
	if strings.TrimSpace(c.Database.Name) == "" {
		problems = append(problems, "DB_NAME wajib diisi")
	}
	if strings.TrimSpace(c.Database.User) == "" {
		problems = append(problems, "DB_USER wajib diisi")
	}
	if strings.TrimSpace(c.Database.Password) == "" {
		problems = append(problems, "DB_PASSWORD wajib diisi")
	}
	if c.Database.Port < 1 || c.Database.Port > 65535 {
		problems = append(problems, "DB_PORT harus berada di antara 1 dan 65535")
	}
	if c.Database.MaxOpenConns < 1 {
		problems = append(problems, "DB_MAX_OPEN_CONNS minimal 1")
	}
	if c.Database.MaxIdleConns < 0 || c.Database.MaxIdleConns > c.Database.MaxOpenConns {
		problems = append(problems, "DB_MAX_IDLE_CONNS harus antara 0 dan DB_MAX_OPEN_CONNS")
	}
	if c.Database.ConnectTimeout <= 0 {
		problems = append(problems, "DB_CONNECT_TIMEOUT harus lebih besar dari 0")
	}
	if _, err := time.LoadLocation(c.Database.Location); err != nil {
		problems = append(problems, "DB_LOCATION tidak valid")
	}

	if len(c.CORS.AllowedOrigins) == 0 {
		problems = append(problems, "CORS_ALLOWED_ORIGINS minimal memiliki satu origin")
	}
	if c.CORS.AllowCredentials {
		for _, origin := range c.CORS.AllowedOrigins {
			if origin == "*" {
				problems = append(problems, "CORS_ALLOWED_ORIGINS tidak boleh * ketika credential diaktifkan")
			}
		}
	}
	if c.App.IsProduction() {
		for _, origin := range c.CORS.AllowedOrigins {
			if origin == "*" {
				problems = append(problems, "CORS_ALLOWED_ORIGINS tidak boleh * di production")
			}
		}
	}

	if c.HTTP.ReadTimeout <= 0 ||
		c.HTTP.ReadHeaderTimeout <= 0 ||
		c.HTTP.WriteTimeout <= 0 ||
		c.HTTP.IdleTimeout <= 0 {
		problems = append(problems, "seluruh HTTP timeout harus lebih besar dari 0")
	}
	if c.Worker.PollInterval <= 0 {
		problems = append(problems, "WORKER_POLL_INTERVAL harus lebih besar dari 0")
	}
	if c.Worker.MaxAttempts < 1 {
		problems = append(problems, "WORKER_MAX_ATTEMPTS minimal 1")
	}
	if strings.TrimSpace(c.Migration.Directory) == "" {
		problems = append(problems, "MIGRATION_DIR wajib diisi")
	}
	if strings.TrimSpace(c.Auth.Issuer) == "" {
		problems = append(problems, "JWT_ISSUER wajib diisi")
	}
	if len(c.Auth.AccessSecret) < 32 {
		problems = append(problems, "JWT_ACCESS_SECRET minimal 32 karakter")
	}
	if c.Auth.AccessTTL <= 0 {
		problems = append(problems, "JWT_ACCESS_TTL harus lebih besar dari 0")
	}
	if c.Auth.RefreshTTL <= c.Auth.AccessTTL {
		problems = append(problems, "JWT_REFRESH_TTL harus lebih besar dari JWT_ACCESS_TTL")
	}
	validSameSite := map[string]bool{"lax": true, "strict": true, "none": true}
	if !validSameSite[c.Auth.CookieSameSite] {
		problems = append(problems, "AUTH_COOKIE_SAME_SITE harus lax, strict, atau none")
	}
	if c.App.IsProduction() {
		if c.Auth.AccessSecret == "replace-with-minimum-32-byte-secret" {
			problems = append(problems, "JWT_ACCESS_SECRET production tidak boleh memakai nilai contoh")
		}
		if !c.Auth.CookieSecure && c.Auth.CookieSameSite == "none" {
			problems = append(problems, "AUTH_COOKIE_SECURE harus true ketika same-site none di production")
		}
	}

	if len(problems) > 0 {
		return fmt.Errorf("konfigurasi tidak valid: %s", strings.Join(problems, "; "))
	}

	return nil
}

func LoadDotEnv(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "export ") {
			line = strings.TrimSpace(strings.TrimPrefix(line, "export "))
		}

		key, value, found := strings.Cut(line, "=")
		if !found {
			return fmt.Errorf("baris %d tidak memiliki pemisah =", lineNumber)
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if key == "" {
			return fmt.Errorf("baris %d memiliki nama environment kosong", lineNumber)
		}
		if len(value) >= 2 {
			if (value[0] == '"' && value[len(value)-1] == '"') ||
				(value[0] == '\'' && value[len(value)-1] == '\'') {
				value = value[1 : len(value)-1]
			}
		}

		if _, exists := os.LookupEnv(key); !exists {
			if err := os.Setenv(key, value); err != nil {
				return fmt.Errorf("set %s: %w", key, err)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("baca dotenv: %w", err)
	}
	return nil
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return strings.TrimSpace(value)
	}
	return fallback
}

func getInt(key string, fallback int) int {
	raw := getEnv(key, "")
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return -1
	}
	return value
}

func getBool(key string, fallback bool) bool {
	raw := getEnv(key, "")
	if raw == "" {
		return fallback
	}
	value, err := strconv.ParseBool(raw)
	if err != nil {
		return false
	}
	return value
}

func getDuration(key string, fallback time.Duration) time.Duration {
	raw := getEnv(key, "")
	if raw == "" {
		return fallback
	}
	value, err := time.ParseDuration(raw)
	if err != nil {
		return -1
	}
	return value
}

func getCSV(key string, fallback []string) []string {
	raw := getEnv(key, "")
	if raw == "" {
		return fallback
	}

	values := make([]string, 0)
	for _, value := range strings.Split(raw, ",") {
		value = strings.TrimSpace(value)
		if value != "" {
			values = append(values, value)
		}
	}
	return values
}
