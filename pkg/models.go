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

	// 文件上传相关参数
	UploadFile       string // 要上传的本地文件路径
	UploadDir        string // 远程目标目录
	UploadPermission int    // 上传文件的权限（默认0644）

	// 文件下载相关参数
	RemotePath string // 要下载的远程文件或目录路径
	LocalPath  string // 本地保存目录
	VerifyMD5  bool   // 是否验证MD5校验和
	BufferSize int64  // 下载缓冲区大小(MB)

	// 数据库相关参数
	DBType string
	DBHost string
	DBPort int
	DBUser string
	DBPass string
	DBName string
	SQL    string

	// 输出相关参数
	JSONOutput     bool
	LogFile        string
	Version        bool
	RealTimeOutput bool // 是否启用实时输出，在非JSON模式下有效
	EnableUTF8     bool // 是否启用UTF-8编码输出

	// 命令执行日志参数
	EnableCommandLog bool
	CommandLogPath   string
	LogRetention     int // 日志保留天数，同时作为日志清理检查间隔
}

// CmdResult 命令执行结果
type CmdResult struct {
	Host           string `json:"host"`
	Type           string `json:"type"`
	Status         string `json:"status"`
	Stdout         string `json:"stdout"`
	Stderr         string `json:"stderr"`
	Duration       string `json:"duration"`
	Error          string `json:"error,omitempty"`
	Timestamp      string `json:"timestamp"`
	SSHUser        string `json:"ssh_user,omitempty"`        // SSH连接使用的用户
	ExecUser       string `json:"exec_user,omitempty"`       // 实际执行命令的用户
	ActualCmd      string `json:"actual_cmd,omitempty"`      // 实际执行的命令（可能是经过转换的）
	TimeoutSetting string `json:"timeout_setting,omitempty"` // 超时设置信息
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

// UploadResult 文件上传结果
type UploadResult struct {
	Host           string `json:"host"`
	Type           string `json:"type"`
	Status         string `json:"status"`
	LocalFile      string `json:"local_file"`
	RemoteFile     string `json:"remote_file"`
	Size           int64  `json:"size"`
	Duration       string `json:"duration"`
	Error          string `json:"error,omitempty"`
	Timestamp      string `json:"timestamp"`
	SSHUser        string `json:"ssh_user,omitempty"`
	TimeoutSetting string `json:"timeout_setting,omitempty"` // 超时设置信息
}

// DownloadResult 文件下载结果
type DownloadResult struct {
	Host           string `json:"host"`
	Type           string `json:"type"`
	Status         string `json:"status"`
	RemotePath     string `json:"remote_path"`
	LocalPath      string `json:"local_path"`
	Size           int64  `json:"size"`
	MD5            string `json:"md5,omitempty"`
	Duration       string `json:"duration"`
	Error          string `json:"error,omitempty"`
	Timestamp      string `json:"timestamp"`
	SSHUser        string `json:"ssh_user,omitempty"`
	TimeoutSetting string `json:"timeout_setting,omitempty"` // 超时设置信息
}
