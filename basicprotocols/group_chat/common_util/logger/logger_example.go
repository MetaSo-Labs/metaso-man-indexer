package logger

import (
	"fmt"
	"time"
)

// LoggerExample 日志使用示例
func LoggerExample() {
	// 初始化日志（通常在main函数或应用启动时调用）
	err := InitDefaultLogger("./logs", INFO)
	if err != nil {
		fmt.Printf("初始化日志失败: %v\n", err)
		return
	}

	// 设置日志配置
	logger := GetDefaultLogger()
	logger.SetMaxSize(5 * 1024 * 1024) // 5MB
	logger.SetMaxBackups(3)            // 保留3个备份文件
	logger.SetMaxAge(7)                // 保留7天

	// 使用全局便捷方法记录日志
	Debug("这是一条调试信息: %s", "debug message")
	Info("应用启动成功，版本: %s", "v1.0.0")
	Warn("配置文件中的某些设置使用了默认值")
	Error("数据库连接失败: %v", fmt.Errorf("connection timeout"))

	// 模拟一些业务日志
	Info("用户登录: %s", "user123")
	Info("创建群组: %s", "测试群组")
	Warn("群组成员数量接近上限: %d/100", 95)
	Error("发送消息失败: %s", "网络超时")

	// 记录结构化数据
	Info("群组统计 - 总群组数: %d, 活跃用户: %d, 今日消息: %d", 150, 1200, 5000)

	// 模拟错误处理
	if err := simulateError(); err != nil {
		Error("处理请求时发生错误: %v", err)
	}

	// 记录性能信息
	start := time.Now()
	time.Sleep(100 * time.Millisecond) // 模拟处理时间
	Info("请求处理完成，耗时: %v", time.Since(start))

	// 注意：Fatal会终止程序，这里不演示
	// Fatal("致命错误，程序退出")

	fmt.Println("日志示例完成，请查看 ./logs/group_chat.log 文件")
}

// simulateError 模拟一个错误
func simulateError() error {
	return fmt.Errorf("模拟的业务错误")
}

// BusinessLogExample 业务日志示例
func BusinessLogExample() {
	// 初始化日志
	InitDefaultLogger("./logs", DEBUG)

	// 群聊相关业务日志
	Info("=== 群聊业务日志示例 ===")

	// 用户操作日志
	Info("用户操作 - 用户ID: %s, 操作: %s, 群组ID: %s", "user123", "加入群组", "group456")
	Info("用户操作 - 用户ID: %s, 操作: %s, 群组ID: %s", "user123", "发送消息", "group456")

	// 系统状态日志
	Info("系统状态 - 在线用户数: %d, 活跃群组数: %d", 1500, 200)
	Warn("系统警告 - 内存使用率: %.2f%%, CPU使用率: %.2f%%", 85.5, 70.2)

	// 错误处理日志
	Error("业务错误 - 群组不存在: %s", "group999")
	Error("网络错误 - 连接超时: %s", "redis://localhost:6379")

	// 性能监控日志
	Info("性能监控 - API响应时间: %v, 数据库查询时间: %v", 150*time.Millisecond, 50*time.Millisecond)

	// 安全相关日志
	Warn("安全警告 - 异常登录尝试: IP=%s, 用户=%s", "192.168.1.100", "user123")
	Info("安全事件 - 用户登录成功: IP=%s, 用户=%s", "192.168.1.100", "user123")
}
