package config

import (
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func TestLoadConfig_DefaultsAndEnv(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)

	t.Setenv("POSTGRES_HOST", "localhost")
	t.Setenv("POSTGRES_PORT", "5432")
	t.Setenv("POSTGRES_DB", "db")
	t.Setenv("POSTGRES_USER", "user")
	t.Setenv("POSTGRES_PASSWORD", "pass")
	t.Setenv("REDIS_HOST", "redis")
	t.Setenv("REDIS_PORT", "6379")

	c := LoadConfig()
	assert.Equal(t, "8080", c.Port)
	assert.Equal(t, "DEBUG", c.LogLevel)
	assert.Equal(t, "localhost", c.PostgresHost)
	assert.Equal(t, "6379", c.RedisPort)
}
