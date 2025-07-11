package database

import (
	"database/sql"
	"fmt"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
)

// Driver represents a database driver
type Driver string

const (
	// MySQL driver
	MySQL Driver = "mysql"
	// PostgreSQL driver
	PostgreSQL Driver = "postgres"
)

// Config holds database connection configuration
type Config struct {
	Driver   Driver
	Host     string
	Port     int
	User     string
	Password string
	Name     string
}

// Connect establishes a connection to the database
func Connect(config Config) (*sql.DB, error) {
	var dsn string

	switch config.Driver {
	case MySQL:
		dsn = fmt.Sprintf("%s:%s@tcp(%s:%d)/%s", config.User, config.Password, config.Host, config.Port, config.Name)
	case PostgreSQL:
		dsn = fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
			config.Host, config.Port, config.User, config.Password, config.Name)
	default:
		return nil, fmt.Errorf("unsupported database driver: %s", config.Driver)
	}

	db, err := sql.Open(string(config.Driver), dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Test the connection
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Set connection pool parameters
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)

	return db, nil
}

// GetPrimaryKey gets the primary key column for a table
func GetPrimaryKey(db *sql.DB, driver Driver, tableName string) (string, error) {
	var query string
	var primaryKey string

	switch driver {
	case MySQL:
		query = `
			SELECT COLUMN_NAME
			FROM INFORMATION_SCHEMA.KEY_COLUMN_USAGE
			WHERE TABLE_SCHEMA = DATABASE()
			AND TABLE_NAME = ?
			AND CONSTRAINT_NAME = 'PRIMARY'
			LIMIT 1
		`
		err := db.QueryRow(query, tableName).Scan(&primaryKey)
		if err != nil {
			if err == sql.ErrNoRows {
				return "", fmt.Errorf("no primary key found for table %s", tableName)
			}
			return "", err
		}
	case PostgreSQL:
		query = `
			SELECT a.attname
			FROM pg_index i
			JOIN pg_attribute a ON a.attrelid = i.indrelid AND a.attnum = ANY(i.indkey)
			WHERE i.indrelid = $1::regclass
			AND i.indisprimary
			LIMIT 1
		`
		err := db.QueryRow(query, tableName).Scan(&primaryKey)
		if err != nil {
			if err == sql.ErrNoRows {
				return "", fmt.Errorf("no primary key found for table %s", tableName)
			}
			return "", err
		}
	default:
		return "", fmt.Errorf("unsupported database driver: %s", driver)
	}

	return primaryKey, nil
}

// GetTableColumns gets the columns for a table
func GetTableColumns(db *sql.DB, driver Driver, tableName string) ([]string, error) {
	var query string
	var rows *sql.Rows
	var err error

	switch driver {
	case MySQL:
		query = `
			SELECT COLUMN_NAME
			FROM INFORMATION_SCHEMA.COLUMNS
			WHERE TABLE_SCHEMA = DATABASE()
			AND TABLE_NAME = ?
			ORDER BY ORDINAL_POSITION
		`
		rows, err = db.Query(query, tableName)
	case PostgreSQL:
		query = `
			SELECT column_name
			FROM information_schema.columns
			WHERE table_schema = 'public'
			AND table_name = $1
			ORDER BY ordinal_position
		`
		rows, err = db.Query(query, tableName)
	default:
		return nil, fmt.Errorf("unsupported database driver: %s", driver)
	}

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var columns []string
	for rows.Next() {
		var column string
		if err := rows.Scan(&column); err != nil {
			return nil, err
		}
		columns = append(columns, column)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return columns, nil
}