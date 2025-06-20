/*
 * @Author: gaoyuan
 * @Date: 2025-06-17
 * @Description: 工具函数包，提供通用的工具函数
 */

package pkg

import (
	"regexp"
)

// 用于匹配ANSI控制序列的正则表达式 - 更全面的版本
var ansiRegex = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]|\[[0-9;]*[a-zA-Z]|\[[0-9]+G`)

// CleanAnsiSequences 清理字符串中的所有ANSI控制序列
func CleanAnsiSequences(s string) string {
	// 先使用正则表达式清理标准ANSI序列
	cleaned := ansiRegex.ReplaceAllString(s, "")

	// 处理特殊格式的输出，例如"[ OK ]"格式化
	//cleaned = strings.ReplaceAll(cleaned, "[ OK ]", "OK")

	return cleaned
}
