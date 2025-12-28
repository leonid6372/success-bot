package config

import (
	"fmt"

	"github.com/ilyakaznacheev/cleanenv"
	"github.com/leonid6372/success-bot/pkg/log"
)

const (
	EnvProd = "prod"
	EnvTest = "test"
)

type Config struct {
	Env      string `yaml:"env" env:"ENV" env-upd:""`
	Location string `yaml:"location" env:"LOCATION" env-upd:""`

	Postgres Postgres `yaml:"postgres"`

	Bot Bot `yaml:"bot"`
}

type Postgres struct {
	Database string `yaml:"database" env:"POSTGRES_DATABASE" env-upd:""`
	Host     string `yaml:"host" env:"POSTGRES_HOST" env-upd:""`
	Schema   string `yaml:"schema" env:"POSTGRES_SCHEMA" env-upd:""`
	Username string `yaml:"username" env:"POSTGRES_USER" env-upd:""`
	Password string `yaml:"password" env:"POSTGRES_PASSWORD" env-upd:""`
	Port     int64  `yaml:"port" env:"POSTGRES_PORT" env-upd:""`
}

type Bot struct {
	APIKey string `yaml:"api_key" env:"BOT_API_KEY" env-upd:""`
}

func (c *Config) GetPostgresURL() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable",
		c.Postgres.Username, c.Postgres.Password, c.Postgres.Host, c.Postgres.Port, c.Postgres.Database)
}

func GetConfig(configPath string) *Config {
	if configPath == "" {
		log.Fatal("config path is required")
	}

	var cfg Config
	if err := cleanenv.ReadConfig(configPath, &cfg); err != nil {
		log.Fatal(err.Error())
	}

	if err := cleanenv.UpdateEnv(&cfg); err != nil {
		log.Fatal(err.Error())
	}

	return &cfg
}
