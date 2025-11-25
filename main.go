// 使用方法：ClickHouse TTL Tool 程序入口
// 执行: go run main.go [参数...]
// 或编译: go build -o clickhouse-ttl-tool && ./clickhouse-ttl-tool [参数...]
package main

import (
	"os"

	"clickhouse-ttl-tool/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
