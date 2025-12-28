package config

import (
	"fmt"
	"os"

	"github.com/ilyakaznacheev/cleanenv"
	"github.com/leonid6372/success-bot/pkg/log"
)

const (
	EnvProd = "prod"
	EnvTest = "test"
)

type Config struct {
	Env      string `yaml:"env" env:"ENV" env-default:"test" env-upd:""`
	Location string `yaml:"location" env:"LOCATION" env-default:"Europe/Moscow" env-upd:""`

	Postgres Postgres `yaml:"postgres"`

	Bot Bot `yaml:"bot"`
}

type Postgres struct {
	Database string `yaml:"database" env:"POSTGRES_DATABASE" env-default:"success_bot"`
	Host     string `yaml:"host" env:"POSTGRES_HOST" env-default:"localhost"`
	Username string `yaml:"user" env:"POSTGRES_USER" env-default:"admin"`
	Password string `yaml:"password" env:"POSTGRES_PASSWORD" env-default:"1111"`
	Port     int64  `yaml:"port" env:"POSTGRES_PORT" env-default:"5432"`
}

type Bot struct {
	APIKey string `yaml:"api_key" env:"BOT_API_KEY" env-required:"true" env-upd:"true"`
}

func (c *Config) GetPostgresURL() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable",
		c.Postgres.Username, c.Postgres.Password, c.Postgres.Host, c.Postgres.Port, c.Postgres.Database)
}

func GetConfigFileName() string {
	env := EnvTest
	if os.Getenv("ENV") == EnvProd {
		env = EnvProd
	}

	return fmt.Sprintf("config/%s.yaml", env)
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
