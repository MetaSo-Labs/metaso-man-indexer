package swagger

import (
	"fmt"
	"os"
	"os/exec"
)

// GenerateDocs 生成swagger文档
func GenerateDocs() error {
	// 检查swag工具是否安装
	if !isSwagInstalled() {
		fmt.Println("正在安装swag工具...")
		if err := installSwag(); err != nil {
			return fmt.Errorf("安装swag工具失败: %v", err)
		}
	}

	// 创建docs目录
	docsDir := "./docs"
	if err := os.MkdirAll(docsDir, 0755); err != nil {
		return fmt.Errorf("创建docs目录失败: %v", err)
	}

	// 生成swagger文档
	fmt.Println("正在生成swagger文档...")
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
		return fmt.Errorf("生成swagger文档失败: %v", err)
	}

	fmt.Println("✅ swagger文档生成成功！")
	fmt.Printf("📁 文档位置: %s\n", docsDir)
	fmt.Println("🌐 访问地址: http://0.0.0.0:7568/group-chat/docs/index.html")

	return nil
}

// isSwagInstalled 检查swag工具是否已安装
func isSwagInstalled() bool {
	_, err := exec.LookPath("swag")
	return err == nil
}

// installSwag 安装swag工具
func installSwag() error {
	cmd := exec.Command("go", "install", "github.com/swaggo/swag/cmd/swag@latest")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// CleanDocs 清理生成的文档
func CleanDocs() error {
	docsDir := "./docs"
	if err := os.RemoveAll(docsDir); err != nil {
		return fmt.Errorf("清理docs目录失败: %v", err)
	}
	fmt.Println("✅ 文档清理完成")
	return nil
}

// RegenerateDocs 重新生成文档
func RegenerateDocs() error {
	fmt.Println("🔄 重新生成swagger文档...")

	// 清理旧文档
	if err := CleanDocs(); err != nil {
		return err
	}

	// 生成新文档
	return GenerateDocs()
}
