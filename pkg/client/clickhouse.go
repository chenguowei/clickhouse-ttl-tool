// 使用方法：封装 ClickHouse 连接和查询操作
// 提供统一的查询和 DDL 执行接口
package client

import (
	"context"
	"fmt"
	"strings"
	"time"

	"clickhouse-ttl-tool/pkg/config"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

// Client ClickHouse 客户端封装
type Client struct {
	conn driver.Conn
	db   string
}

// NewClient 创建新的 ClickHouse 客户端连接
func NewClient(cfg *config.Config) (*Client, error) {
	conn, err := clickhouse.Open(&clickhouse.Options{
		Addr: []string{fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)},
		Auth: clickhouse.Auth{
			Database: cfg.Database,
			Username: cfg.User,
			Password: cfg.Password,
		},
		Settings: clickhouse.Settings{
			"max_execution_time": 60,
		},
		DialTimeout: 10 * time.Second,
		Compression: &clickhouse.Compression{
			Method: clickhouse.CompressionLZ4,
		},
	})

	if err != nil {
		return nil, fmt.Errorf("failed to connect to ClickHouse: %w", err)
	}

	// 验证连接
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := conn.Ping(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping ClickHouse: %w", err)
	}

	return &Client{
		conn: conn,
		db:   cfg.Database,
	}, nil
}

// Close 关闭连接
func (c *Client) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// createScanTarget 根据列类型创建适合的扫描目标
func createScanTarget(dbType string) interface{} {
	// 处理 Nullable 类型
	if strings.HasPrefix(dbType, "Nullable(") {
		innerType := strings.TrimPrefix(dbType, "Nullable(")
		innerType = strings.TrimSuffix(innerType, ")")
		return createNullableScanTarget(innerType)
	}

	// 基础类型映射
	switch {
	case strings.HasPrefix(dbType, "String"), strings.HasPrefix(dbType, "FixedString"):
		var v string
		return &v
	case strings.HasPrefix(dbType, "Int64"):
		var v int64
		return &v
	case strings.HasPrefix(dbType, "Int32"):
		var v int32
		return &v
	case strings.HasPrefix(dbType, "Int16"):
		var v int16
		return &v
	case strings.HasPrefix(dbType, "Int8"):
		var v int8
		return &v
	case strings.HasPrefix(dbType, "UInt64"):
		var v uint64
		return &v
	case strings.HasPrefix(dbType, "UInt32"):
		var v uint32
		return &v
	case strings.HasPrefix(dbType, "UInt16"):
		var v uint16
		return &v
	case strings.HasPrefix(dbType, "UInt8"):
		var v uint8
		return &v
	case strings.HasPrefix(dbType, "Float64"):
		var v float64
		return &v
	case strings.HasPrefix(dbType, "Float32"):
		var v float32
		return &v
	case strings.HasPrefix(dbType, "DateTime"), strings.HasPrefix(dbType, "Date"):
		var v time.Time
		return &v
	default:
		// 默认使用 string
		var v string
		return &v
	}
}

// createNullableScanTarget 为 Nullable 类型创建扫描目标
func createNullableScanTarget(innerType string) interface{} {
	switch {
	case strings.HasPrefix(innerType, "String"), strings.HasPrefix(innerType, "FixedString"):
		var v *string
		return &v
	case strings.HasPrefix(innerType, "Int64"):
		var v *int64
		return &v
	case strings.HasPrefix(innerType, "Int32"):
		var v *int32
		return &v
	case strings.HasPrefix(innerType, "UInt64"):
		var v *uint64
		return &v
	case strings.HasPrefix(innerType, "UInt32"):
		var v *uint32
		return &v
	case strings.HasPrefix(innerType, "Float64"):
		var v *float64
		return &v
	case strings.HasPrefix(innerType, "Float32"):
		var v *float32
		return &v
	default:
		var v *string
		return &v
	}
}

// Query 执行查询并返回结果
func (c *Client) Query(ctx context.Context, query string, args ...interface{}) ([]map[string]interface{}, error) {
	rows, err := c.conn.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	// 获取列信息
	columnTypes := rows.ColumnTypes()
	columnCount := len(columnTypes)

	if columnCount == 0 {
		return []map[string]interface{}{}, nil
	}

	var results []map[string]interface{}
	for rows.Next() {
		// 为每列创建适合其类型的扫描目标
		scanTargets := make([]interface{}, columnCount)
		for i, colType := range columnTypes {
			scanTargets[i] = createScanTarget(colType.DatabaseTypeName())
		}

		// 扫描当前行
		if err := rows.Scan(scanTargets...); err != nil {
			return nil, fmt.Errorf("scan failed: %w", err)
		}

		// 构建结果 map，解引用指针获取实际值
		row := make(map[string]interface{})
		for i, colType := range columnTypes {
			row[colType.Name()] = derefValue(scanTargets[i])
		}

		results = append(results, row)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return results, nil
}

// derefValue 解引用指针，获取实际值
func derefValue(ptr interface{}) interface{} {
	switch v := ptr.(type) {
	case *string:
		return *v
	case *int64:
		return *v
	case *int32:
		return *v
	case *int16:
		return *v
	case *int8:
		return *v
	case *uint64:
		return *v
	case *uint32:
		return *v
	case *uint16:
		return *v
	case *uint8:
		return *v
	case *float64:
		return *v
	case *float32:
		return *v
	case *time.Time:
		return *v
	case **string:
		if v == nil || *v == nil {
			return nil
		}
		return **v
	case **int64:
		if v == nil || *v == nil {
			return nil
		}
		return **v
	case **int32:
		if v == nil || *v == nil {
			return nil
		}
		return **v
	case **uint64:
		if v == nil || *v == nil {
			return nil
		}
		return **v
	case **uint32:
		if v == nil || *v == nil {
			return nil
		}
		return **v
	case **float64:
		if v == nil || *v == nil {
			return nil
		}
		return **v
	case **float32:
		if v == nil || *v == nil {
			return nil
		}
		return **v
	default:
		return ptr
	}
}

// Exec 执行 DDL 或 DML 语句（如 ALTER TABLE）
func (c *Client) Exec(ctx context.Context, query string, args ...interface{}) error {
	if err := c.conn.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("exec failed: %w", err)
	}
	return nil
}

// GetDatabase 获取当前连接的数据库名
func (c *Client) GetDatabase() string {
	return c.db
}
