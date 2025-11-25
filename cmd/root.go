// 使用方法：定义 CLI 命令行接口，集成所有模块
// 提供参数解析、执行流程控制和结果输出
package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"clickhouse-ttl-tool/pkg/client"
	"clickhouse-ttl-tool/pkg/config"
	"clickhouse-ttl-tool/pkg/detector"
	"clickhouse-ttl-tool/pkg/executor"
	"clickhouse-ttl-tool/pkg/reporter"
	"clickhouse-ttl-tool/pkg/scanner"

	"github.com/spf13/cobra"
)

var (
	// 配置参数
	cfg config.Config
)

// rootCmd 根命令
var rootCmd = &cobra.Command{
	Use:   "clickhouse-ttl-tool",
	Short: "为 ClickHouse 数据库中的所有表设置 TTL",
	Long: `ClickHouse TTL Tool - 批量设置数据保留策略

此工具自动扫描指定 ClickHouse 数据库中的所有表，
检测时间字段（timestamp/event_time/created_at），
并为每个表设置统一的 TTL 数据保留策略。

支持 DateTime/DateTime64/UInt64(纳秒) 类型的时间字段。`,
	Example: `  # 预览模式（不实际执行）
  clickhouse-ttl-tool --host localhost --database my_db --retention-days 30 --dry-run

  # 实际执行
  clickhouse-ttl-tool --host localhost --database my_db --retention-days 30

  # 使用环境变量配置密码
  export CH_PASSWORD="secret"
  clickhouse-ttl-tool --host localhost --database my_db --retention-days 30`,
	RunE: run,
}

// Execute 执行命令
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	// 连接参数
	rootCmd.Flags().StringVar(&cfg.Host, "host",
		config.GetEnvOrDefault("CH_HOST", "localhost"),
		"ClickHouse 服务器地址 (环境变量: CH_HOST)")

	rootCmd.Flags().IntVar(&cfg.Port, "port",
		9000,
		"ClickHouse Native 协议端口 (环境变量: CH_PORT)")

	rootCmd.Flags().StringVar(&cfg.User, "user",
		config.GetEnvOrDefault("CH_USER", "default"),
		"ClickHouse 用户名 (环境变量: CH_USER)")

	rootCmd.Flags().StringVar(&cfg.Password, "password",
		os.Getenv("CH_PASSWORD"),
		"ClickHouse 密码 (环境变量: CH_PASSWORD，推荐使用环境变量)")

	// 必填参数
	rootCmd.Flags().StringVar(&cfg.Database, "database", "",
		"目标数据库名 (必填)")
	rootCmd.MarkFlagRequired("database")

	rootCmd.Flags().IntVar(&cfg.RetentionDays, "retention-days", 0,
		"数据保留天数 (必填)")
	rootCmd.MarkFlagRequired("retention-days")

	// 可选参数
	rootCmd.Flags().BoolVar(&cfg.DryRun, "dry-run", false,
		"预览模式，仅显示将要执行的 SQL，不实际执行")

	rootCmd.Flags().BoolVar(&cfg.Verbose, "verbose", false,
		"详细输出，显示每个表的 SQL 语句")
}

