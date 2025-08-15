package swagger

import (
	"fmt"
	"os"
	"os/exec"
)

// GenerateDocs Generate swagger documentation
func GenerateDocs() error {
	// Check if swag tool is installed
	if !isSwagInstalled() {
		fmt.Println("Installing swag tool...")
		if err := installSwag(); err != nil {
			return fmt.Errorf("failed to install swag tool: %v", err)
		}
	}

	// Create docs directory
	docsDir := "./docs"
	if err := os.MkdirAll(docsDir, 0755); err != nil {
		return fmt.Errorf("failed to create docs directory: %v", err)
	}

	// Generate swagger documentation
	fmt.Println("Generating swagger documentation...")
	cmd := exec.Command("swag", "init",
		"-g", "cmd/main.go",
		"-o", docsDir,
		"--parseDependency",
		"--parseInternal",
		"--parseDepth", "2",
		"--generatedTime")

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to generate swagger documentation: %v", err)
	}

	fmt.Println("✅ Swagger documentation generated successfully!")
	fmt.Printf("📁 Documentation location: %s\n", docsDir)
	fmt.Println("🌐 Access URL: http://0.0.0.0:7568/group-chat/docs/index.html")

	return nil
}

// isSwagInstalled Check if swag tool is installed
func isSwagInstalled() bool {
	_, err := exec.LookPath("swag")
	return err == nil
}

// installSwag Install swag tool
func installSwag() error {
	cmd := exec.Command("go", "install", "github.com/swaggo/swag/cmd/swag@latest")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// CleanDocs Clean generated documentation
func CleanDocs() error {
	docsDir := "./docs"
	if err := os.RemoveAll(docsDir); err != nil {
		return fmt.Errorf("failed to clean docs directory: %v", err)
	}
	fmt.Println("✅ Documentation cleanup completed")
	return nil
}

// RegenerateDocs Regenerate documentation
func RegenerateDocs() error {
	fmt.Println("🔄 Regenerating swagger documentation...")

	// Clean old documentation
	if err := CleanDocs(); err != nil {
		return err
	}

	// Generate new documentation
	return GenerateDocs()
}
