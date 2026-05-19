package config

import (
	"github.com/spf13/viper"
)

type Config struct {
	PostgresHost     string
	PostgresPort     string
	PostgresDB       string
	PostgresUser     string
	PostgresPassword string
	RedissHost       string
	RedisPort        string
	Port             string
	LogLevel         string
}

func LoadConfig() *Config {
	viper.AutomaticEnv()
	viper.SetDefault("PORT", "8080")
	viper.SetDefault("LOG_LEVEL", "DEBUG")

	return &Config{
		PostgresHost:     viper.GetString("POSTGRES_HOST"),
		PostgresPort:     viper.GetString("POSTGRES_PORT"),
		PostgresDB:       viper.GetString("POSTGRES_DB"),
		PostgresUser:     viper.GetString("POSTGRES_USER"),
		PostgresPassword: viper.GetString("POSTGRES_PASSWORD"),
		RedissHost:       viper.GetString("REDIS_HOST"),
		RedisPort:        viper.GetString("REDIS_PORT"),
		Port:             viper.GetString("PORT"),
		LogLevel:         viper.GetString("LOG_LEVEL"),
	}
}

// TODO
func ValidateConf(c *Config) error {
	return nil
}
