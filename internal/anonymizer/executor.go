package anonymizer

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	"db-gdpr-anonymizer/internal/faker"
	"db-gdpr-anonymizer/internal/logger"
)

// ExecutionResult represents the result of an anonymization operation
type ExecutionResult struct {
	TableName    string
	FieldName    string
	RowsScanned  int64
	RowsAffected int64
	Strategy     string
	Duration     time.Duration
	Error        error
}

// Executor executes the anonymization plan
type Executor struct {
	db         *sql.DB
	plan       *AnonymizationPlan
	sqlGen     *SQLGenerator
	logger     *logger.Logger
	dryRun     bool
	maxWorkers int
	faker      *faker.Generator
}

// NewExecutor creates a new executor
func NewExecutor(db *sql.DB, plan *AnonymizationPlan, logger *logger.Logger, dryRun bool, maxWorkers int) *Executor {
	return &Executor{
		db:         db,
		plan:       plan,
		sqlGen:     NewSQLGenerator(plan),
		logger:     logger,
		dryRun:     dryRun,
		maxWorkers: maxWorkers,
		faker:      faker.NewGenerator(),
	}
}

// Execute executes the anonymization plan
func (e *Executor) Execute(ctx context.Context) ([]ExecutionResult, error) {
	results := make([]ExecutionResult, 0)
	resultsMutex := &sync.Mutex{}

	// Create a worker pool
	workerPool := make(chan struct{}, e.maxWorkers)
	var wg sync.WaitGroup

	for _, tablePlan := range e.plan.Tables {
		// Check if context is cancelled
		select {
		case <-ctx.Done():
			return results, ctx.Err()
		default:
		}

		// Use the primary key from the plan if it's set, otherwise get it
		if tablePlan.PrimaryKey == "" {
			primaryKey, err := e.getPrimaryKey(tablePlan.Name)
			if err != nil {
				e.logger.Error("Failed to get primary key", map[string]interface{}{
					"table":  tablePlan.Name,
					"error":  err.Error(),
				})
				continue
			}
			tablePlan.PrimaryKey = primaryKey
		}

		// Count rows to be anonymized
		rowCount, err := e.countRows(tablePlan)
		if err != nil {
			e.logger.Error("Failed to count rows", map[string]interface{}{
				"table":  tablePlan.Name,
				"error":  err.Error(),
			})
			continue
		}

		e.logger.Info("Starting anonymization", map[string]interface{}{
			"table":     tablePlan.Name,
			"rowCount":  rowCount,
			"dryRun":    e.dryRun,
			"columns":   len(tablePlan.Columns),
		})

		// If the table has a primary key, we can process it in chunks
		if tablePlan.PrimaryKey != "" && rowCount > 1000 {
			// Get primary key range
			minPK, maxPK, err := e.getPrimaryKeyRange(tablePlan)
			if err != nil {
				e.logger.Error("Failed to get primary key range", map[string]interface{}{
					"table":  tablePlan.Name,
					"error":  err.Error(),
				})
				continue
			}

			// Process in chunks
			chunkSize := 1000
			for offset := minPK; offset <= maxPK; offset += chunkSize {
				// Limit concurrent workers
				workerPool <- struct{}{}
				wg.Add(1)

				go func(tablePlan *TablePlan, offset int) {
					defer func() {
						<-workerPool
						wg.Done()
					}()

					chunkResult, err := e.processChunk(ctx, tablePlan, offset, chunkSize)
					if err != nil {
						e.logger.Error("Failed to process chunk", map[string]interface{}{
							"table":  tablePlan.Name,
							"offset": offset,
							"error":  err.Error(),
						})
						return
					}

					resultsMutex.Lock()
					results = append(results, chunkResult...)
					resultsMutex.Unlock()
				}(tablePlan, offset)
			}
		} else {
			// Process the whole table at once
			tableResults, err := e.processTable(ctx, tablePlan)
			if err != nil {
				e.logger.Error("Failed to process table", map[string]interface{}{
					"table": tablePlan.Name,
					"error": err.Error(),
				})
				continue
			}

			resultsMutex.Lock()
			results = append(results, tableResults...)
			resultsMutex.Unlock()
		}
	}

	// Wait for all workers to finish
	wg.Wait()

	return results, nil
}

// processTable processes a whole table at once
func (e *Executor) processTable(ctx context.Context, tablePlan *TablePlan) ([]ExecutionResult, error) {
	results := make([]ExecutionResult, 0, len(tablePlan.Columns))

	// Generate SQL
	sqlQuery, err := e.sqlGen.GenerateTableSQL(tablePlan)
	if err != nil {
		return nil, err
	}

	// Replace faker placeholders with actual fake data
	sqlQuery, err = e.faker.ReplaceFakerPlaceholders(sqlQuery)
	if err != nil {
		return nil, err
	}

	// Count rows to be anonymized
	rowCount, err := e.countRows(tablePlan)
	if err != nil {
		return nil, err
	}

	// Execute SQL
	startTime := time.Now()
	var rowsAffected int64

	if !e.dryRun {
		result, err := e.db.ExecContext(ctx, sqlQuery)
		if err != nil {
			return nil, err
		}
		rowsAffected, _ = result.RowsAffected()
	} else {
		// In dry run mode, we simulate the number of rows affected
		rowsAffected = rowCount
	}

	duration := time.Since(startTime)

	// Log the operation
	e.logger.Info("Processed table", map[string]interface{}{
		"table":        tablePlan.Name,
		"rowsScanned":  rowCount,
		"rowsAffected": rowsAffected,
		"dryRun":       e.dryRun,
		"duration":     duration.String(),
	})

	// Create results for each column
	for _, column := range tablePlan.Columns {
		results = append(results, ExecutionResult{
			TableName:    tablePlan.Name,
			FieldName:    column.Name,
			RowsScanned:  rowCount,
			RowsAffected: rowsAffected,
			Strategy:     column.Strategy.GetType(),
			Duration:     duration,
		})
	}

	return results, nil
}

