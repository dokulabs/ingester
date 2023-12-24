package config

import (
	"fmt"
	"gopkg.in/yaml.v2"
	"os"
	"strconv"
)

type Configuration struct {
	IngesterPort string `yaml:"ingesterPort"`
	PricingInfo  struct {
		LocalFile struct {
			Path string `yaml:"path"`
		} `yaml:"localFile"`
		URL string `yaml:"url"`
	} `yaml:"pricingInfo"`
	DBConfig struct {
		DBName          string `yaml:"name"`
		DBUser          string `yaml:"username"`
		DBPassword      string `yaml:"password"`
		DBHost          string `yaml:"host"`
		DBPort          string `yaml:"port"`
		DBSSLMode       string `yaml:"sslMode"`
		MaxOpenConns    int    `yaml:"maxOpenConns"`
		MaxIdleConns    int    `yaml:"maxIdleConns"`
		DataTableName   string `yaml:"dataTable"`
		APIKeyTableName string `yaml:"apiKeyTable"`
	} `yaml:"dbConfig"`
	ObservabilityPlatform struct {
		Enabled      bool `yaml:"enabled"`
		GrafanaCloud struct {
			LogsURL          string `yaml:"logsUrl"`
			LogsUsername     string `yaml:"logsUsername"`
			CloudAccessToken string `yaml:"cloudAccessToken"`
		} `yaml:"grafanaCloud"`
	} `yaml:"observabilityPlatform"`
}

func validateConfig(cfg *Configuration) error {
	if _, err := strconv.Atoi(cfg.IngesterPort); err != nil {
		return fmt.Errorf("Ingester Port is not defined")
	}

	// Check if at least one PricingInfo configuration is set.
	if cfg.PricingInfo.LocalFile.Path == "" && cfg.PricingInfo.URL == "" {
		return fmt.Errorf("PricingInfo configuration is not defined")
	}

	// Check if both PricingInfo configurations are set.
	if cfg.PricingInfo.LocalFile.Path != "" && cfg.PricingInfo.URL != "" {
		return fmt.Errorf("Both LocalFile and URL configurations are defined in PricingInfo; only one is allowed")
	}

	return nil
}

func LoadConfiguration(configPath string) (*Configuration, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	var cfg Configuration
	err = yaml.Unmarshal(data, &cfg)
	if err != nil {
		return nil, err
	}

	// Validate the loaded configuration
	if err := validateConfig(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
