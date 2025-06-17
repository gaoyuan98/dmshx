/*
 * @Author: gaoyuan
 * @Date: 2025-06-17
 * @Description: 定义dmshx的数据模型和配置结构体，包括版本信息、配置参数、命令执行结果和SQL查询结果
 */

package pkg

import "time"

// 版本信息
var (
	Version   = "1.0.0"
	BuildTime = time.Now().Format("2006-01-02 15:04:05")
	Author    = "gaoyuan"
	BuildDate = "20250617"
)

// Config 命令行参数配置
type Config struct {
	// SSH相关参数
	Hosts    string
	HostFile string
	Port     int
	User     string
	Key      string
	Password string
	Cmd      string
	Timeout  int
	ExecUser string // 执行命令的用户，如果设置，将使用su切换到该用户执行命令

	// 数据库相关参数
	DBType string
	DBHost string
	DBPort int
	DBUser string
	DBPass string
	DBName string
	SQL    string

	// 输出相关参数
	JSONOutput bool
	LogFile    string
	Version    bool

	// 命令执行日志参数
	EnableCommandLog bool
	CommandLogPath   string
	LogRetention     int
}

// CmdResult 命令执行结果
type CmdResult struct {
	Host      string `json:"host"`
	Type      string `json:"type"`
	Status    string `json:"status"`
	Stdout    string `json:"stdout"`
	Stderr    string `json:"stderr"`
	Duration  string `json:"duration"`
	Error     string `json:"error,omitempty"`
	Timestamp string `json:"timestamp"`
	SSHUser   string `json:"ssh_user,omitempty"`   // SSH连接使用的用户
	ExecUser  string `json:"exec_user,omitempty"`  // 实际执行命令的用户
	ActualCmd string `json:"actual_cmd,omitempty"` // 实际执行的命令（可能是经过转换的）
}

// SQLResult SQL执行结果
type SQLResult struct {
	Host           string        `json:"host"`
	Type           string        `json:"type"`
	DB             string        `json:"db"`
	Status         string        `json:"status"`
	Rows           []interface{} `json:"rows"`
	Duration       string        `json:"duration"`
	Error          string        `json:"error,omitempty"`
	Timestamp      string        `json:"timestamp"`
	TimeoutSetting string        `json:"timeout_setting,omitempty"` // 超时设置信息
}
