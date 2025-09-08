package logger

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// LogLevel 日志级别
type LogLevel int

const (
	DEBUG LogLevel = iota
	INFO
	WARN
	ERROR
	FATAL
)

// Logger 日志结构体
type Logger struct {
	level      LogLevel
	logger     *log.Logger
	file       *os.File
	logDir     string
	maxSize    int64 // 最大文件大小（字节）
	maxBackups int   // 最大备份文件数量
	maxAge     int   // 最大保存天数
}

// 全局日志实例
var defaultLogger *Logger

// 日志级别字符串映射
var levelStrings = map[LogLevel]string{
	DEBUG: "DEBUG",
	INFO:  "INFO",
	WARN:  "WARN",
	ERROR: "ERROR",
	FATAL: "FATAL",
}

// 日志级别颜色映射
var levelColors = map[LogLevel]string{
	DEBUG: "\033[36m", // 青色
	INFO:  "\033[32m", // 绿色
	WARN:  "\033[33m", // 黄色
	ERROR: "\033[31m", // 红色
	FATAL: "\033[35m", // 紫色
}

const resetColor = "\033[0m"

// NewLogger 创建新的日志实例
func NewLogger(logDir string, level LogLevel) (*Logger, error) {
	// 确保日志目录存在
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, fmt.Errorf("创建日志目录失败: %v", err)
	}

	// 创建日志文件
	logFile := filepath.Join(logDir, "group_chat.log")
	file, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return nil, fmt.Errorf("打开日志文件失败: %v", err)
	}

	// 创建多写入器，同时输出到文件和控制台
	multiWriter := io.MultiWriter(file, os.Stdout)

	logger := &Logger{
		level:      level,
		logger:     log.New(multiWriter, "", 0),
		file:       file,
		logDir:     logDir,
		maxSize:    10 * 1024 * 1024, // 10MB
		maxBackups: 5,
		maxAge:     30, // 30天
	}

	return logger, nil
}

// InitDefaultLogger 初始化默认日志实例
func InitDefaultLogger(logDir string, level LogLevel) error {
	logger, err := NewLogger(logDir, level)
	if err != nil {
		return err
	}
	defaultLogger = logger
	return nil
}

// GetDefaultLogger 获取默认日志实例
func GetDefaultLogger() *Logger {
	if defaultLogger == nil {
		// 如果没有初始化，使用默认配置
		logger, _ := NewLogger("./logs", INFO)
		defaultLogger = logger
	}
	return defaultLogger
}

// log 内部日志方法
func (l *Logger) log(level LogLevel, format string, args ...interface{}) {
	if level < l.level {
		return
	}

	// 获取调用者信息
	_, file, line, ok := runtime.Caller(2)
	if !ok {
		file = "unknown"
		line = 0
	} else {
		file = filepath.Base(file)
	}

	// 格式化时间
	now := time.Now()
	timestamp := now.Format("2006-01-02 15:04:05.000")

	// 判断format尾部是否有换行符
	if strings.HasSuffix(format, "\n") {
		format = strings.TrimSuffix(format, "\n")
	}

	// 构建日志消息
	message := fmt.Sprintf(format, args...)
	levelStr := levelStrings[level]
	color := levelColors[level]

	// 构建完整的日志行
	logLine := fmt.Sprintf("[%s] %s[%s]%s %s:%d - %s",
		timestamp,
		color,
		levelStr,
		resetColor,
		file,
		line,
		message,
	)

	// 写入日志
	l.logger.Println(logLine)

	// 检查是否需要轮转日志文件
	l.rotateIfNeeded()
}

// Debug 调试日志
func (l *Logger) Debug(format string, args ...interface{}) {
	l.log(DEBUG, format, args...)
}

// Info 信息日志
func (l *Logger) Info(format string, args ...interface{}) {
	l.log(INFO, format, args...)
}

// Warn 警告日志
func (l *Logger) Warn(format string, args ...interface{}) {
	l.log(WARN, format, args...)
}

// Error 错误日志
func (l *Logger) Error(format string, args ...interface{}) {
	l.log(ERROR, format, args...)
}

