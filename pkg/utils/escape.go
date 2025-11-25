// 使用方法：SQL 标识符转义工具函数
// 防止 SQL 注入攻击，确保数据库/表名/字段名安全
package utils

import "strings"

// EscapeIdentifier 转义 ClickHouse 标识符（数据库名、表名、字段名）
// 使用反引号包裹，并转义内部的反引号
//
// 示例:
//   EscapeIdentifier("my_table") -> "`my_table`"
//   EscapeIdentifier("table`with`backticks") -> "`table``with``backticks`"
func EscapeIdentifier(identifier string) string {
	// 转义内部的反引号（`` -> ````）
	escaped := strings.ReplaceAll(identifier, "`", "``")
	// 用反引号包裹
	return "`" + escaped + "`"
}
