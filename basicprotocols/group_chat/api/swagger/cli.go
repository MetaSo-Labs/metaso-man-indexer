package swagger

import (
	"fmt"
	"os"
)

// CLI Command line interface
func CLI() {
	if len(os.Args) < 2 {
		printUsage()
		return
	}

	command := os.Args[1]

	switch command {
	case "generate", "gen":
		if err := GenerateDocs(); err != nil {
			fmt.Printf("❌ Generation failed: %v\n", err)
			os.Exit(1)
		}
	case "clean":
		if err := CleanDocs(); err != nil {
			fmt.Printf("❌ Cleanup failed: %v\n", err)
			os.Exit(1)
		}
	case "regenerate", "regen":
		if err := RegenerateDocs(); err != nil {
			fmt.Printf("❌ Regeneration failed: %v\n", err)
			os.Exit(1)
		}
	case "install":
		if err := installSwag(); err != nil {
			fmt.Printf("❌ Installation failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("✅ swag tool installed successfully")
	case "help", "--help", "-h":
		printUsage()
	default:
		fmt.Printf("❌ Unknown command: %s\n", command)
		printUsage()
		os.Exit(1)
	}
}

// printUsage Print usage instructions
func printUsage() {
	fmt.Println("Group Chat Module Swagger Documentation Management Tool")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  generate, gen     - Generate swagger documentation")
	fmt.Println("  clean             - Clean generated documentation")
	fmt.Println("  regenerate, regen - Regenerate documentation")
	fmt.Println("  install           - Install swag tool")
	fmt.Println("  help              - Show this help information")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  go run api/swagger/cmd/main.go generate")
	fmt.Println("  go run api/swagger/cmd/main.go clean")
	fmt.Println("  go run api/swagger/cmd/main.go regenerate")
}
