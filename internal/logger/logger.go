/*
 * @Author: gaoyuan
 * @Date: 2025-06-17
 * @Description: 日志记录模块，负责记录SSH命令和SQL查询的执行结果，支持按日期组织日志文件和自动清理过期日志
 */

package logger

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"dmshx/pkg"
)

// Logger 命令执行日志记录器
type Logger struct {
	config          *pkg.Config
	logPath         string
	lastCleanupTime time.Time
}

// NewLogger 创建一个新的日志记录器
func NewLogger(config *pkg.Config) *Logger {
	logger := &Logger{
		config:          config,
		lastCleanupTime: time.Now(),
	}

	// 确保日志目录存在
	if config.EnableCommandLog {
		err := os.MkdirAll(config.CommandLogPath, 0755)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating log directory: %v\n", err)
		}
	}

	// 启动时清理过期日志
	logger.CleanupExpiredLogs()

	return logger
}

// LogCommand 记录SSH命令执行结果
func (l *Logger) LogCommand(result *pkg.CmdResult) {
	if !l.config.EnableCommandLog {
		return
	}

	// 设置时间戳
	now := time.Now()
	result.Timestamp = now.Format("2006-01-02 15:04:05")

	// 创建日期目录
	dateDir := filepath.Join(l.config.CommandLogPath, now.Format("2006-01-02"))
	err := os.MkdirAll(dateDir, 0755)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating date directory for logs: %v\n", err)
		return
	}

	// 创建日志文件
	logFilePath := filepath.Join(dateDir, fmt.Sprintf("command_%s.log", now.Format("150405.000")))
	logFile, err := os.Create(logFilePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating log file: %v\n", err)
		return
	}
	defer logFile.Close()

	// 添加UTF-8 BOM，解决中文显示问题
	logFile.Write([]byte{0xEF, 0xBB, 0xBF})

	// 写入日志内容
	fmt.Fprintf(logFile, "执行时间: %s\n", result.Timestamp)
	fmt.Fprintf(logFile, "命令类型: SSH\n")
	fmt.Fprintf(logFile, "目标主机: %s\n", result.Host)
	fmt.Fprintf(logFile, "SSH用户: %s\n", result.SSHUser)
	if result.ExecUser != result.SSHUser {
		fmt.Fprintf(logFile, "执行用户: %s\n", result.ExecUser)
	}
	fmt.Fprintf(logFile, "原始命令: %s\n", l.config.Cmd)

	// 如果有实际执行命令（可能是包装后的命令）
	if result.ActualCmd != "" && result.ActualCmd != l.config.Cmd {
		fmt.Fprintf(logFile, "实际命令: %s\n", result.ActualCmd)
	}

	// 添加超时设置信息
	if result.TimeoutSetting != "" {
		fmt.Fprintf(logFile, "超时设置: %s\n", result.TimeoutSetting)
	}

	fmt.Fprintf(logFile, "执行状态: %s\n", result.Status)
	fmt.Fprintf(logFile, "执行耗时: %s\n", result.Duration)
	fmt.Fprintf(logFile, "标准输出:\n%s\n", pkg.CleanAnsiSequences(result.Stdout))
	if result.Stderr != "" {
		fmt.Fprintf(logFile, "标准错误:\n%s\n", pkg.CleanAnsiSequences(result.Stderr))
	}
	if result.Error != "" {
		fmt.Fprintf(logFile, "错误信息: %s\n", result.Error)
	}

	// 根据LogRetention设置的天数检查是否需要清理日志
	cleanupInterval := time.Duration(l.config.LogRetention) * 24 * time.Hour
	if time.Since(l.lastCleanupTime) > cleanupInterval {
		l.CleanupExpiredLogs()
		l.lastCleanupTime = time.Now()
	}
}

