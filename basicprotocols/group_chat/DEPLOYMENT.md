# Group Chat 服务部署指南

## 概述

本文档介绍如何部署 Group Chat 服务，包括本地开发、Docker 部署和 Swagger 文档配置。

## 目录结构

```
group_chat/
├── cmd/
│   └── main.go              # 主程序入口
├── server.go                 # 服务器实现
├── Dockerfile               # Docker 构建文件
├── docker-compose.yml       # Docker Compose 配置
├── Makefile                 # 构建脚本
└── DEPLOYMENT.md           # 部署文档
```

## 快速开始

### 1. 本地开发

#### 安装依赖
```bash
# 安装 Go 依赖
go mod download

# 安装 Swagger 工具
go install github.com/swaggo/swag/cmd/swag@latest
```

#### 运行服务
```bash
# 使用 Makefile
make run

# 或者直接运行
go run ./cmd/main.go

# 指定端口运行
go run ./cmd/main.go -port 8080
```

### 2. Docker 部署

#### 构建镜像
```bash
# 使用 Makefile
make docker-build

# 或者直接构建
docker build -t group-chat-service:latest -f Dockerfile ../../
```

#### 运行容器
```bash
# 使用 Makefile
make docker-run

# 或者直接运行
docker run -d --name group-chat-service -p 8080:8080 group-chat-service:latest
```

#### 使用 Docker Compose
```bash
# 启动服务
make compose-up

# 查看日志
make compose-logs

# 停止服务
make compose-down
```

## API 端点

### 基础端点
- `GET /` - 服务信息
- `GET /health` - 健康检查
- `GET /status` - 服务状态
- `GET /swagger/index.html` - Swagger 文档

### 群聊 API
- `GET /group-chat/group-list` - 获取群组列表
- `GET /group-chat/user/latest-group-list` - 获取最新聊天群组
- `GET /group-chat/group-info` - 获取群组信息
- `GET /group-chat/group-chat-list` - 获取聊天记录
- `GET /group-chat/group-member-list` - 获取成员列表

## 配置选项

### 环境变量
- `GIN_MODE` - Gin 运行模式 (debug/release)
- `HOST` - 服务器监听地址 (默认: 0.0.0.0)
- `PORT` - 服务器监听端口 (默认: 8080)

### 命令行参数
```bash
./group-chat-service -host 0.0.0.0 -port 8080
```

## 生产环境部署

### 1. 使用 Docker

#### 构建生产镜像
```bash
docker build -t group-chat-service:prod -f Dockerfile ../../
```

#### 运行生产容器
```bash
docker run -d \
  --name group-chat-service \
  -p 8080:8080 \
  -v /path/to/data:/app/data \
  --restart unless-stopped \
  group-chat-service:prod
```

### 2. 使用 Docker Compose

#### 生产环境配置
```yaml
version: '3.8'
services:
  group-chat-service:
    image: group-chat-service:prod
    ports:
      - "8080:8080"
    volumes:
      - /path/to/data:/app/data
    restart: unless-stopped
    environment:
      - GIN_MODE=release
```

#### 启动服务
```bash
docker-compose -f docker-compose.prod.yml up -d
```

### 3. 使用 Kubernetes

#### 创建 Deployment
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: group-chat-service
spec:
  replicas: 3
  selector:
    matchLabels:
      app: group-chat-service
  template:
    metadata:
      labels:
        app: group-chat-service
    spec:
      containers:
      - name: group-chat-service
        image: group-chat-service:latest
        ports:
        - containerPort: 8080
        env:
        - name: GIN_MODE
          value: "release"
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 5
```

#### 创建 Service
```yaml
apiVersion: v1
kind: Service
metadata:
  name: group-chat-service
spec:
  selector:
    app: group-chat-service
  ports:
  - port: 80
    targetPort: 8080
  type: LoadBalancer
```

## 监控和日志

### 健康检查
```bash
# 检查服务健康状态
curl http://localhost:8080/health

# 检查服务状态
curl http://localhost:8080/status
```

### 日志查看
```bash
# Docker 容器日志
docker logs group-chat-service

# Docker Compose 日志
docker-compose logs -f group-chat-service
```

### 性能监控
```bash
# 查看容器资源使用
docker stats group-chat-service
```

## 故障排除

### 常见问题

1. **端口被占用**
   ```bash
   # 检查端口使用
   lsof -i :8080
   
   # 使用不同端口
   ./group-chat-service -port 8081
   ```

2. **数据库连接失败**
   - 检查数据库服务是否运行
   - 检查数据库连接配置
   - 查看服务日志

3. **内存不足**
   ```bash
   # 增加容器内存限制
   docker run -d --memory=512m group-chat-service:latest
   ```

### 调试模式
```bash
# 设置调试模式
export GIN_MODE=debug
./group-chat-service
```

## 安全考虑

### 1. 网络安全
- 使用 HTTPS 在生产环境中
- 配置防火墙规则
- 限制网络访问

### 2. 容器安全
- 使用非 root 用户运行
- 定期更新基础镜像
- 扫描安全漏洞

### 3. 数据安全
- 加密敏感数据
- 定期备份数据
- 实施访问控制

## 扩展和优化

### 1. 负载均衡
```bash
# 使用 Nginx 负载均衡
upstream group_chat {
    server 127.0.0.1:8080;
    server 127.0.0.1:8081;
    server 127.0.0.1:8082;
}
```

### 2. 缓存策略
- 使用 Redis 缓存热点数据
- 实施 CDN 加速静态资源
- 配置数据库连接池

### 3. 监控告警
- 集成 Prometheus 监控
- 配置 Grafana 仪表板
- 设置告警规则

## 更新和升级

### 1. 滚动更新
```bash
# 构建新镜像
docker build -t group-chat-service:v2.0 .

# 更新服务
docker-compose up -d --no-deps group-chat-service
```

### 2. 回滚策略
```bash
# 回滚到上一个版本
docker-compose up -d --no-deps group-chat-service:previous
```

## 支持

如有问题，请查看：
- 服务日志
- Swagger 文档
- 健康检查端点
- 项目文档 