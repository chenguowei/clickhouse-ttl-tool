// 使用方法：自动检测表中的时间字段
// 按优先级 timestamp -> event_time -> created_at 查找
// 支持 DateTime/DateTime64/UInt64(纳秒)类型
package detector

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"clickhouse-ttl-tool/pkg/client"
	"clickhouse-ttl-tool/pkg/utils"
)

var (
	// ErrNoTimeColumn 未找到合适的时间字段
	ErrNoTimeColumn = errors.New("no suitable time column found")
)

// Detector 时间字段检测器
type Detector struct {
	client *client.Client
}

// TimeColumn 时间字段信息
type TimeColumn struct {
	Name     string // 字段名
	Type     string // 字段类型（原始类型）
	IsUInt64 bool   // 是否为 UInt64 纳秒时间戳
}

// NewDetector 创建新的检测器
func NewDetector(client *client.Client) *Detector {
	return &Detector{
		client: client,
	}
}

// DetectTimeColumn 检测表的时间字段
// 如果提供了 preferredColumns，则优先检测这些列
func (d *Detector) DetectTimeColumn(ctx context.Context, database, table string, preferredColumns ...string) (*TimeColumn, error) {
	// 查询表的所有字段信息
	query := `
		SELECT
			name,
			type
		FROM system.columns
		WHERE database = ?
		  AND table = ?
		ORDER BY position
	`

	rows, err := d.client.Query(ctx, query, database, table)
	if err != nil {
		return nil, fmt.Errorf("failed to query columns: %w", err)
	}

	// 构建字段映射
	columns := make(map[string]string)
	for _, row := range rows {
		name, ok := row["name"].(string)
		if !ok {
			continue
		}
		colType, ok := row["type"].(string)
		if !ok {
			continue
		}
		columns[name] = colType
	}

	// 构建候选列表：优先使用 preferredColumns，然后是默认候选
	var candidates []string
	if len(preferredColumns) > 0 {
		candidates = preferredColumns
	} else {
		candidates = []string{"timestamp", "event_time", "created_at"}
	}

	// 按优先级检测时间字段
	for _, name := range candidates {
		colType, exists := columns[name]
		if !exists {
			continue
		}

		// 检查是否为时间类型
		if isDateTimeType(colType) {
			return &TimeColumn{
				Name:     name,
				Type:     colType,
				IsUInt64: false,
			}, nil
		}

		// 检查是否为 UInt64（可能是纳秒时间戳）
		if strings.HasPrefix(colType, "UInt64") {
			// 采样检查是否为纳秒级别时间戳
			isNano, err := d.isNanoTimestamp(ctx, database, table, name)
			if err != nil {
				// 采样失败，跳过此字段
				continue
			}

			if isNano {
				return &TimeColumn{
					Name:     name,
					Type:     colType,
					IsUInt64: true,
				}, nil
			}
		}
	}

	return nil, ErrNoTimeColumn
}

// isDateTimeType 判断是否为日期时间类型
func isDateTimeType(colType string) bool {
	// 支持 DateTime, DateTime64, Date 等类型
	return strings.HasPrefix(colType, "DateTime") ||
		strings.HasPrefix(colType, "Date")
}

// isNanoTimestamp 采样判断是否为纳秒级别时间戳
// 纳秒时间戳通常 > 1e17 (约 2001 年以后)
// 调整阈值从 1e15 到 1e17 以提高精确度
func (d *Detector) isNanoTimestamp(ctx context.Context, database, table, column string) (bool, error) {
	// 采样查询前 10 行，使用标识符转义防止 SQL 注入
	query := fmt.Sprintf(
		"SELECT %s FROM %s.%s WHERE %s > 0 LIMIT 10",
		utils.EscapeIdentifier(column),
		utils.EscapeIdentifier(database),
		utils.EscapeIdentifier(table),
		utils.EscapeIdentifier(column),
	)

	rows, err := d.client.Query(ctx, query)
	if err != nil {
		// 区分错误类型，提供更友好的错误信息
		return false, fmt.Errorf("failed to sample column %s: %w", column, err)
	}

	if len(rows) == 0 {
		// 表为空或字段全为 0，这是正常情况，不应该报错
		// 返回 false 表示无法判断，让调用方跳过此字段
		return false, fmt.Errorf("no data to sample in column %s", column)
	}

	// 检查采样值
	for _, row := range rows {
		val, ok := row[column]
		if !ok {
			continue
		}

		// 尝试转换为数值
		var numVal uint64
		switch v := val.(type) {
		case uint64:
			numVal = v
		case int64:
			numVal = uint64(v)
		case uint32:
			numVal = uint64(v)
		case int32:
			numVal = uint64(v)
		case uint:
			numVal = uint64(v)
		case int:
			numVal = uint64(v)
		default:
			continue
		}

		// 纳秒时间戳判断：> 1e17
		// 时间戳范围参考：
		//   秒级:     ~1e9  (2001年约 1000000000)
		//   毫秒级:   ~1e12 (2001年约 1000000000000)
		//   微秒级:   ~1e15 (2001年约 1000000000000000)
		//   纳秒级:   ~1e18 (2001年约 1000000000000000000)
		// 使用 1e17 作为阈值可以准确区分纳秒级别
		const nanoThreshold = 1e17
		if numVal > nanoThreshold {
			return true, nil
		}
	}

	return false, nil
}
