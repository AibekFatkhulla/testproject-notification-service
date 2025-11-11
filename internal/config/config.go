package config

import (
	"time"

	"github.com/caarlos0/env/v11"
)

type DB struct {
	URL             string        `env:"DATABASE_URL,required"`
	MaxOpenConns    int           `env:"DB_MAX_OPEN_CONNS" envDefault:"16"`
	MaxIdleConns    int           `env:"DB_MAX_IDLE_CONNS" envDefault:"8"`
	ConnMaxLifetime time.Duration `env:"DB_CONN_MAX_LIFETIME" envDefault:"1h"`
	ConnMaxIdleTime time.Duration `env:"DB_CONN_MAX_IDLE_TIME" envDefault:"15m"`
}

type Config struct {
	DB DB
}

func Load() (*Config, error) {
	cfg := &Config{}
	if err := env.Parse(cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}
