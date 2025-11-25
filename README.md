# ClickHouse TTL Tool

批量为 ClickHouse 数据库中的所有表设置 TTL（数据保留策略）工具。

## 功能特性

- ✅ 自动扫描数据库中的所有用户表
- ✅ 智能检测时间字段（按优先级：`timestamp` → `event_time` → `created_at`）
- ✅ 支持多种时间类型：`DateTime`、`DateTime64`、`UInt64`（纳秒时间戳）
- ✅ 统一设置数据保留天数
- ✅ Dry-Run 预览模式
- ✅ 详细的执行报告和统计
- ✅ 安全的环境变量配置

## 安装

### 前置要求

- Go 1.19 或更高版本
- ClickHouse 服务器（支持 TTL 功能）

### 编译

```bash
# 克隆或进入项目目录
cd clickhouse-ttl-tool

# 下载依赖
go mod tidy

# 编译
go build -o clickhouse-ttl-tool

# 或直接运行
go run main.go --help
```

## 使用方法

### 基本用法

```bash
# 预览模式（推荐先使用，查看将要执行的 SQL）
./clickhouse-ttl-tool \
  --host localhost \
  --port 9000 \
  --user default \
  --password "" \
  --database my_database \
  --retention-days 30 \
  --dry-run

# 实际执行
./clickhouse-ttl-tool \
  --host localhost \
  --database my_database \
  --retention-days 30
```

### 使用环境变量（推荐）

```bash
# 设置环境变量
export CH_HOST=192.168.1.100
export CH_PORT=9000
export CH_USER=admin
export CH_PASSWORD="MySecretPassword"

# 执行（密码不会出现在命令历史中）
./clickhouse-ttl-tool \
  --database production_db \
  --retention-days 90 \
  --verbose
```

### 参数说明

| 参数 | 类型 | 默认值 | 必填 | 说明 |
|------|------|--------|------|------|
| `--host` | string | `localhost` | 否 | ClickHouse 服务器地址 |
| `--port` | int | `9000` | 否 | Native 协议端口 |
| `--user` | string | `default` | 否 | 用户名 |
| `--password` | string | `""` | 否 | 密码（推荐用环境变量 `CH_PASSWORD`）|
| `--database` | string | - | **是** | 目标数据库名 |
| `--retention-days` | int | - | **是** | 数据保留天数 |
| `--dry-run` | bool | `false` | 否 | 预览模式，不实际执行 |
| `--verbose` | bool | `false` | 否 | 显示详细日志和 SQL 语句 |

## 工作原理

1. **连接数据库**：建立到 ClickHouse 的连接
2. **扫描表**：查询 `system.tables` 获取所有用户表（排除系统表和视图）
3. **检测时间字段**：按优先级查找 `timestamp` → `event_time` → `created_at`
4. **生成 TTL SQL**：
   - `DateTime/DateTime64`：`ALTER TABLE xxx MODIFY TTL column_name + INTERVAL N DAY`
   - `UInt64`（纳秒）：`ALTER TABLE xxx MODIFY TTL fromUnixTimestamp64Nano(column_name) + INTERVAL N DAY`
5. **执行或预览**：根据 `--dry-run` 参数决定是否实际执行
6. **输出报告**：显示成功/失败/跳过的统计

## 输出示例

### Dry-Run 模式

