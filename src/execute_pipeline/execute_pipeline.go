package execute_pipeline

import (
	"fmt"
	"os"
	"os/exec"
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
func ExecutePipeline(configPath string, executor CommandExecutor) error {
	fmt.Printf("DEBUG: Running pipeline from config: %s\n", configPath)

	err := executor.Execute("rpk", "connect", "run", configPath)
	if err != nil {
		return fmt.Errorf("failed to execute pipeline: %w", err)
	}

	fmt.Println("Pipeline executed successfully.")
	return nil
}
