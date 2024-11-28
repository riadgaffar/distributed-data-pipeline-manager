package execute_pipeline

import (
	"distributed-data-pipeline-manager/src/config"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// CommandExecutor abstracts command execution for testability.
type CommandExecutor interface {
	Execute(name string, args ...string) error
}

// RealCommandExecutor implements CommandExecutor and uses exec.Command.
type RealCommandExecutor struct{}

// Execute runs the actual command using exec.Command.
func (r *RealCommandExecutor) Execute(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// ExecutePipeline runs the pipeline using the provided CommandExecutor.
// ExecutePipeline generates a dynamic pipeline configuration and runs it using rpk.
func ExecutePipeline(configPath string, executor CommandExecutor) error {
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	fmt.Printf("DEBUG: Generating pipeline configuration from: %s\n", configPath)

	// Path for the dynamically generated pipeline file
	generatedPipelinePath := cfg.App.PipelineTemplatePath + "generated-pipeline.yaml"

	// Generate the pipeline configuration
	err = GeneratePipelineFile(configPath, generatedPipelinePath)
	if err != nil {
		return fmt.Errorf("failed to generate pipeline configuration: %w", err)
	}
	fmt.Printf("DEBUG: Generated pipeline configuration at: %s\n", generatedPipelinePath)

	// Execute the pipeline using rpk
	fmt.Printf("DEBUG: Running pipeline from config: %s\n", generatedPipelinePath)
	err = executor.Execute("rpk", "connect", "run", generatedPipelinePath)
	if err != nil {
		return fmt.Errorf("failed to execute pipeline: %w", err)
	}

	fmt.Println("Pipeline executed successfully.")
	return nil
}

// GeneratePipelineFile dynamically generates a pipeline configuration file.
func GeneratePipelineFile(configPath string, outputPath string) error {
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	templateFilePath := cfg.App.PipelineTemplatePath + "pipeline.yaml"
	if templateFilePath == "" {
		return fmt.Errorf("pipeline template file path is not defined in the configuration")
	}

	fmt.Printf("DEBUG: Template file path: %s\n", templateFilePath)
	template, err := os.ReadFile(templateFilePath)
	if err != nil {
		return fmt.Errorf("failed to read template pipeline file at %s: %w", templateFilePath, err)
	}

	pipeline := string(template)
	placeholders := map[string]string{
		"${KAFKA_BROKERS}":        strings.Join(cfg.App.Kafka.Brokers, ","),
		"${KAFKA_TOPICS}":         strings.Join(cfg.App.Kafka.Topics, ","),
		"${KAFKA_CONSUMER_GROUP}": cfg.App.Kafka.ConsumerGroup,
		"${POSTGRES_URL}":         cfg.App.Postgres.URL,
		"${POSTGRES_TABLE}":       cfg.App.Postgres.Table,
		"${LOG_LEVEL}":            cfg.App.LoggerConfig.Level,
	}

	for placeholder, value := range placeholders {
		if value == "" {
			return fmt.Errorf("placeholder %s is not set in the configuration", placeholder)
		}
		pipeline = strings.ReplaceAll(pipeline, placeholder, value)
	}

	err = os.WriteFile(outputPath, []byte(pipeline), 0644)
	if err != nil {
		return fmt.Errorf("failed to write generated pipeline to %s: %w", outputPath, err)
	}

	return nil
}
