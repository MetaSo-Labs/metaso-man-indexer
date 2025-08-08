package swagger

import (
	"fmt"
	"os"
)

// CLI 命令行接口
func CLI() {
	if len(os.Args) < 2 {
		printUsage()
		return
	}

	command := os.Args[1]

	switch command {
	case "generate", "gen":
		if err := GenerateDocs(); err != nil {
			fmt.Printf("❌ 生成失败: %v\n", err)
			os.Exit(1)
		}
	case "clean":
		if err := CleanDocs(); err != nil {
			fmt.Printf("❌ 清理失败: %v\n", err)
			os.Exit(1)
		}
	case "regenerate", "regen":
		if err := RegenerateDocs(); err != nil {
			fmt.Printf("❌ 重新生成失败: %v\n", err)
			os.Exit(1)
		}
	case "install":
		if err := installSwag(); err != nil {
			fmt.Printf("❌ 安装失败: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("✅ swag工具安装成功")
	case "help", "--help", "-h":
		printUsage()
	default:
		fmt.Printf("❌ 未知命令: %s\n", command)
		printUsage()
		os.Exit(1)
	}
}

// printUsage 打印使用说明
func printUsage() {
	fmt.Println("群聊模块 Swagger 文档管理工具")
	fmt.Println()
	fmt.Println("用法:")
	fmt.Println("  generate, gen     - 生成swagger文档")
	fmt.Println("  clean             - 清理生成的文档")
	fmt.Println("  regenerate, regen - 重新生成文档")
	fmt.Println("  install           - 安装swag工具")
	fmt.Println("  help              - 显示此帮助信息")
	fmt.Println()
	fmt.Println("示例:")
	fmt.Println("  go run api/swagger/cmd/main.go generate")
	fmt.Println("  go run api/swagger/cmd/main.go clean")
	fmt.Println("  go run api/swagger/cmd/main.go regenerate")
}
