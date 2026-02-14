package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

const (
	envPort                  = "PORT"
	envServerReadTimeout     = "SERVER_READ_TIMEOUT"
	envServerWriteTimeout    = "SERVER_WRITE_TIMEOUT"
	envServerShutdownTimeout = "SERVER_SHUTDOWN_TIMEOUT"
	envDBHost                = "DB_HOST"
	envDBPort                = "DB_PORT"
	envDBName                = "DB_NAME"
	envDBUser                = "DB_USER"
	envDBPassword            = "DB_PASSWORD"
	envDBSSLMode             = "DB_SSL_MODE"
	envDBMaxConns            = "DB_MAX_CONNS"
	envDBMinConns            = "DB_MIN_CONNS"
	envAWSRegion             = "REGION"
	envAWSAccessKeyID        = "AWS_ACCESS_KEY_ID"
	envAWSSecretAccessKey    = "AWS_SECRET_ACCESS_KEY"
	envJWTSecret             = "JWT_SECRET"
	envJWTExpiry             = "JWT_EXPIRY_MINUTES"
	envDownloadURLTimeLimit  = "DOWNLOAD_URL_TIME_LIMIT"
	envPaginationPageSize    = "PAGINATION_PAGE_SIZE"
	envMaxUploadSize         = "MAX_UPLOAD_SIZE"
)

const (
	defaultServerPort          = "8080"
	defaultServerReadTimeout   = 10 * time.Second
	defaultServerWriteTimeout  = 10 * time.Second
	defaultServerShutdown      = 10 * time.Second
	defaultDBHost              = "localhost"
	defaultDBPort              = 5432
	defaultDBName              = "fileservice"
	defaultDBUser              = "fileservice_app"
	defaultDBSSLMode           = "disable"
	defaultDBMaxConns          = 25
	defaultDBMinConns          = 5
	defaultJWTExpiry           = 60 * time.Minute
	defaultPresignedURLExpiry  = 15 * time.Minute
	defaultPageSize            = 100
	defaultMaxUploadSize       = int64(100 * 1024 * 1024 * 1024)
	minJWTSecretLength         = 32
	minUniqueCharsInSecret     = 16
	minRepeatedCharThreshold   = 4
	maxRepeatedChars           = 2
	errPortRequiredFmt         = "PORT must be set"
	errDBPasswordRequiredFmt   = "DB_PASSWORD must be set"
	errRegionRequiredFmt       = "REGION must be set"
	errAWSAccessKeyRequiredFmt = "AWS_ACCESS_KEY_ID must be set"
	errAWSSecretKeyRequiredFmt = "AWS_SECRET_ACCESS_KEY must be set"
	errJWTSecretRequiredFmt    = "JWT_SECRET must be set"
	errJWTSecretMinLengthFmt   = "JWT_SECRET must be at least %d characters"
	errJWTSecretLowEntropyFmt  = "JWT_SECRET has insufficient entropy (appears non-random). Use a cryptographically secure random string."
	errInvalidConfigurationFmt = "invalid configuration: %w"
)

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	AWS      AWSConfig
	JWT      JWTConfig
	App      AppConfig
}

type ServerConfig struct {
	Port            string
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	ShutdownTimeout time.Duration
}

type DatabaseConfig struct {
	Host     string
	Port     int
	Database string
	User     string
	Password string
	SSLMode  string
	MaxConns int
	MinConns int
}

type AWSConfig struct {
	Region          string
	AccessKeyID     string
	SecretAccessKey string
}

type JWTConfig struct {
	Secret         string
	ExpiryDuration time.Duration
}

type AppConfig struct {
	PresignedURLExpiry time.Duration
	PageSize           int
	MaxUploadSize      int64
}

func Load() (*Config, error) {
	cfg := &Config{
		Server: ServerConfig{
			Port:            getEnv(envPort, defaultServerPort),
			ReadTimeout:     getDurationEnv(envServerReadTimeout, defaultServerReadTimeout),
			WriteTimeout:    getDurationEnv(envServerWriteTimeout, defaultServerWriteTimeout),
			ShutdownTimeout: getDurationEnv(envServerShutdownTimeout, defaultServerShutdown),
		},
		Database: DatabaseConfig{
			Host:     getEnv(envDBHost, defaultDBHost),
			Port:     getIntEnv(envDBPort, defaultDBPort),
			Database: getEnv(envDBName, defaultDBName),
			User:     getEnv(envDBUser, defaultDBUser),
			Password: requireEnv(envDBPassword),
			SSLMode:  getEnv(envDBSSLMode, defaultDBSSLMode),
			MaxConns: getIntEnv(envDBMaxConns, defaultDBMaxConns),
			MinConns: getIntEnv(envDBMinConns, defaultDBMinConns),
		},
		AWS: AWSConfig{
			Region:          requireEnv(envAWSRegion),
			AccessKeyID:     requireEnv(envAWSAccessKeyID),
			SecretAccessKey: requireEnv(envAWSSecretAccessKey),
		},
		JWT: JWTConfig{
			Secret:         requireEnv(envJWTSecret),
			ExpiryDuration: getDurationEnv(envJWTExpiry, defaultJWTExpiry),
		},
		App: AppConfig{
			PresignedURLExpiry: getDurationEnv(envDownloadURLTimeLimit, defaultPresignedURLExpiry),
			PageSize:           getIntEnv(envPaginationPageSize, defaultPageSize),
			MaxUploadSize:      getInt64Env(envMaxUploadSize, defaultMaxUploadSize),
		},
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf(errInvalidConfigurationFmt, err)
	}

	return cfg, nil
}