```
============================================================
ClickHouse TTL Tool v1.0.0
============================================================

配置信息:
  连接地址: localhost:9000
  数据库: test_db
  用户名: default
  保留天数: 30 天
  模式: 预览 (Dry-Run)

正在连接 ClickHouse...
✓ 连接成功

正在扫描数据库表...
✓ 找到 5 个表

⚠️  预览模式：将显示 SQL 语句但不实际执行

开始处理...

[1/5] test_db.logs_table
  ✓ 找到时间字段: timestamp (UInt64)
  → SQL: ALTER TABLE test_db.logs_table MODIFY TTL fromUnixTimestamp64Nano(timestamp) + INTERVAL 30 DAY
  ✓ TTL 设置成功

[2/5] test_db.events_table
  ✓ 找到时间字段: event_time (DateTime)
  → SQL: ALTER TABLE test_db.events_table MODIFY TTL event_time + INTERVAL 30 DAY
  ✓ TTL 设置成功

[3/5] test_db.metrics_table
  ✗ 跳过: 未找到合适的时间字段 (timestamp/event_time/created_at)

============================================================
执行总结
============================================================

总表数: 5
✓ 成功: 4
✗ 失败: 0
⊝ 跳过: 1 (无时间字段)

执行耗时: 0.52s

提示：去掉 --dry-run 参数以实际执行
```

## 时间字段检测规则

工具按以下优先级自动检测时间字段：

1. **timestamp**
   - 类型：`DateTime`、`DateTime64`、`UInt64`
   - `UInt64` 需满足：值 > 1e15（纳秒级别）

2. **event_time**
   - 类型：`DateTime`、`DateTime64`

3. **created_at**
   - 类型：`DateTime`、`DateTime64`

如果表中不存在上述任何字段，该表将被跳过。

## 注意事项

⚠️ **重要提醒**：

1. **数据删除不可逆**：TTL 设置后，超过保留期的数据将被 ClickHouse 自动删除
2. **先测试后执行**：务必先使用 `--dry-run` 预览
3. **已有 TTL 覆盖**：工具会覆盖表的现有 TTL 设置
4. **权限要求**：需要 `ALTER TABLE` 权限
5. **网络稳定性**：大量表时建议在稳定的网络环境下执行

## 安全建议

1. **使用环境变量存储密码**
   ```bash
   export CH_PASSWORD="your_password"
   ```

2. **最小权限原则**
   ```sql
   -- 创建专用账号，仅授予必要权限
   CREATE USER ttl_admin IDENTIFIED BY 'password';
   GRANT ALTER TABLE ON database.* TO ttl_admin;
   ```

3. **避免命令历史泄露**
   ```bash
   # 命令前加空格（某些 shell 配置下不会记录历史）
    ./clickhouse-ttl-tool --password "secret" ...
   ```

## 故障排查

### 连接失败

```
Error: 连接失败: dial tcp: connect: connection refused
```
**解决方案**：
- 检查 ClickHouse 服务是否运行：`systemctl status clickhouse-server`
- 确认端口号（默认 9000 为 Native 协议，8123 为 HTTP）
- 检查防火墙设置

### 权限不足

```
Error: 执行失败: Code: 497. DB::Exception: user: Access denied
```
**解决方案**：
- 确保用户有 `ALTER TABLE` 权限
- 检查用户密码是否正确

### UInt64 字段未识别为纳秒时间戳

**原因**：表数据为空或时间戳值 < 1e15

**解决方案**：
- 手动指定 TTL（工具暂不支持自定义字段）
- 检查数据格式是否正确

## 项目结构

```
clickhouse-ttl-tool/
├── main.go                      # 程序入口
├── go.mod                       # Go 模块定义
├── cmd/
│   └── root.go                 # CLI 命令实现
├── pkg/
│   ├── config/
│   │   └── config.go           # 配置管理
│   ├── client/
│   │   └── clickhouse.go       # ClickHouse 客户端
│   ├── scanner/
│   │   └── scanner.go          # 表扫描器
│   ├── detector/
│   │   └── detector.go         # 时间字段检测器
│   ├── executor/
│   │   └── executor.go         # TTL 执行器
│   └── reporter/
│       └── reporter.go         # 结果报告器
└── README.md                    # 本文档
```

## 依赖

- [clickhouse-go/v2](https://github.com/ClickHouse/clickhouse-go) - ClickHouse Go 驱动
- [cobra](https://github.com/spf13/cobra) - CLI 框架

## 许可证

MIT License

## 贡献

欢迎提交 Issue 和 Pull Request！

## 联系方式

如有问题或建议，请通过 Issue 联系。
