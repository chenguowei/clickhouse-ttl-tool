// 使用方法：定义 ClickHouse TTL 工具的配置结构
// 支持命令行参数和环境变量配置
package config

import (
	"errors"
	"fmt"
	"os"
)

// Config 定义 ClickHouse 连接和 TTL 设置的配置
type Config struct {
	Host          string // ClickHouse 服务器地址
	Port          int    // ClickHouse Native 协议端口
	User          string // 用户名
	Password      string // 密码
	Database      string // 目标数据库名
	RetentionDays int    // 数据保留天数
	DryRun        bool   // 是否为预览模式（不实际执行）
	Verbose       bool   // 是否输出详细日志
}

// Validate 验证配置的完整性和合法性
func (c *Config) Validate() error {
	if c.Host == "" {
		return errors.New("host cannot be empty")
	}

	if c.Port <= 0 || c.Port > 65535 {
		return fmt.Errorf("invalid port: %d, must be between 1 and 65535", c.Port)
	}

	if c.User == "" {
		return errors.New("user cannot be empty")
	}

	if c.Database == "" {
		return errors.New("database cannot be empty")
	}

	if c.RetentionDays <= 0 {
		return fmt.Errorf("invalid retention days: %d, must be greater than 0", c.RetentionDays)
	}

	return nil
}

// GetEnvOrDefault 获取环境变量，如果不存在则返回默认值
func GetEnvOrDefault(key, defaultValue string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultValue
}
