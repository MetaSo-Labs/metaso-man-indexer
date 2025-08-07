#!/bin/bash

# Group Chat 服务启动脚本

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 默认配置
HOST=${HOST:-"0.0.0.0"}
PORT=${PORT:-"8080"}
MODE=${MODE:-"release"}

# 打印带颜色的消息
print_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

print_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# 检查依赖
check_dependencies() {
    print_info "Checking dependencies..."
    
    if ! command -v go &> /dev/null; then
        print_error "Go is not installed"
        exit 1
    fi
    
    if ! command -v docker &> /dev/null; then
        print_warn "Docker is not installed, Docker features will be disabled"
    fi
    
    print_info "Dependencies check passed"
}

# 构建服务
build_service() {
    print_info "Building Group Chat service..."
    
    cd ../../
    
    # 下载依赖
    go mod download
    
    # 构建服务
    CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o build/group-chat-service ./basicprotocols/group_chat/cmd/main.go
    
    if [ $? -eq 0 ]; then
        print_info "Build completed successfully"
    else
        print_error "Build failed"
        exit 1
    fi
}

# 启动服务
start_service() {
    print_info "Starting Group Chat service..."
    print_info "Host: $HOST"
    print_info "Port: $PORT"
    print_info "Mode: $MODE"
    
    # 设置环境变量
    export GIN_MODE=$MODE
    
    # 启动服务
    ./build/group-chat-service -host $HOST -port $PORT
}

# 使用 Docker 启动
start_docker() {
    print_info "Starting Group Chat service with Docker..."
    
    # 检查 Docker 是否可用
    if ! command -v docker &> /dev/null; then
        print_error "Docker is not available"
        exit 1
    fi
    
    # 构建 Docker 镜像
    print_info "Building Docker image..."
    docker build -t group-chat-service:latest -f basicprotocols/group_chat/Dockerfile .
    
    # 运行容器
    print_info "Starting Docker container..."
    docker run -d \
        --name group-chat-service \
        -p $PORT:8080 \
        -e GIN_MODE=$MODE \
        --restart unless-stopped \
        group-chat-service:latest
    
    print_info "Service started successfully"
    print_info "API documentation: http://localhost:$PORT/swagger/index.html"
    print_info "Health check: http://localhost:$PORT/health"
}

# 使用 Docker Compose 启动
start_compose() {
    print_info "Starting Group Chat service with Docker Compose..."
    
    # 检查 Docker Compose 是否可用
    if ! command -v docker-compose &> /dev/null; then
        print_error "Docker Compose is not available"
        exit 1
    fi
    
    cd basicprotocols/group_chat/
    
    # 启动服务
    docker-compose up -d
    
    print_info "Service started successfully"
    print_info "API documentation: http://localhost:8080/swagger/index.html"
    print_info "Health check: http://localhost:8080/health"
}

# 停止服务
stop_service() {
    print_info "Stopping Group Chat service..."
    
    # 停止本地服务
    pkill -f group-chat-service || true
    
    # 停止 Docker 容器
    docker stop group-chat-service 2>/dev/null || true
    docker rm group-chat-service 2>/dev/null || true
    
    # 停止 Docker Compose 服务
    cd basicprotocols/group_chat/ 2>/dev/null && docker-compose down 2>/dev/null || true
    
    print_info "Service stopped"
}

# 显示状态
show_status() {
    print_info "Checking service status..."
    
    # 检查本地服务
    if pgrep -f group-chat-service > /dev/null; then
        print_info "Local service is running"
    else
        print_warn "Local service is not running"
    fi
    
    # 检查 Docker 容器
    if docker ps | grep group-chat-service > /dev/null; then
        print_info "Docker container is running"
    else
        print_warn "Docker container is not running"
    fi
    
    # 检查健康状态
    if curl -s http://localhost:$PORT/health > /dev/null; then
        print_info "Service is healthy"
    else
        print_warn "Service health check failed"
    fi
}

# 显示帮助
show_help() {
    echo "Group Chat Service Management Script"
    echo ""
    echo "Usage: $0 [COMMAND] [OPTIONS]"
    echo ""
    echo "Commands:"
    echo "  start       Start the service locally"
    echo "  docker      Start the service with Docker"
    echo "  compose     Start the service with Docker Compose"
    echo "  stop        Stop all services"
    echo "  status      Show service status"
    echo "  build       Build the service"
    echo "  help        Show this help message"
    echo ""
    echo "Options:"
    echo "  -h, --host HOST    Server host (default: 0.0.0.0)"
    echo "  -p, --port PORT    Server port (default: 8080)"
    echo "  -m, --mode MODE    Gin mode (default: release)"
    echo ""
    echo "Environment Variables:"
    echo "  HOST               Server host"
    echo "  PORT               Server port"
    echo "  MODE               Gin mode"
    echo ""
    echo "Examples:"
    echo "  $0 start                    # Start locally"
    echo "  $0 docker                   # Start with Docker"
    echo "  $0 compose                  # Start with Docker Compose"
    echo "  $0 start -p 8081           # Start on port 8081"
    echo "  $0 start -m debug          # Start in debug mode"
}

# 解析命令行参数
parse_args() {
    while [[ $# -gt 0 ]]; do
        case $1 in
            -h|--host)
                HOST="$2"
                shift 2
                ;;
            -p|--port)
                PORT="$2"
                shift 2
                ;;
            -m|--mode)
                MODE="$2"
                shift 2
                ;;
            start)
                COMMAND="start"
                shift
                ;;
            docker)
                COMMAND="docker"
                shift
                ;;
            compose)
                COMMAND="compose"
                shift
                ;;
            stop)
                COMMAND="stop"
                shift
                ;;
            status)
                COMMAND="status"
                shift
                ;;
            build)
                COMMAND="build"
                shift
                ;;
            help)
                show_help
                exit 0
                ;;
            *)
                print_error "Unknown option: $1"
                show_help
                exit 1
                ;;
        esac
    done
}

# 主函数
main() {
    # 解析参数
    parse_args "$@"
    
    # 检查依赖
    check_dependencies
    
    # 执行命令
    case $COMMAND in
        start)
            build_service
            start_service
            ;;
        docker)
            start_docker
            ;;
        compose)
            start_compose
            ;;
        stop)
            stop_service
            ;;
        status)
            show_status
            ;;
        build)
            build_service
            ;;
        *)
            print_error "No command specified"
            show_help
            exit 1
            ;;
    esac
}

# 运行主函数
main "$@" 