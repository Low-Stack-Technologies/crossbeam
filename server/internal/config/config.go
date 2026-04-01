package config

import (
	"fmt"
	"strconv"
	"time"

	"github.com/caarlos0/env/v11"
)

type Config struct {
	NodeEnv string `env:"NODE_ENV" envDefault:"development"`
	AppURL  string `env:"APP_URL"  envDefault:"http://localhost:3000"`
	Port    int    `env:"PORT"     envDefault:"3000"`

	DatabaseURL string `env:"DATABASE_URL,required"`
	RedisURL    string `env:"REDIS_URL,required"`

	JWTSecret    string `env:"JWT_SECRET,required"`
	JWTExpiresIn string `env:"JWT_EXPIRES_IN" envDefault:"30d"`

	S3Endpoint  string `env:"S3_ENDPOINT,required"`
	S3Bucket    string `env:"S3_BUCKET"    envDefault:"crossbeam-uploads"`
	S3AccessKey string `env:"S3_ACCESS_KEY,required"`
	S3SecretKey string `env:"S3_SECRET_KEY,required"`
	S3Region    string `env:"S3_REGION"    envDefault:"us-east-1"`
}

func Load() (*Config, error) {
	cfg := &Config{}
	if err := env.Parse(cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

// ParseExpiry converts strings like "30d", "24h", "60m" to a time.Duration.
func ParseExpiry(s string) (time.Duration, error) {
	if len(s) < 2 {
		return 0, fmt.Errorf("invalid expiry: %q", s)
	}
	unit := s[len(s)-1]
	n, err := strconv.Atoi(s[:len(s)-1])
	if err != nil {
		return time.ParseDuration(s)
	}
	switch unit {
	case 'd':
		return time.Duration(n) * 24 * time.Hour, nil
	case 'h':
		return time.Duration(n) * time.Hour, nil
	case 'm':
		return time.Duration(n) * time.Minute, nil
	default:
		return time.ParseDuration(s)
	}
}
