package config

import (
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

type EmbeddingConfig struct {
	Provider string `mapstructure:"provider"` // "openai", "anthropic", etc.
	Model    string `mapstructure:"model"`    // Provider-specific model name
}

type Config struct {
	DBDSN        string          `mapstructure:"db_dsn"`
	OpenAIAPIKey string          `mapstructure:"openai_api_key"`
	Embedding    EmbeddingConfig `mapstructure:"embedding"`
}

func Load() (*Config, error) {
	v := viper.New()

	v.SetDefault("db_dsn", "postgres://postgres:postgres@localhost/postgres")
	v.SetDefault("embedding.provider", "openai")
	v.SetDefault("embedding.model", "text-embedding-3-small")

	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath(".")

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed to read config: %w", err)
		}
	}

	fmt.Println("Using config file:", v.ConfigFileUsed())

	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	if cfg.DBDSN == "" {
		return nil, errors.New("dsn is required")
	}

	return &cfg, nil
}
