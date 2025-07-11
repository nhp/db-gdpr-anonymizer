package report

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/olekukonko/tablewriter"
)

// ExecutionMode represents the mode of execution
type ExecutionMode string

const (
	// ModeReal represents a real execution
	ModeReal ExecutionMode = "real"
	// ModeDryRun represents a dry run execution
	ModeDryRun ExecutionMode = "dry-run"
)

// Report represents the final report of the anonymization process
type Report struct {
	Execution struct {
		Mode            ExecutionMode `json:"mode"`
		StartTime       time.Time     `json:"start_time"`
		EndTime         time.Time     `json:"end_time"`
		DurationSeconds int           `json:"duration_seconds"`
	} `json:"execution"`
	Summary struct {
		TotalTables       int   `json:"total_tables"`
		TotalFields       int   `json:"total_fields"`
		TotalRowsScanned  int64 `json:"total_rows_scanned"`
		TotalRowsAffected int64 `json:"total_rows_affected"`
	} `json:"summary"`
	Tables []TableReport `json:"tables"`
	Errors struct {
		Count   int    `json:"count"`
		LogFile string `json:"log_file"`
	} `json:"errors"`
}

// TableReport represents the report for a single table
type TableReport struct {
	Name         string        `json:"name"`
	RowsScanned  int64         `json:"rows_scanned"`
	RowsAffected int64         `json:"rows_affected"`
	Fields       []FieldReport `json:"fields"`
}

// FieldReport represents the report for a single field
type FieldReport struct {
	Name         string `json:"name"`
	Strategy     string `json:"strategy"`
	RowsAffected int64  `json:"rows_affected"`
}

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

// Generator generates reports
type Generator struct {
	startTime time.Time
	dryRun    bool
	errorLog  string
}

// NewGenerator creates a new report generator
func NewGenerator(dryRun bool, errorLog string) *Generator {
	return &Generator{
		startTime: time.Now(),
		dryRun:    dryRun,
		errorLog:  errorLog,
	}
}

// GenerateReport generates a report from the execution results
func (g *Generator) GenerateReport(results []ExecutionResult) *Report {
	report := &Report{}

	// Set execution information
	if g.dryRun {
		report.Execution.Mode = ModeDryRun
	} else {
		report.Execution.Mode = ModeReal
	}
	report.Execution.StartTime = g.startTime
	report.Execution.EndTime = time.Now()
	report.Execution.DurationSeconds = int(report.Execution.EndTime.Sub(report.Execution.StartTime).Seconds())

	// Group results by table
	tableMap := make(map[string]*TableReport)
	errorCount := 0

	for _, result := range results {
		if result.Error != nil {
			errorCount++
			continue
		}

		// Get or create table report
		tableReport, ok := tableMap[result.TableName]
		if !ok {
			tableReport = &TableReport{
				Name:         result.TableName,
				RowsScanned:  result.RowsScanned,
				RowsAffected: result.RowsAffected,
				Fields:       make([]FieldReport, 0),
			}
			tableMap[result.TableName] = tableReport
		}

		// Add field report
		tableReport.Fields = append(tableReport.Fields, FieldReport{
			Name:         result.FieldName,
			Strategy:     result.Strategy,
			RowsAffected: result.RowsAffected,
		})
	}

	// Convert map to slice
	report.Tables = make([]TableReport, 0, len(tableMap))
	for _, tableReport := range tableMap {
		report.Tables = append(report.Tables, *tableReport)
	}

	// Calculate summary
	report.Summary.TotalTables = len(report.Tables)
	for _, table := range report.Tables {
		report.Summary.TotalFields += len(table.Fields)
		report.Summary.TotalRowsScanned += table.RowsScanned
		report.Summary.TotalRowsAffected += table.RowsAffected
	}

	// Set error information
	report.Errors.Count = errorCount
	report.Errors.LogFile = g.errorLog

	return report
}

// OutputJSON outputs the report as JSON
func (g *Generator) OutputJSON(report *Report, outputFile string) error {
	jsonData, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal report to JSON: %w", err)
	}

	if outputFile != "" {
		if err := os.WriteFile(outputFile, jsonData, 0644); err != nil {
			return fmt.Errorf("failed to write JSON report to file: %w", err)
		}
	} else {
		fmt.Println(string(jsonData))
	}

	return nil
}

// OutputText outputs the report as text
func (g *Generator) OutputText(report *Report) {
	// Print execution information
	fmt.Println("=== Anonymization Report ===")
	fmt.Printf("Mode: %s\n", report.Execution.Mode)
	fmt.Printf("Start Time: %s\n", report.Execution.StartTime.Format(time.RFC3339))
	fmt.Printf("End Time: %s\n", report.Execution.EndTime.Format(time.RFC3339))
	fmt.Printf("Duration: %d seconds\n", report.Execution.DurationSeconds)
	fmt.Println()

	// Print summary
	fmt.Println("=== Summary ===")
	fmt.Printf("Total Tables: %d\n", report.Summary.TotalTables)
	fmt.Printf("Total Fields: %d\n", report.Summary.TotalFields)
	fmt.Printf("Total Rows Scanned: %d\n", report.Summary.TotalRowsScanned)
	fmt.Printf("Total Rows Affected: %d\n", report.Summary.TotalRowsAffected)
	fmt.Println()

	// Print table information
	fmt.Println("=== Tables ===")
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Table", "Field", "Strategy", "Rows Affected"})
	table.SetBorder(false)
	table.SetColumnSeparator("|")

	for _, tableReport := range report.Tables {
		for i, fieldReport := range tableReport.Fields {
			tableName := tableReport.Name
			if i > 0 {
				tableName = ""
			}

			table.Append([]string{
				tableName,
				fieldReport.Name,
				fieldReport.Strategy,
				fmt.Sprintf("%d", fieldReport.RowsAffected),
			})
		}
		table.Append([]string{"", "", "", ""})
	}

	table.Render()
	fmt.Println()

	// Print error information
	fmt.Println("=== Errors ===")
	fmt.Printf("Error Count: %d\n", report.Errors.Count)
	if report.Errors.Count > 0 {
		fmt.Printf("Error Log: %s\n", report.Errors.LogFile)
	}
}