func (c *Config) Validate() error {
	var errs []error

	// Server validation
	if c.Server.Port == "" {
		errs = append(errs, fmt.Errorf("PORT must be set"))
	}
	if c.Server.ReadTimeout <= 0 {
		errs = append(errs, fmt.Errorf("SERVER_READ_TIMEOUT must be positive"))
	}
	if c.Server.WriteTimeout <= 0 {
		errs = append(errs, fmt.Errorf("SERVER_WRITE_TIMEOUT must be positive"))
	}
	if c.Server.ShutdownTimeout <= 0 {
		errs = append(errs, fmt.Errorf("SERVER_SHUTDOWN_TIMEOUT must be positive"))
	}

	// Database validation
	if c.Database.Password == "" {
		errs = append(errs, fmt.Errorf("DB_PASSWORD must be set"))
	}
	if c.Database.Host == "" {
		errs = append(errs, fmt.Errorf("DB_HOST must be set"))
	}
	if c.Database.Port <= 0 || c.Database.Port > 65535 {
		errs = append(errs, fmt.Errorf("DB_PORT must be between 1 and 65535"))
	}
	if c.Database.Database == "" {
		errs = append(errs, fmt.Errorf("DB_NAME must be set"))
	}
	if c.Database.User == "" {
		errs = append(errs, fmt.Errorf("DB_USER must be set"))
	}
	if c.Database.MaxConns <= 0 {
		errs = append(errs, fmt.Errorf("DB_MAX_CONNS must be positive"))
	}
	if c.Database.MinConns < 0 {
		errs = append(errs, fmt.Errorf("DB_MIN_CONNS must be non-negative"))
	}
	if c.Database.MinConns > c.Database.MaxConns {
		errs = append(errs, fmt.Errorf("DB_MIN_CONNS cannot exceed DB_MAX_CONNS"))
	}

	// AWS validation
	if c.AWS.Region == "" {
		errs = append(errs, fmt.Errorf("REGION must be set"))
	}
	if c.AWS.AccessKeyID == "" {
		errs = append(errs, fmt.Errorf("AWS_ACCESS_KEY_ID must be set"))
	}
	if c.AWS.SecretAccessKey == "" {
		errs = append(errs, fmt.Errorf("AWS_SECRET_ACCESS_KEY must be set"))
	}

	// JWT validation (security-sensitive)
	if c.JWT.Secret == "" {
		errs = append(errs, fmt.Errorf("JWT_SECRET must be set"))
	} else {
		if len(c.JWT.Secret) < minJWTSecretLength {
			errs = append(errs, fmt.Errorf("JWT_SECRET must be at least %d characters", minJWTSecretLength))
		}
		if !hasMinimumEntropy(c.JWT.Secret) {
			errs = append(errs, fmt.Errorf("JWT_SECRET has insufficient entropy (appears non-random). Use a cryptographically secure random string"))
		}
	}
	if c.JWT.ExpiryDuration <= 0 {
		errs = append(errs, fmt.Errorf("JWT_EXPIRY_MINUTES must be positive"))
	}

	// App validation
	if c.App.PresignedURLExpiry <= 0 {
		errs = append(errs, fmt.Errorf("DOWNLOAD_URL_TIME_LIMIT must be positive"))
	}
	if c.App.PageSize <= 0 {
		errs = append(errs, fmt.Errorf("PAGINATION_PAGE_SIZE must be positive"))
	}
	if c.App.MaxUploadSize <= 0 {
		errs = append(errs, fmt.Errorf("MAX_UPLOAD_SIZE must be positive"))
	}

	// Return all errors joined
	if len(errs) > 0 {
		return fmt.Errorf("configuration validation failed: %w", joinErrors(errs))
	}

	return nil
}

// joinErrors combines multiple errors into a single error
func joinErrors(errs []error) error {
	if len(errs) == 0 {
		return nil
	}
	if len(errs) == 1 {
		return errs[0]
	}

	msg := errs[0].Error()
	for i := 1; i < len(errs); i++ {
		msg += "; " + errs[i].Error()
	}
	return fmt.Errorf("%s", msg)
}

func hasMinimumEntropy(secret string) bool {
	if len(secret) < minJWTSecretLength {
		return false
	}

	charCounts := make(map[rune]int)
	for _, char := range secret {
		charCounts[char]++
	}

	uniqueChars := len(charCounts)
	if uniqueChars < minUniqueCharsInSecret {
		return false
	}

	repeatedChars := 0
	for _, count := range charCounts {
		if count > len(secret)/minRepeatedCharThreshold {
			repeatedChars++
		}
	}

	return repeatedChars <= maxRepeatedChars
}

func (c *DatabaseConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.Database, c.SSLMode,
	)
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func requireEnv(key string) string {
	value := os.Getenv(key)
	if value == "" {
		panic(fmt.Sprintf("required environment variable %s is not set", key))
	}
	return value
}

func getIntEnv(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}

func getInt64Env(key string, defaultValue int64) int64 {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.ParseInt(value, 10, 64); err == nil {
			return intVal
		}
	}
	return defaultValue
}

func getDurationEnv(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
		if minutes, err := strconv.Atoi(value); err == nil {
			return time.Duration(minutes) * time.Minute
		}
	}
	return defaultValue
}
