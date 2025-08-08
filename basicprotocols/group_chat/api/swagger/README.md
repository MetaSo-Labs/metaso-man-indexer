# 群聊模块 Swagger 文档管理

## 概述

本目录包含群聊模块的 Swagger 文档管理工具，所有 swagger 相关的功能都集中在这里。

## 目录结构

```
api/swagger/
├── swagger.go      # Swagger 路由配置
├── generate.go     # 文档生成工具
├── cli.go          # 命令行接口
├── cmd/
│   └── main.go     # CLI 工具入口
├── generate.sh     # 生成脚本
└── README.md       # 本文件
```

## 使用方法

### 1. 生成 Swagger 文档

#### 方法一：使用脚本（推荐）
```bash
cd api/swagger
./generate.sh
```

#### 方法二：使用 Go 工具
```bash
go run api/swagger/cmd/main.go generate
```

#### 方法三：直接使用 swag 命令
```bash
swag init -g cmd/main.go -o ./docs --parseDependency --parseInternal
```

### 2. 清理文档
```bash
go run api/swagger/cmd/main.go clean
```

### 3. 重新生成文档
```bash
go run api/swagger/cmd/main.go regenerate
```

### 4. 安装 swag 工具
```bash
go run api/swagger/cmd/main.go install
```

## 启动服务器

生成文档后，启动服务器：

```bash
go run cmd/main.go -host 0.0.0.0 -port 7568
```

## 访问 Swagger 文档

服务器启动后，访问以下地址查看 API 文档：

```
http://0.0.0.0:7568/group-chat/docs/index.html
```

## 开发工作流

### 1. 修改 API 代码后

1. 更新 API 代码和注释
2. 重新生成文档：
   ```bash
   cd api/swagger && ./generate.sh
   ```
3. 重启服务器
4. 访问文档查看更新

### 2. 添加新的 API

1. 在 `api/db_controller.go` 中添加新的 API 函数
2. 添加完整的 Swagger 注释
3. 在 `api/db_routes.go` 中注册路由
4. 生成文档：`cd api/swagger && ./generate.sh`
5. 启动服务器测试

## Swagger 注释格式

```go
// @Summary API 标题
// @Description API 详细描述
// @Tags 数据库查询
// @Accept json
// @Produce json
// @Param 参数名 query 参数类型 是否必需 "参数描述"
// @Success 200 {object} map[string]interface{} "成功描述"
// @Failure 400 {object} map[string]interface{} "错误描述"
// @Router /api/path [get]
func YourAPIHandler(c *gin.Context) {
    // 你的代码
}
```

## 注意事项

1. **注释位置**：Swagger 注释必须紧贴在函数上方
2. **注释格式**：必须使用 `//` 开头的注释
3. **参数类型**：确保参数类型与实际代码匹配
4. **路由路径**：确保 `@Router` 中的路径与实际路由一致
5. **标签使用**：使用合适的标签来组织 API

## 故障排除

### 1. swag 命令未找到
```bash
go install github.com/swaggo/swag/cmd/swag@latest
```

### 2. 生成的文档不完整
检查注释格式是否正确，确保：
- 注释以 `//` 开头
- 每个注释都在函数上方
- 参数类型和描述正确

### 3. Swagger UI 无法访问
确保：
- 服务器正在运行
- 访问正确的 URL：`http://0.0.0.0:7568/group-chat/swagger/index.html`
- 端口没有被其他服务占用

## 文件说明

- `swagger.go`: 配置 Swagger 路由和 URL 生成
- `generate.go`: 提供文档生成、清理、重新生成等功能
- `cli.go`: 命令行接口，支持各种操作
- `generate.sh`: 简单的生成脚本，方便快速使用

## 更多资源

- [Swaggo 官方文档](https://github.com/swaggo/swag)
- [Swagger 规范](https://swagger.io/specification/)
- [Gin 框架文档](https://gin-gonic.com/docs/) 