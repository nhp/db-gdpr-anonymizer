package config

import (
	"os"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	// Create a temporary config file
	configContent := `
database:
  host: localhost
  port: 3306
  user: testuser
  password: testpass
  name: testdb
  driver: mysql

tables:
  customer_entity:
    columns:
      email:
        type: faker.email
      firstname:
        type: faker.firstname
      lastname:
        type: faker.lastname
  
  sales_order:
    where: "entity_id > 1000"
    columns:
      customer_email:
        type: faker.email
      customer_firstname:
        value: "John"
      customer_lastname:
        null: true
`
	tmpFile, err := os.CreateTemp("", "config-*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.Write([]byte(configContent)); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	if err := tmpFile.Close(); err != nil {
		t.Fatalf("Failed to close temp file: %v", err)
	}

	// Test loading the config
	cfg, err := LoadConfig(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Verify database config
	if cfg.Database.Host != "localhost" {
		t.Errorf("Expected database host to be 'localhost', got '%s'", cfg.Database.Host)
	}
	if cfg.Database.Port != 3306 {
		t.Errorf("Expected database port to be 3306, got %d", cfg.Database.Port)
	}
	if cfg.Database.User != "testuser" {
		t.Errorf("Expected database user to be 'testuser', got '%s'", cfg.Database.User)
	}
	if cfg.Database.Password != "testpass" {
		t.Errorf("Expected database password to be 'testpass', got '%s'", cfg.Database.Password)
	}
	if cfg.Database.Name != "testdb" {
		t.Errorf("Expected database name to be 'testdb', got '%s'", cfg.Database.Name)
	}
	if cfg.Database.Driver != "mysql" {
		t.Errorf("Expected database driver to be 'mysql', got '%s'", cfg.Database.Driver)
	}

	// Verify tables config
	if len(cfg.Tables) != 2 {
		t.Errorf("Expected 2 tables, got %d", len(cfg.Tables))
	}

	// Verify customer_entity table
	customerTable, ok := cfg.Tables["customer_entity"]
	if !ok {
		t.Fatalf("Expected 'customer_entity' table to exist")
	}
	if len(customerTable.Columns) != 3 {
		t.Errorf("Expected 3 columns in 'customer_entity', got %d", len(customerTable.Columns))
	}
	emailCol, ok := customerTable.Columns["email"]
	if !ok {
		t.Fatalf("Expected 'email' column to exist in 'customer_entity'")
	}
	if emailCol.Type != "faker.email" {
		t.Errorf("Expected 'email' column type to be 'faker.email', got '%s'", emailCol.Type)
	}

	// Verify sales_order table
	salesTable, ok := cfg.Tables["sales_order"]
	if !ok {
		t.Fatalf("Expected 'sales_order' table to exist")
	}
	if salesTable.Where != "entity_id > 1000" {
		t.Errorf("Expected 'sales_order' where clause to be 'entity_id > 1000', got '%s'", salesTable.Where)
	}
	if len(salesTable.Columns) != 3 {
		t.Errorf("Expected 3 columns in 'sales_order', got %d", len(salesTable.Columns))
	}
	firstnameCol, ok := salesTable.Columns["customer_firstname"]
	if !ok {
		t.Fatalf("Expected 'customer_firstname' column to exist in 'sales_order'")
	}
	if firstnameCol.Value != "John" {
		t.Errorf("Expected 'customer_firstname' value to be 'John', got '%v'", firstnameCol.Value)
	}
	lastnameCol, ok := salesTable.Columns["customer_lastname"]
	if !ok {
		t.Fatalf("Expected 'customer_lastname' column to exist in 'sales_order'")
	}
	if !lastnameCol.Null {
		t.Errorf("Expected 'customer_lastname' to be null")
	}
}

func TestValidateConfig(t *testing.T) {
	// Test missing database host
	cfg := &Config{
		Database: DatabaseConfig{
			User: "user",
			Name: "name",
		},
		Tables: map[string]TableConfig{
			"test": {},
		},
	}
	if err := validateConfig(cfg); err == nil {
		t.Error("Expected error for missing database host, got nil")
	}

	// Test missing database user
	cfg = &Config{
		Database: DatabaseConfig{
			Host: "host",
			Name: "name",
		},
		Tables: map[string]TableConfig{
			"test": {},
		},
	}
	if err := validateConfig(cfg); err == nil {
		t.Error("Expected error for missing database user, got nil")
	}

	// Test missing database name
	cfg = &Config{
		Database: DatabaseConfig{
			Host: "host",
			User: "user",
		},
		Tables: map[string]TableConfig{
			"test": {},
		},
	}
	if err := validateConfig(cfg); err == nil {
		t.Error("Expected error for missing database name, got nil")
	}

	// Test missing tables
	cfg = &Config{
		Database: DatabaseConfig{
			Host: "host",
			User: "user",
			Name: "name",
		},
		Tables: map[string]TableConfig{},
	}
	if err := validateConfig(cfg); err == nil {
		t.Error("Expected error for missing tables, got nil")
	}

	// Test valid config
	cfg = &Config{
		Database: DatabaseConfig{
			Host: "host",
			User: "user",
			Name: "name",
		},
		Tables: map[string]TableConfig{
			"test": {},
		},
	}
	if err := validateConfig(cfg); err != nil {
		t.Errorf("Expected no error for valid config, got %v", err)
	}
}

func TestGetDSN(t *testing.T) {
	// Test MySQL DSN
	dbConfig := &DatabaseConfig{
		Driver:   "mysql",
		Host:     "localhost",
		Port:     3306,
		User:     "user",
		Password: "pass",
		Name:     "dbname",
	}
	expectedDSN := "user:pass@tcp(localhost:3306)/dbname"
	if dsn := dbConfig.GetDSN(); dsn != expectedDSN {
		t.Errorf("Expected MySQL DSN to be '%s', got '%s'", expectedDSN, dsn)
	}

	// Test PostgreSQL DSN
	dbConfig = &DatabaseConfig{
		Driver:   "postgres",
		Host:     "localhost",
		Port:     5432,
		User:     "user",
		Password: "pass",
		Name:     "dbname",
	}
	expectedDSN = "host=localhost port=5432 user=user password=pass dbname=dbname sslmode=disable"
	if dsn := dbConfig.GetDSN(); dsn != expectedDSN {
		t.Errorf("Expected PostgreSQL DSN to be '%s', got '%s'", expectedDSN, dsn)
	}

	// Test unsupported driver
	dbConfig = &DatabaseConfig{
		Driver: "unsupported",
	}
	if dsn := dbConfig.GetDSN(); dsn != "" {
		t.Errorf("Expected empty DSN for unsupported driver, got '%s'", dsn)
	}
}