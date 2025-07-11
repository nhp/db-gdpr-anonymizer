package anonymizer

import (
	"fmt"
	"strings"
)

// SQLGenerator generates SQL statements for anonymization
type SQLGenerator struct {
	plan *AnonymizationPlan
}

// NewSQLGenerator creates a new SQL generator
func NewSQLGenerator(plan *AnonymizationPlan) *SQLGenerator {
	return &SQLGenerator{
		plan: plan,
	}
}

// GenerateTableSQL generates SQL statements for anonymizing a table
func (g *SQLGenerator) GenerateTableSQL(tablePlan *TablePlan) (string, error) {
	if len(tablePlan.Columns) == 0 {
		return "", fmt.Errorf("no columns to anonymize in table %s", tablePlan.Name)
	}

	// Build SET clause
	setClause := make([]string, 0, len(tablePlan.Columns))
	for _, column := range tablePlan.Columns {
		setClause = append(setClause, fmt.Sprintf(
			"%s = %s",
			column.Name,
			column.Strategy.GenerateSQL(tablePlan.Name, column.Name),
		))
	}

	// Build WHERE clause
	whereClause := ""
	if tablePlan.Where != "" {
		whereClause = fmt.Sprintf("WHERE %s", tablePlan.Where)
	}

	// Build LIMIT clause
	limitClause := ""
	if tablePlan.Limit > 0 {
		limitClause = fmt.Sprintf("LIMIT %d", tablePlan.Limit)
	}

	// Build ORDER BY clause
	orderByClause := ""
	if tablePlan.OrderBy != "" {
		orderByClause = fmt.Sprintf("ORDER BY %s", tablePlan.OrderBy)
	}

	// Build the complete SQL statement
	sql := fmt.Sprintf(
		"UPDATE %s SET %s %s %s %s",
		tablePlan.Name,
		strings.Join(setClause, ", "),
		whereClause,
		orderByClause,
		limitClause,
	)

	return sql, nil
}

// GenerateCountSQL generates SQL statements for counting rows that will be anonymized
func (g *SQLGenerator) GenerateCountSQL(tablePlan *TablePlan) string {
	whereClause := ""
	if tablePlan.Where != "" {
		whereClause = fmt.Sprintf("WHERE %s", tablePlan.Where)
	}

	return fmt.Sprintf("SELECT COUNT(*) FROM %s %s", tablePlan.Name, whereClause)
}

// GenerateChunkedSQL generates SQL statements for anonymizing a table in chunks
func (g *SQLGenerator) GenerateChunkedSQL(tablePlan *TablePlan, primaryKey string, chunkSize int, offset int) (string, error) {
	if len(tablePlan.Columns) == 0 {
		return "", fmt.Errorf("no columns to anonymize in table %s", tablePlan.Name)
	}

	if primaryKey == "" {
		return "", fmt.Errorf("primary key is required for chunked anonymization")
	}

	// Build SET clause
	setClause := make([]string, 0, len(tablePlan.Columns))
	for _, column := range tablePlan.Columns {
		setClause = append(setClause, fmt.Sprintf(
			"%s = %s",
			column.Name,
			column.Strategy.GenerateSQL(tablePlan.Name, column.Name),
		))
	}

	// Build WHERE clause with chunk boundaries
	whereClause := fmt.Sprintf("%s >= %d AND %s < %d", primaryKey, offset, primaryKey, offset+chunkSize)
	if tablePlan.Where != "" {
		whereClause = fmt.Sprintf("(%s) AND %s", whereClause, tablePlan.Where)
	}

	// Build the complete SQL statement
	sql := fmt.Sprintf(
		"UPDATE %s SET %s WHERE %s",
		tablePlan.Name,
		strings.Join(setClause, ", "),
		whereClause,
	)

	return sql, nil
}

// GeneratePrimaryKeyRangeSQL generates SQL to get the min and max primary key values
func (g *SQLGenerator) GeneratePrimaryKeyRangeSQL(tableName string, primaryKey string, whereClause string) string {
	where := ""
	if whereClause != "" {
		where = fmt.Sprintf("WHERE %s", whereClause)
	}

	return fmt.Sprintf(
		"SELECT MIN(%s), MAX(%s) FROM %s %s",
		primaryKey,
		primaryKey,
		tableName,
		where,
	)
}

// GenerateSingleRowSQL generates SQL for anonymizing a single row
func (g *SQLGenerator) GenerateSingleRowSQL(tablePlan *TablePlan, primaryKey string, pkValue int) (string, error) {
	if len(tablePlan.Columns) == 0 {
		return "", fmt.Errorf("no columns to anonymize in table %s", tablePlan.Name)
	}

	if primaryKey == "" {
		return "", fmt.Errorf("primary key is required for single row anonymization")
	}

	// Build SET clause
	setClause := make([]string, 0, len(tablePlan.Columns))
	for _, column := range tablePlan.Columns {
		// For FakerStrategy, we need to include the primary key value in the placeholder
		// to ensure each row gets unique fake data
		if strategy, ok := column.Strategy.(*FakerStrategy); ok {
			// Create a custom placeholder that includes the primary key value
			placeholder := fmt.Sprintf("'[FAKER:%s:%s:%s:%d]'", strategy.FakerType, tablePlan.Name, column.Name, pkValue)
			setClause = append(setClause, fmt.Sprintf(
				"%s = %s",
				column.Name,
				placeholder,
			))
		} else {
			// For other strategies, use the normal GenerateSQL method
			setClause = append(setClause, fmt.Sprintf(
				"%s = %s",
				column.Name,
				column.Strategy.GenerateSQL(tablePlan.Name, column.Name),
			))
		}
	}

	// Build WHERE clause for a single row
	whereClause := fmt.Sprintf("%s = %d", primaryKey, pkValue)
	if tablePlan.Where != "" {
		whereClause = fmt.Sprintf("(%s) AND %s", whereClause, tablePlan.Where)
	}

	// Build the complete SQL statement
	sql := fmt.Sprintf(
		"UPDATE %s SET %s WHERE %s",
		tablePlan.Name,
		strings.Join(setClause, ", "),
		whereClause,
	)

	return sql, nil
}
