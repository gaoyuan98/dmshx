/*
 * @Author: gaoyuan
 * @Date: 2025-06-17
 * @Description: dmshx主程序入口，负责解析命令行参数并执行相应的SSH命令或SQL查询
 */

package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"

	"dmshx/internal/config"
	"dmshx/internal/logger"
	"dmshx/internal/sql"
	"dmshx/internal/ssh"
	"dmshx/pkg"
)

func main() {
	// 解析命令行参数
	cfg := config.Parse()

	// 设置控制台输出编码为UTF-8
	if runtime.GOOS == "windows" && cfg.EnableUTF8 {
		// 在Windows系统上，设置控制台代码页为65001(UTF-8)
		cmd := exec.Command("chcp", "65001")
		cmd.Stdout = nil
		cmd.Run()
	}

	// 显示版本信息
	if cfg.Version {
		fmt.Printf("dmshx version: %s\nBuild time: %s\nAuthor: %s\nBuild date: %s\n",
			pkg.Version, pkg.BuildTime, pkg.Author, pkg.BuildDate)
		return
	}

	// 创建日志记录器
	cmdLogger := logger.NewLogger(cfg)

	// 设置日志输出
	var logWriter io.Writer = os.Stdout
	if cfg.LogFile != "" {
		logFile, err := os.OpenFile(cfg.LogFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error opening log file: %v\n", err)
		} else {
			defer logFile.Close()
			logWriter = io.MultiWriter(os.Stdout, logFile)
		}
	}

	// 获取主机列表
	hosts := config.GetHosts(cfg)

	// 执行命令、上传文件或SQL
	if cfg.UploadFile != "" && cfg.UploadDir != "" {
		// 上传文件需要主机列表
		if len(hosts) == 0 {
			fmt.Fprintf(os.Stderr, "No hosts specified for file upload. Use -hosts or -host-file\n")
			os.Exit(1)
		}
		// 上传文件
		ssh.UploadFiles(hosts, cfg, logWriter, cmdLogger)
	} else if cfg.RemotePath != "" && cfg.LocalPath != "" {
		// 下载文件需要主机列表
		if len(hosts) == 0 {
			fmt.Fprintf(os.Stderr, "No hosts specified for file download. Use -hosts or -host-file\n")
			os.Exit(1)
		}
		// 下载文件
		ssh.DownloadFiles(hosts, cfg, logWriter, cmdLogger)
	} else if cfg.Cmd != "" {
		// 执行SSH命令需要主机列表
		if len(hosts) == 0 {
			fmt.Fprintf(os.Stderr, "No hosts specified for SSH command. Use -hosts or -host-file\n")
			os.Exit(1)
		}
		// 执行SSH命令
		ssh.ExecuteCommands(hosts, cfg, logWriter, cmdLogger)
	} else if cfg.SQL != "" {
		// 执行SQL查询
		sql.ExecuteQuery(cfg, logWriter, cmdLogger)
	} else {
		fmt.Fprintf(os.Stderr, "No command, upload file, download file or SQL query specified. Use -cmd, -upload-file and -upload-dir, -remote-path and -local-path, or -sql\n")
		os.Exit(1)
	}
}
