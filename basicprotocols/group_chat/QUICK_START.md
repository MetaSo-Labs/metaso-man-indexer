# Group Chat 服务快速开始

## 🚀 快速部署

### 方法一：使用启动脚本（推荐）

```bash
# 进入 group_chat 目录
cd basicprotocols/group_chat/

# 本地启动
./start.sh start

# Docker 启动
./start.sh docker

# Docker Compose 启动
./start.sh compose

# 查看状态
./start.sh status

# 停止服务
./start.sh stop
```

### 方法二：使用 Makefile

```bash
# 本地运行
make run

# Docker 构建和运行
make docker-build
make docker-run

# Docker Compose
make compose-up
make compose-logs
make compose-down
```

### 方法三：直接运行

```bash
# 构建
go build -o group-chat-service ./cmd/main.go

# 运行
./group-chat-service -port 8080
```

## 📋 服务端点

启动后，您可以访问以下端点：

- **服务信息**: http://localhost:8080/
- **健康检查**: http://localhost:8080/health
- **服务状态**: http://localhost:8080/status
- **API 文档**: http://localhost:8080/swagger/index.html

## 🔧 API 测试

### 获取群组列表
```bash
curl "http://localhost:8080/group-chat/group-list?cursor=1&size=20"
```

### 获取用户最新聊天群组
```bash
curl "http://localhost:8080/group-chat/user/latest-group-list?metaId=user123&cursor=1&size=20"
```

### 获取群组信息
```bash
curl "http://localhost:8080/group-chat/group-info?groupId=group123"
```

### 获取群组聊天记录
```bash
curl "http://localhost:8080/group-chat/group-chat-list?groupId=group123&size=20"
```

### 获取群组成员
```bash
curl "http://localhost:8080/group-chat/group-member-list?groupId=group123&cursor=1&size=20"
```

## 🐳 Docker 部署

### 构建镜像
```bash
docker build -t group-chat-service:latest -f Dockerfile ../../
```

### 运行容器
```bash
docker run -d \
  --name group-chat-service \
  -p 8080:8080 \
  --restart unless-stopped \
  group-chat-service:latest
```

### 使用 Docker Compose
```bash
docker-compose up -d
```

## 🔍 监控和调试

### 查看日志
```bash
# 本地服务
tail -f logs/group-chat.log

# Docker 容器
docker logs -f group-chat-service

# Docker Compose
docker-compose logs -f
```

### 健康检查
```bash
curl http://localhost:8080/health
```

### 性能监控
```bash
# 查看容器资源使用
docker stats group-chat-service

# 查看进程
ps aux | grep group-chat-service
```

## ⚙️ 配置选项

### 环境变量
```bash
export HOST=0.0.0.0
export PORT=8080
export GIN_MODE=release
```

### 命令行参数
```bash
./group-chat-service -host 0.0.0.0 -port 8080
```

## 🛠️ 开发模式

### 调试模式启动
```bash
export GIN_MODE=debug
./start.sh start -m debug
```

### 热重载开发
```bash
# 安装 air 工具
go install github.com/cosmtrek/air@latest

# 创建 air 配置
cat > .air.toml << EOF
root = "."
test_delay = 0
test_timeout = "30s"
test_recursive = false
test_stop_on_failure = false
test_send_interrupt = false
test_kill_delay = "0.5s"

[build]
  args_bin = []
  bin = "./tmp/main"
  cmd = "go build -o ./tmp/main ./cmd/main.go"
  delay = 1000
  exclude_dir = ["assets", "tmp", "vendor", "testdata"]
  exclude_file = []
  exclude_regex = ["_test.go"]
  exclude_unchanged = false
  follow_symlink = false
  full_bin = ""
  include_dir = []
  include_ext = ["go", "tpl", "tmpl", "html"]
  include_file = []
  kill_delay = "0s"
  log = "build-errors.log"
  poll = false
  poll_interval = 0
  rerun = false
  rerun_delay = 500
  send_interrupt = false
  stop_on_root = false

[color]
  app = ""
  build = "yellow"
  main = "magenta"
  runner = "green"
  watcher = "cyan"

[log]
  main_only = false
  time = false

[misc]
  clean_on_exit = false
EOF

# 启动开发服务器
air
```

## 🔧 故障排除

### 常见问题

1. **端口被占用**
   ```bash
   # 检查端口
   lsof -i :8080
   
   # 使用不同端口
   ./start.sh start -p 8081
   ```

2. **权限问题**
   ```bash
   # 添加执行权限
   chmod +x start.sh
   ```

3. **Docker 问题**
   ```bash
   # 清理 Docker 资源
   docker system prune -f
   
   # 重新构建
   docker build --no-cache -t group-chat-service:latest .
   ```

4. **数据库连接问题**
   - 检查数据库服务是否运行
   - 检查数据库连接配置
   - 查看服务日志

### 调试技巧

1. **启用详细日志**
   ```bash
   export GIN_MODE=debug
   ./group-chat-service
   ```

2. **查看实时日志**
   ```bash
   tail -f logs/group-chat.log
   ```

3. **性能分析**
   ```bash
   # 使用 pprof
   go tool pprof http://localhost:8080/debug/pprof/profile
   ```

## 📚 更多信息

- [完整部署指南](DEPLOYMENT.md)
- [API 文档](http://localhost:8080/swagger/index.html)
- [项目文档](../README.md)

## 🆘 获取帮助

如果遇到问题，请：

1. 查看服务日志
2. 检查健康状态端点
3. 参考 Swagger 文档
4. 查看项目文档 