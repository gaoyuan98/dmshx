/*
 * @Author: gaoyuan
 * @Date: 2025-06-17
 * @Description: 工具函数包，提供通用的工具函数
 */

package pkg

import (
	"regexp"
	"strconv"
)

// 用于匹配ANSI控制序列的正则表达式 - 更全面的版本
var ansiRegex = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]|\[[0-9;]*[a-zA-Z]|\[[0-9]+G`)

// 用于匹配Unicode转义序列的正则表达式，例如 \u003e
var unicodeEscapeRegex = regexp.MustCompile(`\\u([0-9a-fA-F]{4})`)

// CleanAnsiSequences 清理字符串中的所有ANSI控制序列
func CleanAnsiSequences(s string) string {
	// 先使用正则表达式清理标准ANSI序列
	cleaned := ansiRegex.ReplaceAllString(s, "")

	// 处理特殊格式的输出，例如"[ OK ]"格式化
	//cleaned = strings.ReplaceAll(cleaned, "[ OK ]", "OK")

	return cleaned
}

// UnescapeUnicode 将Unicode转义序列转换回普通字符
func UnescapeUnicode(s string) string {
	return unicodeEscapeRegex.ReplaceAllStringFunc(s, func(match string) string {
		// 从 \uXXXX 中提取4位16进制数
		hexStr := match[2:] // 去掉 \u 前缀
		// 将16进制字符串转换为整数
		i, err := strconv.ParseInt(hexStr, 16, 32)
		if err != nil {
			// 如果解析错误，保留原始字符串
			return match
		}
		// 将整数转换为对应的Unicode字符
		return string(rune(i))
	})
}

// CleanAndUnescapeText 清理ANSI控制序列并将Unicode转义序列转换回普通字符
// 此函数结合了CleanAnsiSequences和UnescapeUnicode的功能，用于一次性处理文本输出
// 参数:
//   - s: 需要处理的字符串
//
// 返回:
//   - 处理后的字符串，已移除ANSI控制序列并转换Unicode转义序列
func CleanAndUnescapeText(s string) string {
	// 先清理ANSI控制序列
	cleaned := CleanAnsiSequences(s)
	// 再转换Unicode转义序列
	return UnescapeUnicode(cleaned)
}
