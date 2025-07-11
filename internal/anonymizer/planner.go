package anonymizer

import (
	"fmt"
	"strings"

	"db-gdpr-anonymizer/internal/config"
)

// AnonymizationPlan represents the plan for anonymizing the database
type AnonymizationPlan struct {
	Tables []*TablePlan
}

// TablePlan represents the plan for anonymizing a single table
type TablePlan struct {
	Name       string
	PrimaryKey string
	Where      string
	Limit      int
	OrderBy    string
	Columns    []*ColumnPlan
}

// ColumnPlan represents the plan for anonymizing a single column
type ColumnPlan struct {
	Name      string
	Strategy  AnonymizationStrategy
	Formatter string
}

// AnonymizationStrategy defines how a column should be anonymized
type AnonymizationStrategy interface {
	// GenerateSQL generates the SQL expression for anonymizing the column
	GenerateSQL(tableName, columnName string) string
	// GetType returns the type of the strategy
	GetType() string
}

// FixedValueStrategy sets a fixed value for the column
type FixedValueStrategy struct {
	Value interface{}
}

// GenerateSQL implements AnonymizationStrategy.GenerateSQL
func (s *FixedValueStrategy) GenerateSQL(tableName, columnName string) string {
	switch v := s.Value.(type) {
	case string:
		return fmt.Sprintf("'%s'", strings.ReplaceAll(v, "'", "''"))
	case nil:
		return "NULL"
	default:
		return fmt.Sprintf("%v", v)
	}
}

// GetType implements AnonymizationStrategy.GetType
func (s *FixedValueStrategy) GetType() string {
	return "fixed"
}

// NullStrategy sets the column to NULL
type NullStrategy struct{}

// GenerateSQL implements AnonymizationStrategy.GenerateSQL
func (s *NullStrategy) GenerateSQL(tableName, columnName string) string {
	return "NULL"
}

// GetType implements AnonymizationStrategy.GetType
func (s *NullStrategy) GetType() string {
	return "null"
}

// ExpressionStrategy uses a SQL expression to anonymize the column
type ExpressionStrategy struct {
	Expression string
}

// GenerateSQL implements AnonymizationStrategy.GenerateSQL
func (s *ExpressionStrategy) GenerateSQL(tableName, columnName string) string {
	return s.Expression
}

// GetType implements AnonymizationStrategy.GetType
func (s *ExpressionStrategy) GetType() string {
	return "expr"
}

// FakerStrategy uses a faker function to generate fake data
type FakerStrategy struct {
	FakerType string
}

// GenerateSQL implements AnonymizationStrategy.GenerateSQL
func (s *FakerStrategy) GenerateSQL(tableName, columnName string) string {
	// Use a placeholder that will be replaced with actual fake data at runtime
	// Include the table name and column name to make each placeholder unique
	return fmt.Sprintf("'[FAKER:%s:%s:%s]'", s.FakerType, tableName, columnName)
}

// GetType implements AnonymizationStrategy.GetType
func (s *FakerStrategy) GetType() string {
	return "faker"
}

// CreatePlan creates an anonymization plan from the configuration
func CreatePlan(cfg *config.Config) (*AnonymizationPlan, error) {
	plan := &AnonymizationPlan{
		Tables: make([]*TablePlan, 0, len(cfg.Tables)),
	}

	for tableName, tableConfig := range cfg.Tables {
		tablePlan := &TablePlan{
			Name:       tableName,
			PrimaryKey: tableConfig.PrimaryKey,
			Where:      tableConfig.Where,
			Limit:      tableConfig.Limit,
			OrderBy:    tableConfig.OrderBy,
			Columns:    make([]*ColumnPlan, 0, len(tableConfig.Columns)),
		}

		for columnName, columnConfig := range tableConfig.Columns {
			strategy, err := createStrategy(columnConfig)
			if err != nil {
				return nil, fmt.Errorf("error creating strategy for %s.%s: %w", tableName, columnName, err)
			}

			columnPlan := &ColumnPlan{
				Name:      columnName,
				Strategy:  strategy,
				Formatter: columnConfig.Formatter,
			}

			tablePlan.Columns = append(tablePlan.Columns, columnPlan)
		}

		plan.Tables = append(plan.Tables, tablePlan)
	}

	return plan, nil
}

// createStrategy creates an anonymization strategy from the column configuration
func createStrategy(columnConfig config.ColumnConfig) (AnonymizationStrategy, error) {
	if columnConfig.Null {
		return &NullStrategy{}, nil
	}

	if columnConfig.Expr != "" {
		return &ExpressionStrategy{Expression: columnConfig.Expr}, nil
	}

	if columnConfig.Value != nil {
		return &FixedValueStrategy{Value: columnConfig.Value}, nil
	}

	if strings.HasPrefix(columnConfig.Type, "faker.") {
		fakerType := strings.TrimPrefix(columnConfig.Type, "faker.")
		return &FakerStrategy{FakerType: fakerType}, nil
	}

	return nil, fmt.Errorf("unsupported anonymization strategy: %s", columnConfig.Type)
}
