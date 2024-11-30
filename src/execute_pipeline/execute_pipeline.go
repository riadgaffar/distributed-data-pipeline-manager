package execute_pipeline

import (
	"distributed-data-pipeline-manager/src/config"
	"fmt"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

// Executor interface to top Stop cmd
type Executor interface {
	ExecuteCommand(cmd *exec.Cmd) error
	Stop(cmd *exec.Cmd) error
}

// CommandExecutor abstracts command execution for testability.
type CommandExecutor interface {
	ExecuteCommand(name string, args ...string) error
	StopPipeline() error
}

// RealCommandExecutor implements CommandExecutor and uses exec.Command.
type RealCommandExecutor struct {
	cmd *exec.Cmd
}

func (r *RealCommandExecutor) ExecuteCommand(name string, args ...string) error {
	r.cmd = exec.Command(name, args...)
	r.cmd.Stdout = nil
	r.cmd.Stderr = nil

	if err := r.cmd.Start(); err != nil {
		return err
	}

	log.Printf("DEBUG: Pipeline process started with PID %d", r.cmd.Process.Pid)
	return nil
}

// ExecutePipeline runs the pipeline using the provided CommandExecutor.
// ExecutePipeline generates a dynamic pipeline configuration and runs it using rpk.
func ExecutePipeline(configPath string, executor CommandExecutor) error {
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}
	log.Printf("INFO: Loaded configuration: %+v", cfg)

	generatedPipelinePath := cfg.App.GeneratePipelineConfig
	err = GeneratePipelineFile(configPath, generatedPipelinePath)
	if err != nil {
		return fmt.Errorf("failed to generate pipeline configuration: %w", err)
	}
	log.Printf("INFO: Generated pipeline configuration at: %s", generatedPipelinePath)

	if err := validatePipelineFile(generatedPipelinePath); err != nil {
		return fmt.Errorf("failed to validate pipeline configuration: %w", err)
	}

	log.Printf("INFO: Executing pipeline using configuration: %s", generatedPipelinePath)
	err = executor.ExecuteCommand("rpk", "connect", "run", generatedPipelinePath)
	if err != nil {
		return fmt.Errorf("failed to execute pipeline: %w", err)
	}

	log.Println("INFO: Pipeline executed successfully.")
	return nil
}

// GeneratePipelineFile dynamically generates a pipeline configuration file.
func GeneratePipelineFile(configPath string, outputPath string) error {
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		return fmt.Errorf("failed to load generated configuration: %w", err)
	}

	templateFilePath := cfg.App.PipelineTemplate
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

// Stop function stops the cmd running the pipeline
func (e *RealCommandExecutor) Stop(cmd *exec.Cmd) error {
	if cmd == nil || cmd.Process == nil {
		return nil // No process to stop
	}
	// Send SIGINT for graceful shutdown
	err := cmd.Process.Signal(os.Interrupt)
	if err != nil {
		return err
	}

	// Wait for the process to terminate
	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	select {
	case <-time.After(10 * time.Second):
		// Force kill if it doesn't terminate
		return cmd.Process.Kill()
	case err := <-done:
		return err
	}
}

// StopPipeline gracefully stops the pipeline process
// StopPipeline gracefully stops the pipeline process
func (e *RealCommandExecutor) StopPipeline() error {
	// Ensure the command has been initialized
	if e.cmd == nil {
		log.Println("ERROR: No pipeline process to stop.")
		return fmt.Errorf("no pipeline process to stop")
	}

	// Ensure the command's process is valid
	if e.cmd.Process == nil {
		log.Println("ERROR: Pipeline process is invalid or already stopped.")
		return fmt.Errorf("pipeline process is invalid or already stopped")
	}

	// Attempt to send a SIGINT signal for graceful shutdown
	log.Printf("DEBUG: Sending SIGINT to pipeline process with PID %d", e.cmd.Process.Pid)
	if err := e.cmd.Process.Signal(os.Interrupt); err != nil {
		log.Printf("ERROR: Failed to send SIGINT: %v", err)
		return fmt.Errorf("failed to send SIGINT: %w", err)
	}

	// Wait for the process to terminate or timeout
	done := make(chan error, 1)
	go func() {
		done <- e.cmd.Wait()
	}()

	select {
	case err := <-done:
		if err != nil {
			log.Printf("ERROR: Pipeline process terminated with error: %v", err)
			return fmt.Errorf("pipeline process terminated with error: %w", err)
		}
		log.Println("INFO: Pipeline process stopped gracefully.")
		return nil
	case <-time.After(10 * time.Second):
		// Force kill the process if it doesn't terminate within the timeout
		log.Printf("WARNING: Pipeline process did not terminate in time. Forcing kill on PID %d", e.cmd.Process.Pid)
		if err := e.cmd.Process.Kill(); err != nil {
			log.Printf("ERROR: Failed to kill pipeline process: %v", err)
			return fmt.Errorf("failed to kill pipeline process: %w", err)
		}
		log.Println("INFO: Pipeline process forcefully stopped.")
		return nil
	}
}

func validatePipelineFile(filePath string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read pipeline file at %s: %w", filePath, err)
	}

	// Regex to detect placeholders like ${PLACEHOLDER}
	placeholderRegex := regexp.MustCompile(`\$\{[^}]+\}`)
	allowedPlaceholders := []string{"${! json() }"}

	if placeholderRegex.Match(data) {
		for _, placeholder := range allowedPlaceholders {
			if strings.Contains(string(data), placeholder) {
				// Skip valid placeholders
				continue
			}
			return fmt.Errorf("pipeline file contains unresolved placeholder: %s", filePath)
		}
	}

	log.Printf("INFO: Pipeline file at %s validated successfully.", filePath)
	return nil
}
