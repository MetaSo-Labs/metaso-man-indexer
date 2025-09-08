package logger

import (
	"os"
	"strconv"
)

// LogConfig 日志配置结构体
type LogConfig struct {
	// 日志目录
	LogDir string
	// 日志级别 (DEBUG, INFO, WARN, ERROR, FATAL)
	Level string
	// 最大文件大小（字节）
	MaxSize int64
	// 最大备份文件数量
	MaxBackups int
	// 最大保存天数
	MaxAge int
	// 是否输出到控制台
	ConsoleOutput bool
	// 是否输出到文件
	FileOutput bool
}

// DefaultLogConfig 返回默认日志配置
func DefaultLogConfig() *LogConfig {
	return &LogConfig{
		LogDir:        "./logs",
		Level:         "INFO",
		MaxSize:       10 * 1024 * 1024, // 10MB
		MaxBackups:    5,
		MaxAge:        30,
		ConsoleOutput: true,
		FileOutput:    true,
	}
}

// LoadLogConfigFromEnv 从环境变量加载日志配置
func LoadLogConfigFromEnv() *LogConfig {
	config := DefaultLogConfig()

	// 从环境变量读取配置
	if logDir := os.Getenv("LOG_DIR"); logDir != "" {
		config.LogDir = logDir
	}

	if level := os.Getenv("LOG_LEVEL"); level != "" {
		config.Level = level
	}

	if maxSizeStr := os.Getenv("LOG_MAX_SIZE"); maxSizeStr != "" {
		if maxSize, err := strconv.ParseInt(maxSizeStr, 10, 64); err == nil {
			config.MaxSize = maxSize
		}
	}

	if maxBackupsStr := os.Getenv("LOG_MAX_BACKUPS"); maxBackupsStr != "" {
		if maxBackups, err := strconv.Atoi(maxBackupsStr); err == nil {
			config.MaxBackups = maxBackups
		}
	}

	if maxAgeStr := os.Getenv("LOG_MAX_AGE"); maxAgeStr != "" {
		if maxAge, err := strconv.Atoi(maxAgeStr); err == nil {
			config.MaxAge = maxAge
		}
	}

	if consoleOutputStr := os.Getenv("LOG_CONSOLE_OUTPUT"); consoleOutputStr != "" {
		if consoleOutput, err := strconv.ParseBool(consoleOutputStr); err == nil {
			config.ConsoleOutput = consoleOutput
		}
	}

	if fileOutputStr := os.Getenv("LOG_FILE_OUTPUT"); fileOutputStr != "" {
		if fileOutput, err := strconv.ParseBool(fileOutputStr); err == nil {
			config.FileOutput = fileOutput
		}
	}

	return config
}

// ParseLogLevel 解析日志级别字符串
func ParseLogLevel(levelStr string) LogLevel {
	switch levelStr {
	case "DEBUG":
		return DEBUG
	case "INFO":
		return INFO
	case "WARN":
		return WARN
	case "ERROR":
		return ERROR
	case "FATAL":
		return FATAL
	default:
		return INFO // 默认级别
	}
}

// InitLoggerWithConfig 使用配置初始化日志
func InitLoggerWithConfig(config *LogConfig) error {
	// 解析日志级别
	level := ParseLogLevel(config.Level)

	// 初始化日志
	err := InitDefaultLogger(config.LogDir, level)
	if err != nil {
		return err
	}

	// 设置日志配置
	logger := GetDefaultLogger()
	logger.SetMaxSize(config.MaxSize)
	logger.SetMaxBackups(config.MaxBackups)
	logger.SetMaxAge(config.MaxAge)

	return nil
}

// InitLoggerFromEnv 从环境变量初始化日志
func InitLoggerFromEnv() error {
	config := LoadLogConfigFromEnv()
	return InitLoggerWithConfig(config)
}
