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
	if c.Server.Port == "" {
		return fmt.Errorf(errPortRequiredFmt)
	}

	if c.Database.Password == "" {
		return fmt.Errorf(errDBPasswordRequiredFmt)
	}

	if c.AWS.Region == "" {
		return fmt.Errorf(errRegionRequiredFmt)
	}

	if c.AWS.AccessKeyID == "" {
		return fmt.Errorf(errAWSAccessKeyRequiredFmt)
	}

	if c.AWS.SecretAccessKey == "" {
		return fmt.Errorf(errAWSSecretKeyRequiredFmt)
	}

	if c.JWT.Secret == "" {
		return fmt.Errorf(errJWTSecretRequiredFmt)
	}

	if len(c.JWT.Secret) < minJWTSecretLength {
		return fmt.Errorf(errJWTSecretMinLengthFmt, minJWTSecretLength)
	}

	if !hasMinimumEntropy(c.JWT.Secret) {
		return fmt.Errorf(errJWTSecretLowEntropyFmt)
	}

	return nil
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
		panic(messages.requiredEnvNotSet(key))
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
