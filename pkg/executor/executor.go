// 使用方法：生成并执行 ALTER TABLE MODIFY TTL 语句
// 支持 DateTime/DateTime64 和 UInt64(纳秒) 类型的时间字段
package executor

import (
	"context"
	"fmt"
	"strings"

	"clickhouse-ttl-tool/pkg/client"
	"clickhouse-ttl-tool/pkg/detector"
	"clickhouse-ttl-tool/pkg/scanner"
	"clickhouse-ttl-tool/pkg/utils"
)

// Executor TTL 执行器
type Executor struct {
	client  *client.Client
	dryRun  bool
	verbose bool
}

// ExecutionResult 执行结果
type ExecutionResult struct {
	Database   string // 数据库名
	Table      string // 表名
	TimeColumn string // 时间字段名
	TimeType   string // 时间字段类型
	SQL        string // 生成的 SQL 语句
	Success    bool   // 是否执行成功
	Error      error  // 错误信息
	Skipped    bool   // 是否跳过
	SkipReason string // 跳过原因
}

// NewExecutor 创建新的执行器
func NewExecutor(client *client.Client, dryRun, verbose bool) *Executor {
	return &Executor{
		client:  client,
		dryRun:  dryRun,
		verbose: verbose,
	}
}

// Execute 执行 TTL 设置
func (e *Executor) Execute(
	ctx context.Context,
	table scanner.TableInfo,
	timeCol *detector.TimeColumn,
	retentionDays int,
) ExecutionResult {
	result := ExecutionResult{
		Database:   table.Database,
		Table:      table.Table,
		TimeColumn: timeCol.Name,
		TimeType:   timeCol.Type,
		Success:    false,
	}

	// 生成 TTL SQL
	sql := e.generateTTLSQL(table.Database, table.Table, timeCol, retentionDays)
	result.SQL = sql

	// Dry-Run 模式：仅记录 SQL，不执行
	if e.dryRun {
		result.Success = true
		return result
	}

	// 执行 ALTER TABLE 语句
	if err := e.client.Exec(ctx, sql); err != nil {
		result.Error = fmt.Errorf("failed to execute TTL: %w", err)
		result.Success = false
		return result
	}

	result.Success = true
	return result
}

// generateTTLSQL 生成 TTL SQL 语句
// 使用标识符转义防止 SQL 注入
func (e *Executor) generateTTLSQL(
	database, table string,
	col *detector.TimeColumn,
	days int,
) string {
	// 转义所有标识符
	dbEscaped := utils.EscapeIdentifier(database)
	tableEscaped := utils.EscapeIdentifier(table)
	colEscaped := utils.EscapeIdentifier(col.Name)

	// UInt64 纳秒时间戳：需要转换为 DateTime（除以 10^9 转为秒）
	// fromUnixTimestamp64Nano 返回 DateTime64(9)，TTL 不支持，所以用除法+toDateTime
	if col.IsUInt64 {
		return fmt.Sprintf(
			"ALTER TABLE %s.%s MODIFY TTL toDateTime(%s / 1000000000) + INTERVAL %d DAY",
			dbEscaped, tableEscaped, colEscaped, days,
		)
	}

	// DateTime64 类型：需要转换为 DateTime（TTL 表达式不支持 DateTime64）
	if strings.HasPrefix(col.Type, "DateTime64") {
		return fmt.Sprintf(
			"ALTER TABLE %s.%s MODIFY TTL toDateTime(%s) + INTERVAL %d DAY",
			dbEscaped, tableEscaped, colEscaped, days,
		)
	}

	// DateTime/Date 类型：直接使用
	return fmt.Sprintf(
		"ALTER TABLE %s.%s MODIFY TTL %s + INTERVAL %d DAY",
		dbEscaped, tableEscaped, colEscaped, days,
	)
}