// LogSQL 记录SQL查询执行结果
func (l *Logger) LogSQL(result *pkg.SQLResult) {
	if !l.config.EnableCommandLog {
		return
	}

	// 设置时间戳
	now := time.Now()
	result.Timestamp = now.Format("2006-01-02 15:04:05")

	// 创建日期目录
	dateDir := filepath.Join(l.config.CommandLogPath, now.Format("2006-01-02"))
	err := os.MkdirAll(dateDir, 0755)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating date directory for logs: %v\n", err)
		return
	}

	// 创建日志文件
	logFilePath := filepath.Join(dateDir, fmt.Sprintf("sql_%s.log", now.Format("150405.000")))
	logFile, err := os.Create(logFilePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating log file: %v\n", err)
		return
	}
	defer logFile.Close()

	// 添加UTF-8 BOM，解决中文显示问题
	logFile.Write([]byte{0xEF, 0xBB, 0xBF})

	// 写入日志内容
	fmt.Fprintf(logFile, "执行时间: %s\n", result.Timestamp)
	fmt.Fprintf(logFile, "命令类型: SQL (%s)\n", result.DB)
	fmt.Fprintf(logFile, "目标主机: %s\n", result.Host)
	fmt.Fprintf(logFile, "执行SQL: %s\n", l.config.SQL)

	if result.TimeoutSetting != "" {
		fmt.Fprintf(logFile, "超时设置: %s\n", result.TimeoutSetting)
	}

	fmt.Fprintf(logFile, "执行状态: %s\n", result.Status)
	fmt.Fprintf(logFile, "执行耗时: %s\n", result.Duration)

	if result.Status == "success" && len(result.Rows) > 0 {
		rows, _ := json.MarshalIndent(result.Rows, "", "  ")
		fmt.Fprintf(logFile, "查询结果:\n%s\n", string(rows))
	}

	if result.Error != "" {
		fmt.Fprintf(logFile, "错误信息: %s\n", result.Error)
	}

	// 根据LogRetention设置的天数检查是否需要清理日志
	cleanupInterval := time.Duration(l.config.LogRetention) * 24 * time.Hour
	if time.Since(l.lastCleanupTime) > cleanupInterval {
		l.CleanupExpiredLogs()
		l.lastCleanupTime = time.Now()
	}
}

// LogUpload 记录文件上传结果
func (l *Logger) LogUpload(result *pkg.UploadResult) {
	if !l.config.EnableCommandLog {
		return
	}

	// 设置时间戳
	now := time.Now()
	result.Timestamp = now.Format("2006-01-02 15:04:05")

	// 创建日期目录
	dateDir := filepath.Join(l.config.CommandLogPath, now.Format("2006-01-02"))
	err := os.MkdirAll(dateDir, 0755)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating date directory for logs: %v\n", err)
		return
	}

	// 创建日志文件
	logFilePath := filepath.Join(dateDir, fmt.Sprintf("upload_%s.log", now.Format("150405.000")))
	logFile, err := os.Create(logFilePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating log file: %v\n", err)
		return
	}
	defer logFile.Close()

	// 添加UTF-8 BOM，解决中文显示问题
	logFile.Write([]byte{0xEF, 0xBB, 0xBF})

	// 写入日志内容
	fmt.Fprintf(logFile, "执行时间: %s\n", result.Timestamp)
	fmt.Fprintf(logFile, "命令类型: 文件上传\n")
	fmt.Fprintf(logFile, "目标主机: %s\n", result.Host)
	fmt.Fprintf(logFile, "SSH用户: %s\n", result.SSHUser)
	fmt.Fprintf(logFile, "本地文件: %s\n", result.LocalFile)
	fmt.Fprintf(logFile, "远程文件: %s\n", result.RemoteFile)
	fmt.Fprintf(logFile, "文件大小: %d字节\n", result.Size)

	if result.TimeoutSetting != "" {
		fmt.Fprintf(logFile, "超时设置: %s\n", result.TimeoutSetting)
	}

	fmt.Fprintf(logFile, "执行状态: %s\n", result.Status)
	fmt.Fprintf(logFile, "执行耗时: %s\n", result.Duration)

	if result.Error != "" {
		fmt.Fprintf(logFile, "错误信息: %s\n", result.Error)
	}

	// 根据LogRetention设置的天数检查是否需要清理日志
	cleanupInterval := time.Duration(l.config.LogRetention) * 24 * time.Hour
	if time.Since(l.lastCleanupTime) > cleanupInterval {
		l.CleanupExpiredLogs()
		l.lastCleanupTime = time.Now()
	}
}

