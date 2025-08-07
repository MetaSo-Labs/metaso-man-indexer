# 变量定义
BINARY_NAME=group-chat-service
BUILD_DIR=build
DOCKER_IMAGE=group-chat-service
DOCKER_TAG=latest

# 默认目标
.PHONY: all
all: clean build

# 清理构建目录
.PHONY: clean
clean:
	@echo "Cleaning build directory..."
	@rm -rf $(BUILD_DIR)
	@mkdir -p $(BUILD_DIR)

# 构建二进制文件
.PHONY: build
build: clean
	@echo "Building $(BINARY_NAME)..."
	@cd ../../ && CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o $(BUILD_DIR)/$(BINARY_NAME) ./basicprotocols/group_chat/cmd/main.go

# 运行本地开发服务器
.PHONY: run
run:
	@echo "Running local development server..."
	@cd ../../ && go run ./basicprotocols/group_chat/cmd/main.go

# 运行本地开发服务器（指定端口）
.PHONY: run-dev
run-dev:
	@echo "Running local development server on port 8080..."
	@cd ../../ && go run ./basicprotocols/group_chat/cmd/main.go -port 8080

# 构建 Docker 镜像
.PHONY: docker-build
docker-build:
	@echo "Building Docker image..."
	@docker build -t $(DOCKER_IMAGE):$(DOCKER_TAG) -f Dockerfile ../../

# 运行 Docker 容器
.PHONY: docker-run
docker-run: docker-build
	@echo "Running Docker container..."
	@docker run -d --name $(BINARY_NAME) -p 8080:8080 $(DOCKER_IMAGE):$(DOCKER_TAG)

# 停止 Docker 容器
.PHONY: docker-stop
docker-stop:
	@echo "Stopping Docker container..."
	@docker stop $(BINARY_NAME) || true
	@docker rm $(BINARY_NAME) || true

# 使用 docker-compose 启动服务
.PHONY: compose-up
compose-up:
	@echo "Starting services with docker-compose..."
	@docker-compose up -d

# 停止 docker-compose 服务
.PHONY: compose-down
compose-down:
	@echo "Stopping services with docker-compose..."
	@docker-compose down

# 查看 docker-compose 日志
.PHONY: compose-logs
compose-logs:
	@docker-compose logs -f

# 测试 API
.PHONY: test-api
test-api:
	@echo "Testing API endpoints..."
	@curl -s http://localhost:8080/health | jq .
	@curl -s http://localhost:8080/status | jq .
	@curl -s http://localhost:8080/ | jq .

# 生成 Swagger 文档
.PHONY: swagger
swagger:
	@echo "Generating Swagger documentation..."
	@cd ../../ && swag init -g basicprotocols/group_chat/cmd/main.go -o basicprotocols/group_chat/docs

# 安装依赖
.PHONY: deps
deps:
	@echo "Installing dependencies..."
	@cd ../../ && go mod download
	@go install github.com/swaggo/swag/cmd/swag@latest

# 格式化代码
.PHONY: fmt
fmt:
	@echo "Formatting code..."
	@cd ../../ && go fmt ./basicprotocols/group_chat/...

# 代码检查
.PHONY: lint
lint:
	@echo "Running linter..."
	@cd ../../ && golangci-lint run ./basicprotocols/group_chat/...

# 运行测试
.PHONY: test
test:
	@echo "Running tests..."
	@cd ../../ && go test ./basicprotocols/group_chat/...

# 帮助信息
.PHONY: help
help:
	@echo "Available targets:"
	@echo "  build        - Build binary file"
	@echo "  run          - Run local development server"
	@echo "  run-dev      - Run local development server on port 8080"
	@echo "  docker-build - Build Docker image"
	@echo "  docker-run   - Run Docker container"
	@echo "  docker-stop  - Stop Docker container"
	@echo "  compose-up   - Start services with docker-compose"
	@echo "  compose-down - Stop services with docker-compose"
	@echo "  compose-logs - View docker-compose logs"
	@echo "  test-api     - Test API endpoints"
	@echo "  swagger      - Generate Swagger documentation"
	@echo "  deps         - Install dependencies"
	@echo "  fmt          - Format code"
	@echo "  lint         - Run linter"
	@echo "  test         - Run tests"
	@echo "  help         - Show this help message" 