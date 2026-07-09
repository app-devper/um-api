package config

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/joho/godotenv"
)

const AccessTokenTime = 24 * time.Hour

type AppConfig struct {
	Port           string
	MongoHost      string
	MongoUMDBName  string
	RedisHost      string
	SecretKey      string
	GatewayHosts   string
	LockoutEnabled bool
}

func LoadAppConfig() (*AppConfig, error) {
	_ = godotenv.Load(".env")

	cfg := &AppConfig{
		Port:           strings.TrimSpace(os.Getenv("PORT")),
		MongoHost:      strings.TrimSpace(os.Getenv("MONGO_HOST")),
		MongoUMDBName:  strings.TrimSpace(os.Getenv("MONGO_UM_DB_NAME")),
		RedisHost:      strings.TrimSpace(os.Getenv("REDIS_HOST")),
		SecretKey:      strings.TrimSpace(os.Getenv("SECRET_KEY")),
		GatewayHosts:   strings.TrimSpace(os.Getenv("GATEWAY_HOSTS")),
		LockoutEnabled: LoginLockoutEnabled(),
	}
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

func (c *AppConfig) Validate() error {
	if c == nil {
		return fmt.Errorf("config is required")
	}
	if c.Port == "" {
		return fmt.Errorf("PORT is required")
	}
	portNum, err := strconv.Atoi(c.Port)
	if err != nil || portNum < 1 || portNum > 65535 {
		return fmt.Errorf("PORT must be a valid TCP port")
	}
	if c.MongoHost == "" {
		return fmt.Errorf("MONGO_HOST is required")
	}
	if err := validateMongoHost(c.MongoHost); err != nil {
		return err
	}
	if c.MongoUMDBName == "" {
		return fmt.Errorf("MONGO_UM_DB_NAME is required")
	}
	if c.RedisHost == "" {
		return fmt.Errorf("REDIS_HOST is required")
	}
	if err := validateRedisHost(c.RedisHost); err != nil {
		return err
	}
	if c.SecretKey == "" {
		return fmt.Errorf("SECRET_KEY is required")
	}
	return nil
}

func (c *AppConfig) ListenAddr() string {
	return net.JoinHostPort("", c.Port)
}

// LoginLockoutEnabled returns whether per-username brute-force lockout is active.
// Controlled by env LOGIN_LOCKOUT_ENABLED; any of "1", "true", "yes", "on" (case-insensitive) enables it.
// Default is disabled.
func LoginLockoutEnabled() bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv("LOGIN_LOCKOUT_ENABLED")))
	return v == "1" || v == "true" || v == "yes" || v == "on"
}

func validateMongoHost(value string) error {
	value = strings.TrimSpace(value)
	if strings.HasPrefix(value, "mongodb://") || strings.HasPrefix(value, "mongodb+srv://") {
		u, err := url.Parse(value)
		if err != nil || u.Host == "" {
			return fmt.Errorf("MONGO_HOST must be a valid MongoDB URI or host:port")
		}
		return nil
	}
	if err := validateHostPort(value); err != nil {
		return fmt.Errorf("MONGO_HOST must be a valid MongoDB URI or host:port")
	}
	return nil
}

func validateRedisHost(value string) error {
	value = strings.TrimSpace(value)
	if strings.Contains(value, "://") {
		u, parseErr := url.Parse(value)
		if parseErr != nil || strings.TrimSpace(u.Host) == "" {
			return fmt.Errorf("REDIS_HOST must be a valid Redis URL or host:port")
		}
		opts, err := redis.ParseURL(value)
		if err != nil || strings.TrimSpace(opts.Addr) == "" {
			return fmt.Errorf("REDIS_HOST must be a valid Redis URL or host:port")
		}
		return nil
	}
	if err := validateHostPort(value); err != nil {
		return fmt.Errorf("REDIS_HOST must be a valid Redis URL or host:port")
	}
	return nil
}

func validateHostPort(value string) error {
	host, port, err := net.SplitHostPort(value)
	if err != nil || strings.TrimSpace(host) == "" {
		return fmt.Errorf("invalid host:port")
	}
	portNum, err := strconv.Atoi(port)
	if err != nil || portNum < 1 || portNum > 65535 {
		return fmt.Errorf("invalid host:port")
	}
	return nil
}