// run 主执行函数
func run(cmd *cobra.Command, args []string) error {
	// 打印工具信息
	printHeader()

	// 验证配置
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("配置验证失败: %w", err)
	}

	// 打印配置信息
	printConfig()

	// 创建 ClickHouse 客户端
	fmt.Println("\n正在连接 ClickHouse...")
	cli, err := client.NewClient(&cfg)
	if err != nil {
		return fmt.Errorf("连接失败: %w", err)
	}
	defer cli.Close()
	fmt.Println("✓ 连接成功")

	// 创建上下文
	ctx := context.Background()

	// 扫描表
	fmt.Println("\n正在扫描数据库表...")
	scn := scanner.NewScanner(cli)
	tables, err := scn.ScanTables(ctx, cfg.Database)
	if err != nil {
		return fmt.Errorf("扫描表失败: %w", err)
	}
	fmt.Printf("✓ 找到 %d 个表\n", len(tables))

	if len(tables) == 0 {
		fmt.Println("\n⚠ 数据库中没有表，无需操作")
		return nil
	}

	// 显示表和时间列信息
	printTablesSummary(tables)

	// Dry-Run 模式提示
	if cfg.DryRun {
		fmt.Println("\n⚠️  预览模式：将显示 SQL 语句但不实际执行")
	} else {
		// 非 Dry-Run 模式，需要用户确认
		fmt.Println("\n" + strings.Repeat("=", 60))
		fmt.Println("⚠️  危险操作警告")
		fmt.Println(strings.Repeat("=", 60))
		fmt.Printf("\n将要执行的操作:\n")
		fmt.Printf("  • 数据库: %s\n", cfg.Database)
		fmt.Printf("  • 影响表数: %d 个\n", len(tables))
		fmt.Printf("  • 保留天数: %d 天\n", cfg.RetentionDays)
		fmt.Printf("  • 操作类型: 设置 TTL（数据超过 %d 天将被自动删除）\n\n", cfg.RetentionDays)
		fmt.Println("⚠️  注意: 此操作将覆盖已有的 TTL 设置，且数据删除不可逆！")
		fmt.Printf("\n请输入数据库名 '%s' 以确认操作: ", cfg.Database)

		var confirm string
		fmt.Scanln(&confirm)

		if confirm != cfg.Database {
			fmt.Println("\n✗ 确认失败，操作已取消")
			return nil
		}
		fmt.Println("\n✓ 确认成功，开始执行...")
	}

	// 创建检测器、执行器和报告器
	det := detector.NewDetector(cli)
	exec := executor.NewExecutor(cli, cfg.DryRun, cfg.Verbose)
	rep := reporter.NewReporter(cfg.Verbose, cfg.DryRun)

	// 执行主流程
	fmt.Println("\n开始处理...\n")

	for i, table := range tables {
		// 检测时间字段，优先使用 Scanner 找到的时间列
		timeCol, err := det.DetectTimeColumn(ctx, table.Database, table.Table, table.TimeColumns...)
		if err != nil {
			// 无时间字段，跳过
			skipReason := "未找到合适的时间字段"
			if len(table.TimeColumns) > 0 {
				skipReason = fmt.Sprintf("时间列 [%s] 验证失败", strings.Join(table.TimeColumns, ", "))
			}
			result := executor.ExecutionResult{
				Database:   table.Database,
				Table:      table.Table,
				Skipped:    true,
				SkipReason: skipReason,
			}
			rep.AddResult(result)
			rep.PrintProgress(i+1, len(tables), result)
			continue
		}

		// 执行 TTL 设置
		result := exec.Execute(ctx, table, timeCol, cfg.RetentionDays)
		rep.AddResult(result)
		rep.PrintProgress(i+1, len(tables), result)

		// 添加短暂延迟，避免对 ClickHouse 造成过大压力
		if !cfg.DryRun && i < len(tables)-1 {
			time.Sleep(100 * time.Millisecond)
		}
	}

	// 打印执行总结
	summary := rep.PrintSummary()

	// 根据结果返回退出码
	if summary.Failed > 0 {
		return errors.New("部分表执行失败")
	}

	if cfg.DryRun {
		fmt.Println("\n提示：去掉 --dry-run 参数以实际执行")
	}

	return nil
}

// printHeader 打印工具头部信息
func printHeader() {
	fmt.Println(strings.Repeat("=", 60))
	fmt.Println("ClickHouse TTL Tool v1.0.0")
	fmt.Println(strings.Repeat("=", 60))
}

// printConfig 打印配置信息
func printConfig() {
	fmt.Println("\n配置信息:")
	fmt.Printf("  连接地址: %s:%d\n", cfg.Host, cfg.Port)
	fmt.Printf("  数据库: %s\n", cfg.Database)
	fmt.Printf("  用户名: %s\n", cfg.User)
	fmt.Printf("  保留天数: %d 天\n", cfg.RetentionDays)
	if cfg.DryRun {
		fmt.Printf("  模式: 预览 (Dry-Run)\n")
	} else {
		fmt.Printf("  模式: 实际执行\n")
	}
}

// printTablesSummary 打印表及时间列摘要信息
func printTablesSummary(tables []scanner.TableInfo) {
	fmt.Println("\n表信息摘要:")
	fmt.Println(strings.Repeat("-", 80))
	fmt.Printf("%-40s %-20s %s\n", "表名", "引擎", "时间列")
	fmt.Println(strings.Repeat("-", 80))

	tablesWithTime := 0
	for _, table := range tables {
		timeColsStr := "无"
		if len(table.TimeColumns) > 0 {
			timeColsStr = strings.Join(table.TimeColumns, ", ")
			tablesWithTime++
		}

		// 截断过长的表名
		tableName := table.Table
		if len(tableName) > 37 {
			tableName = tableName[:34] + "..."
		}

		// 截断过长的引擎名
		engine := table.Engine
		if len(engine) > 17 {
			engine = engine[:14] + "..."
		}

		fmt.Printf("%-40s %-20s %s\n", tableName, engine, timeColsStr)
	}

	fmt.Println(strings.Repeat("-", 80))
	fmt.Printf("统计: 有时间列 %d 个 / 总计 %d 个表\n", tablesWithTime, len(tables))
}
