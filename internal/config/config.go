/*
 * @Author: gaoyuan
 * @Date: 2025-06-17
 * @Description: 命令行参数配置模块，负责解析和处理dmshx的命令行参数，包括SSH、数据库和日志相关配置
 */

package config

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"dmshx/pkg"
)

// normalizeArgs 将命令行参数标准化为 -flag=value 格式
func normalizeArgs() {
	if len(os.Args) <= 1 {
		return
	}

	// 创建一个新的参数列表，第一个元素是程序名称
	newArgs := []string{os.Args[0]}

	// 定义布尔类型的参数列表
	boolFlags := map[string]bool{
		"-version":            true,
		"-v":                  true,
		"-json-output":        true,
		"-real-time":          true,
		"-enable-utf8":        true,
		"-enable-command-log": true,
		"-verify-md5":         true,
	}

	for i := 1; i < len(os.Args); i++ {
		arg := os.Args[i]

		// 检查是否是以'-'开头的参数
		if strings.HasPrefix(arg, "-") {
			// 检查参数是否已经是 -flag=value 格式
			if strings.Contains(arg, "=") {
				newArgs = append(newArgs, arg)
				continue
			}

			// 检查是否是布尔类型的参数
			if _, ok := boolFlags[arg]; ok {
				// 如果是最后一个参数或者下一个参数是以'-'开头，则视为布尔标志
				if i+1 >= len(os.Args) || strings.HasPrefix(os.Args[i+1], "-") {
					newArgs = append(newArgs, arg+"=true")
				} else {
					// 否则，将下一个参数视为值
					value := os.Args[i+1]
					newArgs = append(newArgs, arg+"="+value)
					i++ // 跳过下一个参数，因为它已经被处理了
				}
			} else {
				// 非布尔类型的参数，必须有值
				if i+1 < len(os.Args) && !strings.HasPrefix(os.Args[i+1], "-") {
					value := os.Args[i+1]
					newArgs = append(newArgs, arg+"="+value)
					i++ // 跳过下一个参数，因为它已经被处理了
				} else {
					// 如果没有值，则保持原样
					newArgs = append(newArgs, arg)
				}
			}
		} else {
			// 非参数，保持原样
			newArgs = append(newArgs, arg)
		}
	}

	// 替换原始参数
	os.Args = newArgs
}

// Parse 解析命令行参数
func Parse() *pkg.Config {
	// 标准化参数格式
	normalizeArgs()

	config := &pkg.Config{}
	// SSH相关参数
	flag.StringVar(&config.Hosts, "hosts", "", "Comma-separated list of hosts in format ip[:port]")
	flag.StringVar(&config.Hosts, "host", "", "Single host in format ip[:port] (alias for -hosts)")
	flag.StringVar(&config.HostFile, "host-file", "", "Path to file containing hosts, one per line")
	flag.IntVar(&config.Port, "port", 22, "Default SSH port")
	flag.StringVar(&config.User, "user", "", "SSH username")
	flag.StringVar(&config.Key, "key", "", "Path to SSH private key")
	flag.StringVar(&config.Password, "password", "", "SSH password")
	flag.StringVar(&config.Cmd, "cmd", "", "Command to execute on remote hosts")
	flag.IntVar(&config.Timeout, "timeout", 30, "Command or SQL execution timeout in seconds")
	flag.StringVar(&config.ExecUser, "exec-user", "", "User to execute the command as (if different from SSH user)")

	// 文件上传相关参数
	flag.StringVar(&config.UploadFile, "upload-file", "", "Path to local file to upload")
	flag.StringVar(&config.UploadDir, "upload-dir", "", "Remote directory to upload file to")
	flag.IntVar(&config.UploadPermission, "upload-perm", 0644, "Permission for uploaded file (octal, default 0644)")

	// 文件下载相关参数
	flag.StringVar(&config.RemotePath, "remote-path", "", "Remote file or directory to download")
	flag.StringVar(&config.LocalPath, "local-path", "", "Local directory to save downloaded files")
	flag.BoolVar(&config.VerifyMD5, "verify-md5", true, "Verify MD5 checksum of downloaded files")
	flag.Int64Var(&config.BufferSize, "buffer-size", 32, "Buffer size for download in MB (default 32MB)")

	// 数据库相关参数
	flag.StringVar(&config.DBType, "db-type", "", "Database type: dm or oracle")
	flag.StringVar(&config.DBHost, "db-host", "", "Database host")
	flag.IntVar(&config.DBPort, "db-port", 0, "Database port")
	flag.StringVar(&config.DBUser, "db-user", "", "Database username")
	flag.StringVar(&config.DBPass, "db-pass", "", "Database password")
	flag.StringVar(&config.DBName, "db-name", "", "Database name or SID")
	flag.StringVar(&config.SQL, "sql", "", "SQL query to execute")

	// 输出相关参数
	flag.BoolVar(&config.JSONOutput, "json-output", true, "Output results in JSON format")
	flag.StringVar(&config.LogFile, "log-file", "", "Path to log file")
	flag.BoolVar(&config.Version, "version", false, "Show version and build time")
	flag.BoolVar(&config.Version, "v", false, "Show version and build time (alias for -version)")
	flag.BoolVar(&config.RealTimeOutput, "real-time", false, "Enable real-time output for command execution, only works when -json-output=false")
	flag.BoolVar(&config.EnableUTF8, "enable-utf8", true, "Enable UTF-8 encoding for console output")

	// 命令执行日志参数
	flag.BoolVar(&config.EnableCommandLog, "enable-command-log", true, "Enable command execution logging")
	flag.StringVar(&config.CommandLogPath, "command-log-path", "./logs", "Directory for command execution logs")
	flag.IntVar(&config.LogRetention, "log-retention", 7, "Log retention period in days and interval between log cleanup checks")

	// 解析命令行参数
	flag.Parse()

	return config
}

// GetHosts 获取主机列表
func GetHosts(config *pkg.Config) []string {
	var hosts []string

	// 从命令行参数获取主机列表
	if config.Hosts != "" {
		hosts = strings.Split(config.Hosts, ",")
	}

	// 从文件获取主机列表
	if config.HostFile != "" {
		content, err := ioutil.ReadFile(config.HostFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading host file: %v\n", err)
		} else {
			lines := strings.Split(string(content), "\n")
			for _, line := range lines {
				line = strings.TrimSpace(line)
				if line != "" {
					hosts = append(hosts, line)
				}
			}
		}
	}

	return hosts
}
