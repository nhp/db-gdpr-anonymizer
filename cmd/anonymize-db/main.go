package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"db-gdpr-anonymizer/internal/anonymizer"
	"db-gdpr-anonymizer/internal/config"
	"db-gdpr-anonymizer/internal/database"
	"db-gdpr-anonymizer/internal/logger"
	"db-gdpr-anonymizer/internal/report"
)

var (
	configFile string
	dryRun     bool
	reportType string
	logDir     string
	workers    int
)

func init() {
	flag.StringVar(&configFile, "config", "", "Path to YAML configuration file")
	flag.BoolVar(&dryRun, "dry-run", false, "Run in simulation mode without making changes")
	flag.StringVar(&reportType, "report", "text", "Final report format (json or text)")
	flag.StringVar(&logDir, "log", "logs", "Directory for log files")
	flag.IntVar(&workers, "workers", runtime.NumCPU(), "Number of parallel workers")
	flag.Parse()
}

func main() {
	// Validate command line arguments
	if configFile == "" {
		fmt.Println("Error: --config flag is required")
		flag.Usage()
		os.Exit(1)
	}

	if reportType != "json" && reportType != "text" {
		fmt.Println("Error: --report must be either 'json' or 'text'")
		flag.Usage()
		os.Exit(1)
	}

	// Ensure log directory exists
	if err := os.MkdirAll(logDir, 0755); err != nil {
		fmt.Printf("Error creating log directory: %v\n", err)
		os.Exit(1)
	}

	// Initialize logger
	log, err := logger.NewLogger(logDir, true)
	if err != nil {
		fmt.Printf("Error initializing logger: %v\n", err)
		os.Exit(1)
	}
	defer log.Close()

	// Initialize report generator
	reportGen := report.NewGenerator(dryRun, log.GetErrorLogPath())

	startTime := time.Now()

	log.Info("Starting anonymize-db", map[string]interface{}{
		"configFile": configFile,
		"dryRun":     dryRun,
		"reportType": reportType,
		"logDir":     logDir,
		"workers":    workers,
	})

	// 1. Parse configuration file
	cfg, err := config.LoadConfig(configFile)
	if err != nil {
		log.Error("Failed to load configuration", map[string]interface{}{
			"error": err.Error(),
		})
		os.Exit(1)
	}

	// 2. Connect to database (in dry run mode, we still connect to get schema information)
	dbConfig := database.Config{
		Driver:   database.Driver(cfg.Database.Driver),
		Host:     cfg.Database.Host,
		Port:     cfg.Database.Port,
		User:     cfg.Database.User,
		Password: cfg.Database.Password,
		Name:     cfg.Database.Name,
	}

	db, err := database.Connect(dbConfig)
	if err != nil {
		log.Error("Failed to connect to database", map[string]interface{}{
			"error": err.Error(),
		})
		os.Exit(1)
	}
	defer db.Close()

	// 3. Create anonymization plan
	plan, err := anonymizer.CreatePlan(cfg)
	if err != nil {
		log.Error("Failed to create anonymization plan", map[string]interface{}{
			"error": err.Error(),
		})
		os.Exit(1)
	}

	// 4. Execute anonymization plan
	executor := anonymizer.NewExecutor(db, plan, log, dryRun, workers)
	results, err := executor.Execute(context.Background())
	if err != nil {
		log.Error("Failed to execute anonymization plan", map[string]interface{}{
			"error": err.Error(),
		})
		os.Exit(1)
	}

	// 5. Generate report
	// Convert anonymizer.ExecutionResult to report.ExecutionResult
	reportResults := make([]report.ExecutionResult, len(results))
	for i, result := range results {
		reportResults[i] = report.ExecutionResult{
			TableName:    result.TableName,
			FieldName:    result.FieldName,
			RowsScanned:  result.RowsScanned,
			RowsAffected: result.RowsAffected,
			Strategy:     result.Strategy,
			Duration:     result.Duration,
			Error:        result.Error,
		}
	}
	finalReport := reportGen.GenerateReport(reportResults)

	// Output report
	if reportType == "json" {
		jsonFile := filepath.Join(logDir, "report.json")
		if err := reportGen.OutputJSON(finalReport, jsonFile); err != nil {
			log.Error("Failed to output JSON report", map[string]interface{}{
				"error": err.Error(),
			})
			os.Exit(1)
		}
		fmt.Printf("JSON report written to %s\n", jsonFile)
	} else {
		reportGen.OutputText(finalReport)
	}

	duration := time.Since(startTime)
	log.Info("Anonymization completed", map[string]interface{}{
		"duration":        duration.String(),
		"tablesProcessed": finalReport.Summary.TotalTables,
		"fieldsProcessed": finalReport.Summary.TotalFields,
		"rowsScanned":     finalReport.Summary.TotalRowsScanned,
		"rowsAffected":    finalReport.Summary.TotalRowsAffected,
	})

	fmt.Printf("Completed in %v\n", duration)
}
