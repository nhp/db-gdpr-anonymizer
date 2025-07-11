package anonymizer

import (
	"strings"
	"testing"
)

func TestGenerateTableSQL(t *testing.T) {
	// Create a test plan
	plan := &AnonymizationPlan{
		Tables: []*TablePlan{
			{
				Name:  "customer_entity",
				Where: "entity_id > 1000",
				Columns: []*ColumnPlan{
					{
						Name:     "email",
						Strategy: &FakerStrategy{FakerType: "email"},
					},
					{
						Name:     "firstname",
						Strategy: &FixedValueStrategy{Value: "John"},
					},
					{
						Name:     "lastname",
						Strategy: &NullStrategy{},
					},
				},
			},
		},
	}

	// Create SQL generator
	generator := NewSQLGenerator(plan)

	// Generate SQL for the table
	sql, err := generator.GenerateTableSQL(plan.Tables[0])
	if err != nil {
		t.Fatalf("Failed to generate SQL: %v", err)
	}

	// Verify SQL
	expectedParts := []string{
		"UPDATE customer_entity SET",
		"email = FAKER('email')",
		"firstname = 'John'",
		"lastname = NULL",
		"WHERE entity_id > 1000",
	}

	for _, part := range expectedParts {
		if !strings.Contains(sql, part) {
			t.Errorf("Expected SQL to contain '%s', but it doesn't: %s", part, sql)
		}
	}
}

func TestGenerateChunkedSQL(t *testing.T) {
	// Create a test plan
	tablePlan := &TablePlan{
		Name:       "customer_entity",
		PrimaryKey: "entity_id",
		Where:      "is_active = 1",
		Columns: []*ColumnPlan{
			{
				Name:     "email",
				Strategy: &FakerStrategy{FakerType: "email"},
			},
			{
				Name:     "firstname",
				Strategy: &FixedValueStrategy{Value: "John"},
			},
		},
	}

	// Create SQL generator
	generator := NewSQLGenerator(&AnonymizationPlan{})

	// Generate chunked SQL
	sql, err := generator.GenerateChunkedSQL(tablePlan, "entity_id", 1000, 5000)
	if err != nil {
		t.Fatalf("Failed to generate chunked SQL: %v", err)
	}

	// Verify SQL
	expectedParts := []string{
		"UPDATE customer_entity SET",
		"email = FAKER('email')",
		"firstname = 'John'",
		"WHERE (entity_id >= 5000 AND entity_id < 6000) AND is_active = 1",
	}

	for _, part := range expectedParts {
		if !strings.Contains(sql, part) {
			t.Errorf("Expected SQL to contain '%s', but it doesn't: %s", part, sql)
		}
	}
}

func TestGenerateCountSQL(t *testing.T) {
	// Create a test plan
	tablePlan := &TablePlan{
		Name:  "customer_entity",
		Where: "is_active = 1",
	}

	// Create SQL generator
	generator := NewSQLGenerator(&AnonymizationPlan{})

	// Generate count SQL
	sql := generator.GenerateCountSQL(tablePlan)

	// Verify SQL
	expected := "SELECT COUNT(*) FROM customer_entity WHERE is_active = 1"
	if sql != expected {
		t.Errorf("Expected SQL to be '%s', got '%s'", expected, sql)
	}

	// Test without where clause
	tablePlan.Where = ""
	sql = generator.GenerateCountSQL(tablePlan)
	expected = "SELECT COUNT(*) FROM customer_entity "
	if sql != expected {
		t.Errorf("Expected SQL to be '%s', got '%s'", expected, sql)
	}
}

func TestGeneratePrimaryKeyRangeSQL(t *testing.T) {
	// Create SQL generator
	generator := NewSQLGenerator(&AnonymizationPlan{})

	// Generate primary key range SQL
	sql := generator.GeneratePrimaryKeyRangeSQL("customer_entity", "entity_id", "is_active = 1")

	// Verify SQL
	expected := "SELECT MIN(entity_id), MAX(entity_id) FROM customer_entity WHERE is_active = 1"
	if sql != expected {
		t.Errorf("Expected SQL to be '%s', got '%s'", expected, sql)
	}

	// Test without where clause
	sql = generator.GeneratePrimaryKeyRangeSQL("customer_entity", "entity_id", "")
	expected = "SELECT MIN(entity_id), MAX(entity_id) FROM customer_entity "
	if sql != expected {
		t.Errorf("Expected SQL to be '%s', got '%s'", expected, sql)
	}
}

func TestFixedValueStrategyGenerateSQL(t *testing.T) {
	// Test with string value
	strategy := &FixedValueStrategy{Value: "test"}
	sql := strategy.GenerateSQL("table", "column")
	if sql != "'test'" {
		t.Errorf("Expected SQL to be ''test'', got '%s'", sql)
	}

	// Test with numeric value
	strategy = &FixedValueStrategy{Value: 123}
	sql = strategy.GenerateSQL("table", "column")
	if sql != "123" {
		t.Errorf("Expected SQL to be '123', got '%s'", sql)
	}

	// Test with nil value
	strategy = &FixedValueStrategy{Value: nil}
	sql = strategy.GenerateSQL("table", "column")
	if sql != "NULL" {
		t.Errorf("Expected SQL to be 'NULL', got '%s'", sql)
	}

	// Test with string containing quotes
	strategy = &FixedValueStrategy{Value: "O'Reilly"}
	sql = strategy.GenerateSQL("table", "column")
	if sql != "'O''Reilly'" {
		t.Errorf("Expected SQL to be ''O''Reilly'', got '%s'", sql)
	}
}

func TestNullStrategyGenerateSQL(t *testing.T) {
	strategy := &NullStrategy{}
	sql := strategy.GenerateSQL("table", "column")
	if sql != "NULL" {
		t.Errorf("Expected SQL to be 'NULL', got '%s'", sql)
	}
}

func TestExpressionStrategyGenerateSQL(t *testing.T) {
	strategy := &ExpressionStrategy{Expression: "CONCAT('test', id)"}
	sql := strategy.GenerateSQL("table", "column")
	if sql != "CONCAT('test', id)" {
		t.Errorf("Expected SQL to be 'CONCAT('test', id)', got '%s'", sql)
	}
}

func TestFakerStrategyGenerateSQL(t *testing.T) {
	strategy := &FakerStrategy{FakerType: "email"}
	sql := strategy.GenerateSQL("table", "column")
	if sql != "FAKER('email')" {
		t.Errorf("Expected SQL to be 'FAKER('email')', got '%s'", sql)
	}
}