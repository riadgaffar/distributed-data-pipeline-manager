package execute_pipeline

import (
	"fmt"
	"os"
	"os/exec"
)

func ExecutePipeline(configPath string) error {
	// Use rpk or an equivalent tool to execute the pipeline
	cmd := exec.Command("rpk", "connect", "run", configPath)

	// Attach command output for real-time debugging
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	fmt.Printf("DEBUG: Running pipeline from config: %s\n", configPath)
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("failed to execute pipeline: %w", err)
	}

	fmt.Println("Pipeline executed successfully.")
	return nil
}
