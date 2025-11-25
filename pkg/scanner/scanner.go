// 使用方法：扫描 ClickHouse 数据库中的所有用户表
// 过滤系统表和视图，仅返回物理数据表
package scanner

import (
	"context"
	"fmt"
	"strings"

	"clickhouse-ttl-tool/pkg/client"
)

// Scanner 表扫描器
type Scanner struct {
	client *client.Client
}

// TableInfo 表信息
type TableInfo struct {
	Database    string   // 数据库名
	Table       string   // 表名
	Engine      string   // 引擎类型
	TimeColumns []string // 时间类型列名（用于 TTL）
}

// NewScanner 创建新的扫描器
func NewScanner(client *client.Client) *Scanner {
	return &Scanner{
		client: client,
	}
}

// ScanTables 扫描指定数据库的所有用户表
func (s *Scanner) ScanTables(ctx context.Context, database string) ([]TableInfo, error) {
	query := `
		SELECT
			database,
			name as table,
			engine
		FROM system.tables
		WHERE database = ?
		  AND database NOT IN ('system', 'INFORMATION_SCHEMA', 'information_schema')
		  AND engine NOT IN ('View', 'MaterializedView', 'Dictionary', 'Distributed')
		ORDER BY name
	`

	rows, err := s.client.Query(ctx, query, database)
	if err != nil {
		return nil, fmt.Errorf("failed to scan tables: %w", err)
	}

	var tables []TableInfo
	for _, row := range rows {
		db, ok := row["database"].(string)
		if !ok {
			continue
		}

		table, ok := row["table"].(string)
		if !ok {
			continue
		}

		engine, ok := row["engine"].(string)
		if !ok {
			continue
		}

		// 额外过滤：跳过临时表和系统相关表
		if strings.HasPrefix(table, ".inner") || strings.HasPrefix(table, "system") {
			continue
		}

		// 扫描表中的时间列
		timeColumns, err := s.scanTimeColumns(ctx, db, table)
		if err != nil {
			// 记录错误但不中断扫描
			fmt.Printf("警告: 扫描表 %s.%s 的时间列失败: %v\n", db, table, err)
			timeColumns = []string{}
		}

		tables = append(tables, TableInfo{
			Database:    db,
			Table:       table,
			Engine:      engine,
			TimeColumns: timeColumns,
		})
	}

	return tables, nil
}

// scanTimeColumns 扫描指定表的时间类型列
// 包括 Date/DateTime 类型和常见时间戳字段名的 UInt64 类型
func (s *Scanner) scanTimeColumns(ctx context.Context, database, table string) ([]string, error) {
	query := `
		SELECT name, type
		FROM system.columns
		WHERE database = ?
		  AND table = ?
		  AND (
			  type LIKE 'Date%'
			  OR type LIKE 'DateTime%'
			  OR (
				  type LIKE 'UInt64%'
				  AND (
					  name = 'timestamp'
					  OR name = 'event_time'
					  OR name = 'created_at'
					  OR name = 'time'
				  )
			  )
		  )
		ORDER BY position
	`

	rows, err := s.client.Query(ctx, query, database, table)
	if err != nil {
		return nil, fmt.Errorf("query time columns failed: %w", err)
	}

	var columns []string
	seen := make(map[string]bool) // 去重
	for _, row := range rows {
		name, ok := row["name"].(string)
		if !ok || seen[name] {
			continue
		}
		columns = append(columns, name)
		seen[name] = true
	}

	return columns, nil
}
