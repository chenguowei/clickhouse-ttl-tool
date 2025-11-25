// 使用方法：格式化输出执行结果和统计摘要
// 提供清晰的进度显示和最终报告
package reporter

import (
	"fmt"
	"strings"
	"time"

	"clickhouse-ttl-tool/pkg/executor"
)

// Reporter 报告生成器
type Reporter struct {
	results   []executor.ExecutionResult
	startTime time.Time
	verbose   bool
	dryRun    bool
}

// Summary 执行统计摘要
type Summary struct {
	Total    int           // 总表数
	Success  int           // 成功数
	Failed   int           // 失败数
	Skipped  int           // 跳过数
	Duration time.Duration // 执行耗时
}

// NewReporter 创建新的报告器
func NewReporter(verbose, dryRun bool) *Reporter {
	return &Reporter{
		results:   make([]executor.ExecutionResult, 0),
		startTime: time.Now(),
		verbose:   verbose,
		dryRun:    dryRun,
	}
}

// AddResult 添加执行结果
func (r *Reporter) AddResult(result executor.ExecutionResult) {
	r.results = append(r.results, result)
}

// PrintProgress 打印单个表的执行进度
func (r *Reporter) PrintProgress(index, total int, result executor.ExecutionResult) {
	// 打印进度头
	fmt.Printf("\n[%d/%d] %s.%s\n", index, total, result.Database, result.Table)

	// 跳过的表
	if result.Skipped {
		fmt.Printf("  ✗ 跳过: %s\n", result.SkipReason)
		return
	}

	// 显示找到的时间字段
	timeTypeDesc := result.TimeType
	if result.TimeType != "" {
		// 提取简化的类型名
		timeTypeDesc = result.TimeType
	}
	fmt.Printf("  ✓ 找到时间字段: %s (%s)\n", result.TimeColumn, timeTypeDesc)

	// Dry-Run 模式或详细模式：显示 SQL
	if r.dryRun || r.verbose {
		fmt.Printf("  → SQL: %s\n", result.SQL)
	}

	// 显示执行结果
	if result.Success {
		if r.dryRun {
			fmt.Printf("  ✓ 预览成功 (未执行)\n")
		} else {
			fmt.Printf("  ✓ TTL 设置成功\n")
		}
	} else if result.Error != nil {
		fmt.Printf("  ✗ 执行失败: %v\n", result.Error)
	}
}

// PrintSummary 打印执行统计摘要
func (r *Reporter) PrintSummary() Summary {
	duration := time.Since(r.startTime)

	summary := Summary{
		Total:    len(r.results),
		Duration: duration,
	}

	// 统计各状态数量
	for _, result := range r.results {
		if result.Skipped {
			summary.Skipped++
		} else if result.Success {
			summary.Success++
		} else {
			summary.Failed++
		}
	}

	// 打印分隔线
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("执行总结")
	fmt.Println(strings.Repeat("=", 60))

	// 打印统计信息
	fmt.Printf("\n总表数: %d\n", summary.Total)
	fmt.Printf("✓ 成功: %d\n", summary.Success)
	fmt.Printf("✗ 失败: %d\n", summary.Failed)
	fmt.Printf("⊝ 跳过: %d (无时间字段)\n", summary.Skipped)
	fmt.Printf("\n执行耗时: %.2fs\n", duration.Seconds())

	// 如果有失败的表，列出详情
	if summary.Failed > 0 {
		fmt.Println("\n失败的表:")
		for _, result := range r.results {
			if !result.Success && !result.Skipped {
				fmt.Printf("  - %s.%s: %v\n", result.Database, result.Table, result.Error)
			}
		}
	}

	return summary
}

// GetResults 获取所有执行结果
func (r *Reporter) GetResults() []executor.ExecutionResult {
	return r.results
}