// Fatal 致命错误日志
func (l *Logger) Fatal(format string, args ...interface{}) {
	l.log(FATAL, format, args...)
	os.Exit(1)
}

// SetLevel 设置日志级别
func (l *Logger) SetLevel(level LogLevel) {
	l.level = level
}

// SetMaxSize 设置最大文件大小
func (l *Logger) SetMaxSize(size int64) {
	l.maxSize = size
}

// SetMaxBackups 设置最大备份文件数量
func (l *Logger) SetMaxBackups(count int) {
	l.maxBackups = count
}

// SetMaxAge 设置最大保存天数
func (l *Logger) SetMaxAge(days int) {
	l.maxAge = days
}

// rotateIfNeeded 检查是否需要轮转日志文件
func (l *Logger) rotateIfNeeded() {
	// 获取当前文件信息
	info, err := l.file.Stat()
	if err != nil {
		return
	}

	// 如果文件大小超过限制，进行轮转
	if info.Size() >= l.maxSize {
		l.rotate()
	}
}

// rotate 轮转日志文件
func (l *Logger) rotate() {
	// 关闭当前文件
	l.file.Close()

	// 生成新的文件名（带时间戳）
	now := time.Now()
	backupName := fmt.Sprintf("group_chat_%s.log", now.Format("2006-01-02_15-04-05"))
	backupPath := filepath.Join(l.logDir, backupName)

	// 重命名当前日志文件
	currentPath := filepath.Join(l.logDir, "group_chat.log")
	os.Rename(currentPath, backupPath)

	// 创建新的日志文件
	file, err := os.OpenFile(currentPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		// 如果创建失败，尝试使用标准输出
		l.logger = log.New(os.Stdout, "", 0)
		return
	}

	// 更新logger
	l.file = file
	multiWriter := io.MultiWriter(file, os.Stdout)
	l.logger = log.New(multiWriter, "", 0)

	// 清理旧的备份文件
	l.cleanupOldBackups()
}

// cleanupOldBackups 清理旧的备份文件
func (l *Logger) cleanupOldBackups() {
	// 读取日志目录中的所有文件
	files, err := os.ReadDir(l.logDir)
	if err != nil {
		return
	}

	var backupFiles []os.DirEntry
	cutoffTime := time.Now().AddDate(0, 0, -l.maxAge)

	// 找到所有备份文件
	for _, file := range files {
		if strings.HasPrefix(file.Name(), "group_chat_") && strings.HasSuffix(file.Name(), ".log") {
			backupFiles = append(backupFiles, file)
		}
	}

	// 如果备份文件数量超过限制，删除最旧的
	if len(backupFiles) > l.maxBackups {
		// 按修改时间排序
		for i := 0; i < len(backupFiles)-l.maxBackups; i++ {
			filePath := filepath.Join(l.logDir, backupFiles[i].Name())
			os.Remove(filePath)
		}
	}

	// 删除超过最大保存天数的文件
	for _, file := range backupFiles {
		info, err := file.Info()
		if err != nil {
			continue
		}
		if info.ModTime().Before(cutoffTime) {
			filePath := filepath.Join(l.logDir, file.Name())
			os.Remove(filePath)
		}
	}
}

// Close 关闭日志文件
func (l *Logger) Close() error {
	if l.file != nil {
		return l.file.Close()
	}
	return nil
}

// 全局便捷方法
func Debug(format string, args ...interface{}) {
	GetDefaultLogger().Debug(format, args...)
}

func Info(format string, args ...interface{}) {
	GetDefaultLogger().Info(format, args...)
}

func Warn(format string, args ...interface{}) {
	GetDefaultLogger().Warn(format, args...)
}

func Error(format string, args ...interface{}) {
	GetDefaultLogger().Error(format, args...)
}

func Fatal(format string, args ...interface{}) {
	GetDefaultLogger().Fatal(format, args...)
}

// SetLevel 设置默认日志级别
func SetLevel(level LogLevel) {
	GetDefaultLogger().SetLevel(level)
}

// Close 关闭默认日志
func Close() error {
	return GetDefaultLogger().Close()
}
