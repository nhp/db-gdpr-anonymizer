package anonymizer

import (
	"testing"

	"db-gdpr-anonymizer/internal/config"
)

func TestCreatePlan(t *testing.T) {
	// Create a test configuration
	cfg := &config.Config{
		Tables: map[string]config.TableConfig{
			"customer_entity": {
				Where: "entity_id > 1000",
				Columns: map[string]config.ColumnConfig{
					"email": {
						Type: "faker.email",
					},
					"firstname": {
						Value: "John",
					},
					"lastname": {
						Null: true,
					},
					"description": {
						Expr: "CONCAT('Customer ', entity_id)",
					},
				},
			},
			"sales_order": {
				Columns: map[string]config.ColumnConfig{
					"customer_email": {
						Type: "faker.email",
					},
				},
			},
		},
	}

	// Create the plan
	plan, err := CreatePlan(cfg)
	if err != nil {
		t.Fatalf("Failed to create plan: %v", err)
	}

	// Verify the plan
	if len(plan.Tables) != 2 {
		t.Errorf("Expected 2 tables in plan, got %d", len(plan.Tables))
	}

	// Find customer_entity table
	var customerTable *TablePlan
	for _, table := range plan.Tables {
		if table.Name == "customer_entity" {
			customerTable = table
			break
		}
	}

	if customerTable == nil {
		t.Fatalf("Expected 'customer_entity' table in plan")
	}

	// Verify customer_entity table properties
	if customerTable.Where != "entity_id > 1000" {
		t.Errorf("Expected 'customer_entity' where clause to be 'entity_id > 1000', got '%s'", customerTable.Where)
	}

	if len(customerTable.Columns) != 4 {
		t.Errorf("Expected 4 columns in 'customer_entity', got %d", len(customerTable.Columns))
	}

	// Verify email column strategy
	var emailColumn *ColumnPlan
	for _, col := range customerTable.Columns {
		if col.Name == "email" {
			emailColumn = col
			break
		}
	}

	if emailColumn == nil {
		t.Fatalf("Expected 'email' column in 'customer_entity'")
	}

	if emailColumn.Strategy.GetType() != "faker" {
		t.Errorf("Expected 'email' strategy type to be 'faker', got '%s'", emailColumn.Strategy.GetType())
	}

	// Verify firstname column strategy
	var firstnameColumn *ColumnPlan
	for _, col := range customerTable.Columns {
		if col.Name == "firstname" {
			firstnameColumn = col
			break
		}
	}

	if firstnameColumn == nil {
		t.Fatalf("Expected 'firstname' column in 'customer_entity'")
	}

	if firstnameColumn.Strategy.GetType() != "fixed" {
		t.Errorf("Expected 'firstname' strategy type to be 'fixed', got '%s'", firstnameColumn.Strategy.GetType())
	}

	// Verify lastname column strategy
	var lastnameColumn *ColumnPlan
	for _, col := range customerTable.Columns {
		if col.Name == "lastname" {
			lastnameColumn = col
			break
		}
	}

	if lastnameColumn == nil {
		t.Fatalf("Expected 'lastname' column in 'customer_entity'")
	}

	if lastnameColumn.Strategy.GetType() != "null" {
		t.Errorf("Expected 'lastname' strategy type to be 'null', got '%s'", lastnameColumn.Strategy.GetType())
	}

	// Verify description column strategy
	var descriptionColumn *ColumnPlan
	for _, col := range customerTable.Columns {
		if col.Name == "description" {
			descriptionColumn = col
			break
		}
	}

	if descriptionColumn == nil {
		t.Fatalf("Expected 'description' column in 'customer_entity'")
	}

	if descriptionColumn.Strategy.GetType() != "expr" {
		t.Errorf("Expected 'description' strategy type to be 'expr', got '%s'", descriptionColumn.Strategy.GetType())
	}
}

func TestCreateStrategy(t *testing.T) {
	// Test fixed value strategy
	fixedConfig := config.ColumnConfig{
		Value: "test",
	}
	strategy, err := createStrategy(fixedConfig)
	if err != nil {
		t.Fatalf("Failed to create fixed value strategy: %v", err)
	}
	if strategy.GetType() != "fixed" {
		t.Errorf("Expected strategy type to be 'fixed', got '%s'", strategy.GetType())
	}
	sql := strategy.GenerateSQL("table", "column")
	if sql != "'test'" {
		t.Errorf("Expected SQL to be ''test'', got '%s'", sql)
	}

	// Test null strategy
	nullConfig := config.ColumnConfig{
		Null: true,
	}
	strategy, err = createStrategy(nullConfig)
	if err != nil {
		t.Fatalf("Failed to create null strategy: %v", err)
	}
	if strategy.GetType() != "null" {
		t.Errorf("Expected strategy type to be 'null', got '%s'", strategy.GetType())
	}
	sql = strategy.GenerateSQL("table", "column")
	if sql != "NULL" {
		t.Errorf("Expected SQL to be 'NULL', got '%s'", sql)
	}

	// Test expression strategy
	exprConfig := config.ColumnConfig{
		Expr: "CONCAT('test', id)",
	}
	strategy, err = createStrategy(exprConfig)
	if err != nil {
		t.Fatalf("Failed to create expression strategy: %v", err)
	}
	if strategy.GetType() != "expr" {
		t.Errorf("Expected strategy type to be 'expr', got '%s'", strategy.GetType())
	}
	sql = strategy.GenerateSQL("table", "column")
	if sql != "CONCAT('test', id)" {
		t.Errorf("Expected SQL to be 'CONCAT('test', id)', got '%s'", sql)
	}

	// Test faker strategy
	fakerConfig := config.ColumnConfig{
		Type: "faker.email",
	}
	strategy, err = createStrategy(fakerConfig)
	if err != nil {
		t.Fatalf("Failed to create faker strategy: %v", err)
	}
	if strategy.GetType() != "faker" {
		t.Errorf("Expected strategy type to be 'faker', got '%s'", strategy.GetType())
	}
	sql = strategy.GenerateSQL("table", "column")
	if sql != "FAKER('email')" {
		t.Errorf("Expected SQL to be 'FAKER('email')', got '%s'", sql)
	}

	// Test unsupported strategy
	unsupportedConfig := config.ColumnConfig{
		Type: "unsupported",
	}
	_, err = createStrategy(unsupportedConfig)
	if err == nil {
		t.Error("Expected error for unsupported strategy, got nil")
	}
}