// processChunk processes a chunk of a table
func (e *Executor) processChunk(ctx context.Context, tablePlan *TablePlan, offset, chunkSize int) ([]ExecutionResult, error) {
	results := make([]ExecutionResult, 0, len(tablePlan.Columns))

	// Get the primary key values for this chunk
	primaryKeyValues, err := e.getPrimaryKeysInRange(tablePlan, offset, chunkSize)
	if err != nil {
		return nil, err
	}

	// Process each row individually to ensure unique fake data for each row
	startTime := time.Now()
	var totalRowsAffected int64

	for _, pkValue := range primaryKeyValues {
		// Generate SQL for a single row
		sqlQuery, err := e.sqlGen.GenerateSingleRowSQL(tablePlan, tablePlan.PrimaryKey, pkValue)
		if err != nil {
			return nil, err
		}

		// Replace faker placeholders with actual fake data
		// This ensures each row gets unique fake data
		sqlQuery, err = e.faker.ReplaceFakerPlaceholders(sqlQuery)
		if err != nil {
			return nil, err
		}

		// Execute SQL for this row
		if !e.dryRun {
			result, err := e.db.ExecContext(ctx, sqlQuery)
			if err != nil {
				return nil, err
			}
			rowsAffected, _ := result.RowsAffected()
			totalRowsAffected += rowsAffected
		} else {
			// In dry run mode, we estimate one row affected per query
			totalRowsAffected++
		}
	}

	duration := time.Since(startTime)

	// Log the operation
	e.logger.Info("Processed chunk", map[string]interface{}{
		"table":        tablePlan.Name,
		"offset":       offset,
		"chunkSize":    chunkSize,
		"rowsAffected": totalRowsAffected,
		"dryRun":       e.dryRun,
		"duration":     duration.String(),
	})

	// Create results for each column
	for _, column := range tablePlan.Columns {
		results = append(results, ExecutionResult{
			TableName:    tablePlan.Name,
			FieldName:    column.Name,
			RowsScanned:  int64(len(primaryKeyValues)),
			RowsAffected: totalRowsAffected,
			Strategy:     column.Strategy.GetType(),
			Duration:     duration,
		})
	}

	return results, nil
}

// countRows counts the number of rows that will be anonymized
func (e *Executor) countRows(tablePlan *TablePlan) (int64, error) {
	sql := e.sqlGen.GenerateCountSQL(tablePlan)
	var count int64
	err := e.db.QueryRow(sql).Scan(&count)
	return count, err
}

// getPrimaryKey gets the primary key column for a table
func (e *Executor) getPrimaryKey(tableName string) (string, error) {
	// First, check if the primary key is specified in the plan
	for _, tablePlan := range e.plan.Tables {
		if tablePlan.Name == tableName && tablePlan.PrimaryKey != "" {
			return tablePlan.PrimaryKey, nil
		}
	}

	// If not specified, fall back to "id" as a default
	// This is a simplified implementation and might not work for all databases
	// In a real implementation, you would query the database schema
	return "id", nil
}

// getPrimaryKeyRange gets the min and max values of the primary key
func (e *Executor) getPrimaryKeyRange(tablePlan *TablePlan) (int, int, error) {
	sql := e.sqlGen.GeneratePrimaryKeyRangeSQL(tablePlan.Name, tablePlan.PrimaryKey, tablePlan.Where)
	var min, max int
	err := e.db.QueryRow(sql).Scan(&min, &max)
	return min, max, err
}

// getPrimaryKeysInRange gets all primary key values in the specified range
func (e *Executor) getPrimaryKeysInRange(tablePlan *TablePlan, offset, chunkSize int) ([]int, error) {
	// Generate SQL to get primary keys in range
	sql := fmt.Sprintf(
		"SELECT %s FROM %s WHERE %s >= %d AND %s < %d",
		tablePlan.PrimaryKey,
		tablePlan.Name,
		tablePlan.PrimaryKey,
		offset,
		tablePlan.PrimaryKey,
		offset+chunkSize,
	)

	// Add additional WHERE clause if specified in the table plan
	if tablePlan.Where != "" {
		sql = fmt.Sprintf("%s AND %s", sql, tablePlan.Where)
	}

	// Execute the query
	rows, err := e.db.Query(sql)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Collect primary key values
	var primaryKeys []int
	for rows.Next() {
		var pk int
		if err := rows.Scan(&pk); err != nil {
			return nil, err
		}
		primaryKeys = append(primaryKeys, pk)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return primaryKeys, nil
}
