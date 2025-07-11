package config

import (
	"encoding/json"
	"fmt"
	"os"
)

// Config represents the top-level configuration structure
type Config struct {
	Database   DatabaseConfig             `json:"database"`
	Tables     map[string]TableConfig     `json:"tables"`
	Converters map[string]ConverterConfig `json:"converters,omitempty"`
}

// DatabaseConfig holds database connection information
type DatabaseConfig struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	User     string `json:"user"`
	Password string `json:"password"`
	Name     string `json:"name"`
	Driver   string `json:"driver"`
}

// TableConfig defines anonymization rules for a specific table
type TableConfig struct {
	Truncate   bool                    `json:"truncate,omitempty"`
	Where      string                  `json:"where,omitempty"`
	Limit      int                     `json:"limit,omitempty"`
	OrderBy    string                  `json:"order_by,omitempty"`
	PrimaryKey string                  `json:"primary_key,omitempty"`
	Columns    map[string]ColumnConfig `json:"columns,omitempty"`
}

// ColumnConfig defines how a specific column should be anonymized
type ColumnConfig struct {
	Type      string      `json:"type"`
	Formatter string      `json:"formatter,omitempty"`
	Value     interface{} `json:"value,omitempty"`
	Expr      string      `json:"expr,omitempty"`
	Null      bool        `json:"null,omitempty"`
}

// ConverterConfig defines custom converters
type ConverterConfig struct {
	Type   string                 `json:"type"`
	Params map[string]interface{} `json:"params,omitempty"`
}

// LoadConfig loads and parses the JSON configuration file
func LoadConfig(filePath string) (*Config, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %v", err)
	}

	if err := validateConfig(&config); err != nil {
		return nil, err
	}

	return &config, nil
}

// validateConfig checks if the configuration is valid
func validateConfig(config *Config) error {
	// Check database configuration
	if config.Database.Host == "" {
		return fmt.Errorf("database host is required")
	}
	if config.Database.User == "" {
		return fmt.Errorf("database user is required")
	}
	if config.Database.Name == "" {
		return fmt.Errorf("database name is required")
	}
	if config.Database.Driver == "" {
		// Default to MySQL if not specified
		config.Database.Driver = "mysql"
	}
	if config.Database.Port == 0 {
		// Set default port based on driver
		switch config.Database.Driver {
		case "mysql":
			config.Database.Port = 3306
		case "postgres":
			config.Database.Port = 5432
		default:
			return fmt.Errorf("unsupported database driver: %s", config.Database.Driver)
		}
	}

	// Check if there are tables to anonymize
	if len(config.Tables) == 0 {
		return fmt.Errorf("no tables specified for anonymization")
	}

	return nil
}

// GetDSN returns the data source name for database connection
func (c *DatabaseConfig) GetDSN() string {
	switch c.Driver {
	case "mysql":
		return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s", c.User, c.Password, c.Host, c.Port, c.Name)
	case "postgres":
		return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
			c.Host, c.Port, c.User, c.Password, c.Name)
	default:
		return ""
	}
}