// LogDownload 记录下载文件结果
func (l *Logger) LogDownload(result *pkg.DownloadResult) {
	if !l.config.EnableCommandLog {
		return
	}

	// 设置时间戳
	now := time.Now()
	result.Timestamp = now.Format("2006-01-02 15:04:05")

	// 创建日期目录
	dateDir := filepath.Join(l.config.CommandLogPath, now.Format("2006-01-02"))
	err := os.MkdirAll(dateDir, 0755)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating date directory for logs: %v\n", err)
		return
	}

	// 创建日志文件
	logFilePath := filepath.Join(dateDir, fmt.Sprintf("download_%s.log", now.Format("150405.000")))
	logFile, err := os.Create(logFilePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating log file: %v\n", err)
		return
	}
	defer logFile.Close()

	// 添加UTF-8 BOM，解决中文显示问题
	logFile.Write([]byte{0xEF, 0xBB, 0xBF})

	// 写入日志内容
	fmt.Fprintf(logFile, "执行时间: %s\n", result.Timestamp)
	fmt.Fprintf(logFile, "命令类型: 文件下载\n")
	fmt.Fprintf(logFile, "目标主机: %s\n", result.Host)
	fmt.Fprintf(logFile, "SSH用户: %s\n", result.SSHUser)
	fmt.Fprintf(logFile, "远程文件: %s\n", result.RemotePath)
	fmt.Fprintf(logFile, "本地文件: %s\n", result.LocalPath)
	fmt.Fprintf(logFile, "文件大小: %d字节\n", result.Size)

	if result.MD5 != "" {
		fmt.Fprintf(logFile, "MD5校验和: %s\n", result.MD5)
	}

	if result.TimeoutSetting != "" {
		fmt.Fprintf(logFile, "超时设置: %s\n", result.TimeoutSetting)
	}

	fmt.Fprintf(logFile, "执行状态: %s\n", result.Status)
	fmt.Fprintf(logFile, "执行耗时: %s\n", result.Duration)

	if result.Error != "" {
		fmt.Fprintf(logFile, "错误信息: %s\n", result.Error)
	}

	// 根据LogRetention设置的天数检查是否需要清理日志
	cleanupInterval := time.Duration(l.config.LogRetention) * 24 * time.Hour
	if time.Since(l.lastCleanupTime) > cleanupInterval {
		l.CleanupExpiredLogs()
		l.lastCleanupTime = time.Now()
	}
}

// CleanupExpiredLogs 清理过期日志文件
func (l *Logger) CleanupExpiredLogs() {
	if !l.config.EnableCommandLog || l.config.LogRetention <= 0 {
		return
	}

	// 计算过期日期
	cutoffDate := time.Now().AddDate(0, 0, -l.config.LogRetention)
	cutoffDateStr := cutoffDate.Format("2006-01-02")

	// 遍历日志目录
	err := filepath.Walk(l.config.CommandLogPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 跳过根目录
		if path == l.config.CommandLogPath {
			return nil
		}

		// 如果是目录，检查名称是否为日期格式
		if info.IsDir() {
			dirName := filepath.Base(path)
			// 检查是否为日期目录
			if len(dirName) == 10 && strings.Count(dirName, "-") == 2 {
				// 如果目录日期早于保留期，则删除整个目录
				if dirName < cutoffDateStr {
					fmt.Printf("清理过期日志目录: %s\n", path)
					return os.RemoveAll(path)
				}
			}
		}
		return nil
	})

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error cleaning up expired logs: %v\n", err)
	}
}
