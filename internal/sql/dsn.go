/*
 * @Author: gaoyuan
 * @Date: 2025-06-17
 * @Description: 数据库连接字符串构建模块，提供安全的DSN构建功能，支持密码转义和连接选项配置
 */

package sql

import (
	"fmt"
	"net/url"
)

// buildDSN 构建数据库连接字符串，确保密码被正确转义
func buildDSN(user, password, host string, port int) string {
	// 转义密码中的特殊字符
	escapedPwd := url.QueryEscape(password)

	// 构建连接字符串
	return fmt.Sprintf("dm://%s:%s@%s:%d?autoCommit=true",
		user, escapedPwd, host, port)
}

// buildDSNWithOptions 构建带有额外选项的数据库连接字符串
func buildDSNWithOptions(user, password, host string, port int, options map[string]string) string {
	// 转义密码中的特殊字符
	escapedPwd := url.QueryEscape(password)

	// 构建基本连接字符串
	dsn := fmt.Sprintf("dm://%s:%s@%s:%d?autoCommit=true",
		user, escapedPwd, host, port)

	// 添加额外选项
	for key, value := range options {
		dsn += "&" + key + "=" + url.QueryEscape(value)
	}

	return dsn
}
