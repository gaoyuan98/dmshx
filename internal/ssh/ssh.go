/*
 * @Author: gaoyuan
 * @Date: 2025-06-17
 * @Description: SSH连接和命令执行模块，支持多主机并发执行SSH命令，包括密码和私钥认证方式
 */

package ssh

import (
	"crypto/md5"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"dmshx/internal/logger"
	"dmshx/internal/output"
	"dmshx/pkg"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

// ExecuteCommands 执行SSH命令
func ExecuteCommands(hosts []string, config *pkg.Config, logWriter io.Writer, cmdLogger *logger.Logger) {
	var wg sync.WaitGroup

	for _, host := range hosts {
		wg.Add(1)
		go func(host string) {
			defer wg.Done()

			// 解析主机和端口
			hostPort := strings.Split(host, ":")
			hostname := hostPort[0]
			port := config.Port
			if len(hostPort) > 1 {
				p, err := strconv.Atoi(hostPort[1])
				if err == nil {
					port = p
				}
			}

			// 创建SSH客户端配置
			clientConfig := &ssh.ClientConfig{
				User:            config.User,
				Auth:            []ssh.AuthMethod{},
				HostKeyCallback: ssh.InsecureIgnoreHostKey(),
				Timeout:         time.Duration(config.Timeout) * time.Second,
			}

			// 添加认证方式
			if config.Key != "" {
				key, err := ioutil.ReadFile(config.Key)
				if err != nil {
					result := &pkg.CmdResult{
						Host:   host,
						Type:   "cmd",
						Status: "error",
						Error:  err.Error(),
					}
					cmdLogger.LogCommand(result)
					output.OutputCmdResult(host, "error", "", "", "cmd", "0s", err.Error(), config.JSONOutput, logWriter)
					return
				}

				signer, err := ssh.ParsePrivateKey(key)
				if err != nil {
					result := &pkg.CmdResult{
						Host:   host,
						Type:   "cmd",
						Status: "error",
						Error:  err.Error(),
					}
					cmdLogger.LogCommand(result)
					output.OutputCmdResult(host, "error", "", "", "cmd", "0s", err.Error(), config.JSONOutput, logWriter)
					return
				}

				clientConfig.Auth = append(clientConfig.Auth, ssh.PublicKeys(signer))
			} else if config.Password != "" {
				clientConfig.Auth = append(clientConfig.Auth, ssh.Password(config.Password))
			} else {
				errMsg := "No authentication method provided. Specify either -key or -password"
				result := &pkg.CmdResult{
					Host:   host,
					Type:   "cmd",
					Status: "error",
					Error:  errMsg,
				}
				cmdLogger.LogCommand(result)
				output.OutputCmdResult(host, "error", "", "", "cmd", "0s", errMsg, config.JSONOutput, logWriter)
				return
			}

			// 连接SSH服务器
			addr := fmt.Sprintf("%s:%d", hostname, port)
			startTime := time.Now()
			client, err := ssh.Dial("tcp", addr, clientConfig)
			if err != nil {
				// 设置超时信息
				var timeoutSetting string
				if config.Timeout > 0 {
					timeoutSetting = fmt.Sprintf("%d秒", config.Timeout)
				} else {
					timeoutSetting = "无限制"
				}

				result := &pkg.CmdResult{
					Host:           host,
					Type:           "cmd",
					Status:         "error",
					Error:          err.Error(),
					SSHUser:        config.User,
					ExecUser:       config.User,
					TimeoutSetting: timeoutSetting,
				}
				cmdLogger.LogCommand(result)
				output.OutputCmdResultComplete(host, "error", "", "", "cmd", "0s", err.Error(), config.User, config.User, "", timeoutSetting, config.JSONOutput, logWriter)
				return
			}
			defer client.Close()

			// 创建会话
			session, err := client.NewSession()
			if err != nil {
				// 设置超时信息
				var timeoutSetting string
				if config.Timeout > 0 {
					timeoutSetting = fmt.Sprintf("%d秒", config.Timeout)
				} else {
					timeoutSetting = "无限制"
				}

				result := &pkg.CmdResult{
					Host:           host,
					Type:           "cmd",
					Status:         "error",
					Error:          err.Error(),
					SSHUser:        config.User,
					ExecUser:       config.User,
					TimeoutSetting: timeoutSetting,
				}
				cmdLogger.LogCommand(result)
				output.OutputCmdResultComplete(host, "error", "", "", "cmd", "0s", err.Error(), config.User, config.User, "", timeoutSetting, config.JSONOutput, logWriter)
				return
			}
			defer session.Close()

			// 获取命令输出
			var stdout, stderr strings.Builder
			session.Stdout = &stdout
			session.Stderr = &stderr

			// 处理命令，如果设置了ExecUser，则切换用户执行
			cmdToExecute := config.Cmd
			execUser := config.User // 默认执行用户与SSH用户相同

			if config.ExecUser != "" && config.ExecUser != config.User {
				// 使用su切换用户执行命令
				cmdToExecute = fmt.Sprintf("su - %s -c '%s'", config.ExecUser, escapeCommand(config.Cmd))
				execUser = config.ExecUser // 更新实际执行用户
			}

			// 设置超时信息
			var timeoutSetting string
			if config.Timeout > 0 {
				timeoutSetting = fmt.Sprintf("%d秒", config.Timeout)
			} else {
				timeoutSetting = "无限制"
			}

			// 创建多写入器，同时写入到strings.Builder和标准输出
			if !config.JSONOutput && config.RealTimeOutput {
				// 实时输出模式：同时写入到变量和屏幕
				fmt.Printf("正在执行命令 [%s]: %s\n", host, cmdToExecute)
				session.Stdout = io.MultiWriter(&stdout, os.Stdout)
				session.Stderr = io.MultiWriter(&stderr, os.Stderr)
			}

			// 执行命令
			err = session.Start(cmdToExecute)
			if err != nil {
				result := &pkg.CmdResult{
					Host:           host,
					Type:           "cmd",
					Status:         "error",
					Error:          err.Error(),
					SSHUser:        config.User,
					ExecUser:       execUser,
					ActualCmd:      cmdToExecute,
					TimeoutSetting: timeoutSetting,
				}
				cmdLogger.LogCommand(result)
				output.OutputCmdResultComplete(host, "error", "", "", "cmd", "0s", err.Error(), config.User, execUser, cmdToExecute, timeoutSetting, config.JSONOutput, logWriter)
				return
			}

			// 设置超时
			done := make(chan error, 1)
			go func() {
				done <- session.Wait()
			}()

			var cmdErr error
			// 只有当超时设置大于0时才设置超时
			if config.Timeout > 0 {
				select {
				case cmdErr = <-done:
					// 命令正常完成
				case <-time.After(time.Duration(config.Timeout) * time.Second):
					session.Signal(ssh.SIGTERM)
					cmdErr = fmt.Errorf("command timed out after %d seconds", config.Timeout)
				}
			} else {
				// 超时为0表示不限制超时时间
				cmdErr = <-done
			}

			duration := time.Since(startTime).String()
			status := "success"
			var errMsg string

			if cmdErr != nil {
				status = "error"
				errMsg = cmdErr.Error()
			}

			// 创建命令执行结果
			result := &pkg.CmdResult{
				Host:           host,
				Type:           "cmd",
				Status:         status,
				Stdout:         pkg.CleanAndUnescapeText(stdout.String()),
				Stderr:         pkg.CleanAndUnescapeText(stderr.String()),
				Duration:       duration,
				Error:          errMsg,
				SSHUser:        config.User,
				ExecUser:       execUser,
				ActualCmd:      cmdToExecute,
				TimeoutSetting: timeoutSetting,
			}

			// 记录命令执行日志
			cmdLogger.LogCommand(result)

			// 如果是实时输出模式，在结束时显示完成信息
			if !config.JSONOutput && config.RealTimeOutput {
				if status == "success" {
					fmt.Printf("命令执行成功 [%s]: %s (耗时: %s)\n", host, cmdToExecute, duration)
				} else {
					fmt.Printf("命令执行失败 [%s]: %s (耗时: %s, 错误: %s)\n", host, cmdToExecute, duration, errMsg)
				}
				fmt.Println("----------------------------------------")
			} else {
				output.OutputCmdResultComplete(host, status, stdout.String(), stderr.String(), "cmd", duration, errMsg, config.User, execUser, cmdToExecute, timeoutSetting, config.JSONOutput, logWriter)
			}
		}(host)
	}

	wg.Wait()
}

// escapeCommand 转义命令中的单引号
func escapeCommand(cmd string) string {
	// 替换单引号为 '\''
	return strings.ReplaceAll(cmd, "'", "'\\''")
}

// UploadFiles 上传文件到远程主机
func UploadFiles(hosts []string, config *pkg.Config, logWriter io.Writer, cmdLogger *logger.Logger) {
	var wg sync.WaitGroup

	// 检查本地文件是否存在
	localFile := config.UploadFile
	fi, err := os.Stat(localFile)
	if err != nil {
		errMsg := fmt.Sprintf("本地文件不存在或无法访问: %v", err)
		fmt.Fprintf(os.Stderr, "%s\n", errMsg)
		return
	}

	// 获取文件大小
	fileSize := fi.Size()

	// 获取文件名
	fileName := filepath.Base(localFile)

	// 确保远程目录有结尾的斜杠
	remoteDir := config.UploadDir
	if !strings.HasSuffix(remoteDir, "/") {
		remoteDir += "/"
	}

	// 计算远程文件路径
	remoteFile := remoteDir + fileName

	for _, host := range hosts {
		wg.Add(1)
		go func(host string) {
			defer wg.Done()

			// 解析主机和端口
			hostPort := strings.Split(host, ":")
			hostname := hostPort[0]
			port := config.Port
			if len(hostPort) > 1 {
				p, err := strconv.Atoi(hostPort[1])
				if err == nil {
					port = p
				}
			}

			// 创建SSH客户端配置
			clientConfig := &ssh.ClientConfig{
				User:            config.User,
				Auth:            []ssh.AuthMethod{},
				HostKeyCallback: ssh.InsecureIgnoreHostKey(),
				Timeout:         time.Duration(config.Timeout) * time.Second,
			}

			// 添加认证方式
			if config.Key != "" {
				key, err := ioutil.ReadFile(config.Key)
				if err != nil {
					result := &pkg.UploadResult{
						Host:       host,
						Type:       "upload",
						Status:     "error",
						LocalFile:  localFile,
						RemoteFile: remoteFile,
						Error:      err.Error(),
						SSHUser:    config.User,
					}
					cmdLogger.LogUpload(result)
					output.OutputUploadResult(host, "error", localFile, remoteFile, 0, "0s", err.Error(), config.User, config.JSONOutput, logWriter)
					return
				}

				signer, err := ssh.ParsePrivateKey(key)
				if err != nil {
					result := &pkg.UploadResult{
						Host:       host,
						Type:       "upload",
						Status:     "error",
						LocalFile:  localFile,
						RemoteFile: remoteFile,
						Error:      err.Error(),
						SSHUser:    config.User,
					}
					cmdLogger.LogUpload(result)
					output.OutputUploadResult(host, "error", localFile, remoteFile, 0, "0s", err.Error(), config.User, config.JSONOutput, logWriter)
					return
				}

				clientConfig.Auth = append(clientConfig.Auth, ssh.PublicKeys(signer))
			} else if config.Password != "" {
				clientConfig.Auth = append(clientConfig.Auth, ssh.Password(config.Password))
			} else {
				errMsg := "No authentication method provided. Specify either -key or -password"
				result := &pkg.UploadResult{
					Host:       host,
					Type:       "upload",
					Status:     "error",
					LocalFile:  localFile,
					RemoteFile: remoteFile,
					Error:      errMsg,
					SSHUser:    config.User,
				}
				cmdLogger.LogUpload(result)
				output.OutputUploadResult(host, "error", localFile, remoteFile, 0, "0s", errMsg, config.User, config.JSONOutput, logWriter)
				return
			}

			// 连接SSH服务器
			addr := fmt.Sprintf("%s:%d", hostname, port)
			startTime := time.Now()
			client, err := ssh.Dial("tcp", addr, clientConfig)
			if err != nil {
				result := &pkg.UploadResult{
					Host:       host,
					Type:       "upload",
					Status:     "error",
					LocalFile:  localFile,
					RemoteFile: remoteFile,
					Error:      err.Error(),
					SSHUser:    config.User,
				}
				cmdLogger.LogUpload(result)
				output.OutputUploadResult(host, "error", localFile, remoteFile, 0, "0s", err.Error(), config.User, config.JSONOutput, logWriter)
				return
			}
			defer client.Close()

			// 创建SFTP客户端
			sftpClient, err := sftp.NewClient(client)
			if err != nil {
				result := &pkg.UploadResult{
					Host:       host,
					Type:       "upload",
					Status:     "error",
					LocalFile:  localFile,
					RemoteFile: remoteFile,
					Error:      err.Error(),
					SSHUser:    config.User,
				}
				cmdLogger.LogUpload(result)
				output.OutputUploadResult(host, "error", localFile, remoteFile, 0, "0s", err.Error(), config.User, config.JSONOutput, logWriter)
				return
			}
			defer sftpClient.Close()

			// 确保远程目录存在
			err = createRemoteDir(sftpClient, remoteDir)
			if err != nil {
				result := &pkg.UploadResult{
					Host:       host,
					Type:       "upload",
					Status:     "error",
					LocalFile:  localFile,
					RemoteFile: remoteFile,
					Error:      fmt.Sprintf("创建远程目录失败: %v", err),
					SSHUser:    config.User,
				}
				cmdLogger.LogUpload(result)
				output.OutputUploadResult(host, "error", localFile, remoteFile, 0, "0s", fmt.Sprintf("创建远程目录失败: %v", err), config.User, config.JSONOutput, logWriter)
				return
			}

			// 打开本地文件
			localFileHandle, err := os.Open(localFile)
			if err != nil {
				result := &pkg.UploadResult{
					Host:       host,
					Type:       "upload",
					Status:     "error",
					LocalFile:  localFile,
					RemoteFile: remoteFile,
					Error:      fmt.Sprintf("打开本地文件失败: %v", err),
					SSHUser:    config.User,
				}
				cmdLogger.LogUpload(result)
				output.OutputUploadResult(host, "error", localFile, remoteFile, 0, "0s", fmt.Sprintf("打开本地文件失败: %v", err), config.User, config.JSONOutput, logWriter)
				return
			}
			defer localFileHandle.Close()

			// 创建远程文件
			remoteFileHandle, err := sftpClient.Create(remoteFile)
			if err != nil {
				result := &pkg.UploadResult{
					Host:       host,
					Type:       "upload",
					Status:     "error",
					LocalFile:  localFile,
					RemoteFile: remoteFile,
					Error:      fmt.Sprintf("创建远程文件失败: %v", err),
					SSHUser:    config.User,
				}
				cmdLogger.LogUpload(result)
				output.OutputUploadResult(host, "error", localFile, remoteFile, 0, "0s", fmt.Sprintf("创建远程文件失败: %v", err), config.User, config.JSONOutput, logWriter)
				return
			}
			defer remoteFileHandle.Close()

			// 设置上传通道和完成通道
			done := make(chan error, 1)
			go func() {
				// 复制文件内容
				_, err := io.Copy(remoteFileHandle, localFileHandle)
				done <- err
			}()

			// 处理上传超时
			var uploadErr error
			if config.Timeout > 0 {
				select {
				case uploadErr = <-done:
					// 上传完成
				case <-time.After(time.Duration(config.Timeout) * time.Second):
					uploadErr = fmt.Errorf("文件上传超时，超过 %d 秒", config.Timeout)
				}
			} else {
				// 超时为0表示不限制超时时间
				uploadErr = <-done
			}

			if uploadErr != nil {
				result := &pkg.UploadResult{
					Host:       host,
					Type:       "upload",
					Status:     "error",
					LocalFile:  localFile,
					RemoteFile: remoteFile,
					Size:       fileSize,
					Error:      fmt.Sprintf("文件上传失败: %v", uploadErr),
					SSHUser:    config.User,
					Duration:   time.Since(startTime).String(),
				}
				cmdLogger.LogUpload(result)
				output.OutputUploadResult(host, "error", localFile, remoteFile, fileSize, time.Since(startTime).String(), fmt.Sprintf("文件上传失败: %v", uploadErr), config.User, config.JSONOutput, logWriter)
				return
			}

			// 如果指定了权限，设置文件权限
			if config.UploadPermission > 0 {
				err = sftpClient.Chmod(remoteFile, os.FileMode(config.UploadPermission))
				if err != nil {
					fmt.Fprintf(os.Stderr, "Warning: 无法设置文件权限 %s: %v\n", remoteFile, err)
				}
			}

			// 记录成功结果
			duration := time.Since(startTime).String()
			result := &pkg.UploadResult{
				Host:       host,
				Type:       "upload",
				Status:     "success",
				LocalFile:  localFile,
				RemoteFile: remoteFile,
				Size:       fileSize,
				Duration:   duration,
				SSHUser:    config.User,
			}

			// 设置超时信息
			var timeoutSetting string
			if config.Timeout > 0 {
				timeoutSetting = fmt.Sprintf("%d秒", config.Timeout)
			} else {
				timeoutSetting = "无限制"
			}
			result.TimeoutSetting = timeoutSetting

			cmdLogger.LogUpload(result)
			output.OutputUploadResultWithTimeout(host, "success", localFile, remoteFile, fileSize, duration, "", config.User, timeoutSetting, config.JSONOutput, logWriter)
		}(host)
	}

	wg.Wait()
}

// createRemoteDir 创建远程目录（包括多级目录）
func createRemoteDir(sftpClient *sftp.Client, dirPath string) error {
	// 去除末尾斜杠
	dirPath = strings.TrimSuffix(dirPath, "/")

	// 检查目录是否已存在
	if _, err := sftpClient.Stat(dirPath); err == nil {
		return nil // 目录已存在
	}

	// 递归创建父目录
	parent := filepath.Dir(dirPath)
	if parent != "." && parent != "/" {
		err := createRemoteDir(sftpClient, parent)
		if err != nil {
			return err
		}
	}

	// 创建当前目录
	return sftpClient.Mkdir(dirPath)
}

// DownloadFiles 从远程主机下载文件或目录到本地
func DownloadFiles(hosts []string, config *pkg.Config, logWriter io.Writer, cmdLogger *logger.Logger) {
	var wg sync.WaitGroup

	for _, host := range hosts {
		wg.Add(1)
		go func(host string) {
			defer wg.Done()

			// 解析主机和端口
			hostPort := strings.Split(host, ":")
			hostname := hostPort[0]
			port := config.Port
			if len(hostPort) > 1 {
				p, err := strconv.Atoi(hostPort[1])
				if err == nil {
					port = p
				}
			}

			// 创建SSH客户端配置
			clientConfig := &ssh.ClientConfig{
				User:            config.User,
				Auth:            []ssh.AuthMethod{},
				HostKeyCallback: ssh.InsecureIgnoreHostKey(),
				Timeout:         time.Duration(config.Timeout) * time.Second,
			}

			// 添加认证方式
			if config.Key != "" {
				key, err := ioutil.ReadFile(config.Key)
				if err != nil {
					result := &pkg.DownloadResult{
						Host:       host,
						Type:       "download",
						Status:     "error",
						RemotePath: config.RemotePath,
						LocalPath:  config.LocalPath,
						Error:      err.Error(),
						SSHUser:    config.User,
						Timestamp:  time.Now().Format("2006-01-02 15:04:05"),
					}
					cmdLogger.LogDownload(result)
					output.OutputDownloadResult(host, "error", config.RemotePath, config.LocalPath, 0, "0s", err.Error(), config.User, config.JSONOutput, logWriter)
					return
				}

				signer, err := ssh.ParsePrivateKey(key)
				if err != nil {
					result := &pkg.DownloadResult{
						Host:       host,
						Type:       "download",
						Status:     "error",
						RemotePath: config.RemotePath,
						LocalPath:  config.LocalPath,
						Error:      err.Error(),
						SSHUser:    config.User,
						Timestamp:  time.Now().Format("2006-01-02 15:04:05"),
					}
					cmdLogger.LogDownload(result)
					output.OutputDownloadResult(host, "error", config.RemotePath, config.LocalPath, 0, "0s", err.Error(), config.User, config.JSONOutput, logWriter)
					return
				}

				clientConfig.Auth = append(clientConfig.Auth, ssh.PublicKeys(signer))
			} else if config.Password != "" {
				clientConfig.Auth = append(clientConfig.Auth, ssh.Password(config.Password))
			} else {
				errMsg := "No authentication method provided. Specify either -key or -password"
				result := &pkg.DownloadResult{
					Host:       host,
					Type:       "download",
					Status:     "error",
					RemotePath: config.RemotePath,
					LocalPath:  config.LocalPath,
					Error:      errMsg,
					SSHUser:    config.User,
					Timestamp:  time.Now().Format("2006-01-02 15:04:05"),
				}
				cmdLogger.LogDownload(result)
				output.OutputDownloadResult(host, "error", config.RemotePath, config.LocalPath, 0, "0s", errMsg, config.User, config.JSONOutput, logWriter)
				return
			}

			// 连接SSH服务器
			addr := fmt.Sprintf("%s:%d", hostname, port)
			startTime := time.Now()
			client, err := ssh.Dial("tcp", addr, clientConfig)
			if err != nil {
				result := &pkg.DownloadResult{
					Host:       host,
					Type:       "download",
					Status:     "error",
					RemotePath: config.RemotePath,
					LocalPath:  config.LocalPath,
					Error:      err.Error(),
					SSHUser:    config.User,
					Timestamp:  time.Now().Format("2006-01-02 15:04:05"),
				}
				cmdLogger.LogDownload(result)
				output.OutputDownloadResult(host, "error", config.RemotePath, config.LocalPath, 0, "0s", err.Error(), config.User, config.JSONOutput, logWriter)
				return
			}
			defer client.Close()

			// 创建SFTP客户端
			sftpClient, err := sftp.NewClient(client)
			if err != nil {
				result := &pkg.DownloadResult{
					Host:       host,
					Type:       "download",
					Status:     "error",
					RemotePath: config.RemotePath,
					LocalPath:  config.LocalPath,
					Error:      err.Error(),
					SSHUser:    config.User,
					Timestamp:  time.Now().Format("2006-01-02 15:04:05"),
				}
				cmdLogger.LogDownload(result)
				output.OutputDownloadResult(host, "error", config.RemotePath, config.LocalPath, 0, "0s", err.Error(), config.User, config.JSONOutput, logWriter)
				return
			}
			defer sftpClient.Close()

			// 检查远程路径是文件还是目录
			remoteFileInfo, err := sftpClient.Stat(config.RemotePath)
			if err != nil {
				result := &pkg.DownloadResult{
					Host:       host,
					Type:       "download",
					Status:     "error",
					RemotePath: config.RemotePath,
					LocalPath:  config.LocalPath,
					Error:      fmt.Sprintf("远程路径不存在或无法访问: %v", err),
					SSHUser:    config.User,
					Timestamp:  time.Now().Format("2006-01-02 15:04:05"),
				}
				cmdLogger.LogDownload(result)
				output.OutputDownloadResult(host, "error", config.RemotePath, config.LocalPath, 0, "0s", fmt.Sprintf("远程路径不存在或无法访问: %v", err), config.User, config.JSONOutput, logWriter)
				return
			}

			// 确保本地目录存在
			err = os.MkdirAll(config.LocalPath, 0755)
			if err != nil {
				result := &pkg.DownloadResult{
					Host:       host,
					Type:       "download",
					Status:     "error",
					RemotePath: config.RemotePath,
					LocalPath:  config.LocalPath,
					Error:      fmt.Sprintf("创建本地目录失败: %v", err),
					SSHUser:    config.User,
					Timestamp:  time.Now().Format("2006-01-02 15:04:05"),
				}
				cmdLogger.LogDownload(result)
				output.OutputDownloadResult(host, "error", config.RemotePath, config.LocalPath, 0, "0s", fmt.Sprintf("创建本地目录失败: %v", err), config.User, config.JSONOutput, logWriter)
				return
			}

			if remoteFileInfo.IsDir() {
				// 下载目录
				err = downloadDirectory(sftpClient, config.RemotePath, config.LocalPath, host, config, logWriter, cmdLogger)
				if err != nil {
					result := &pkg.DownloadResult{
						Host:       host,
						Type:       "download",
						Status:     "error",
						RemotePath: config.RemotePath,
						LocalPath:  config.LocalPath,
						Error:      fmt.Sprintf("下载目录失败: %v", err),
						SSHUser:    config.User,
						Duration:   time.Since(startTime).String(),
						Timestamp:  time.Now().Format("2006-01-02 15:04:05"),
					}
					cmdLogger.LogDownload(result)
					output.OutputDownloadResult(host, "error", config.RemotePath, config.LocalPath, 0, time.Since(startTime).String(), fmt.Sprintf("下载目录失败: %v", err), config.User, config.JSONOutput, logWriter)
					return
				}
			} else {
				// 下载单个文件
				localFilePath := filepath.Join(config.LocalPath, filepath.Base(config.RemotePath))
				fileSize, md5sum, err := downloadFile(sftpClient, config.RemotePath, localFilePath, host, config, logWriter)
				if err != nil {
					result := &pkg.DownloadResult{
						Host:       host,
						Type:       "download",
						Status:     "error",
						RemotePath: config.RemotePath,
						LocalPath:  localFilePath,
						Size:       fileSize,
						Error:      fmt.Sprintf("下载文件失败: %v", err),
						SSHUser:    config.User,
						Duration:   time.Since(startTime).String(),
						Timestamp:  time.Now().Format("2006-01-02 15:04:05"),
					}
					cmdLogger.LogDownload(result)
					output.OutputDownloadResult(host, "error", config.RemotePath, localFilePath, fileSize, time.Since(startTime).String(), fmt.Sprintf("下载文件失败: %v", err), config.User, config.JSONOutput, logWriter)
					return
				}

				// 记录成功结果
				duration := time.Since(startTime).String()
				result := &pkg.DownloadResult{
					Host:       host,
					Type:       "download",
					Status:     "success",
					RemotePath: config.RemotePath,
					LocalPath:  localFilePath,
					Size:       fileSize,
					MD5:        md5sum,
					Duration:   duration,
					SSHUser:    config.User,
					Timestamp:  time.Now().Format("2006-01-02 15:04:05"),
				}
				cmdLogger.LogDownload(result)
				output.OutputDownloadResult(host, "success", config.RemotePath, localFilePath, fileSize, duration, "", config.User, config.JSONOutput, logWriter)
			}
		}(host)
	}

	wg.Wait()
}

// downloadFile 下载单个文件并显示进度
func downloadFile(sftpClient *sftp.Client, remotePath, localPath, host string, config *pkg.Config, logWriter io.Writer) (int64, string, error) {
	// 打开远程文件
	remoteFile, err := sftpClient.Open(remotePath)
	if err != nil {
		return 0, "", fmt.Errorf("打开远程文件失败: %v", err)
	}
	defer remoteFile.Close()

	// 获取文件信息
	fileInfo, err := remoteFile.Stat()
	if err != nil {
		return 0, "", fmt.Errorf("获取远程文件信息失败: %v", err)
	}
	fileSize := fileInfo.Size()

	// 创建本地文件
	localFile, err := os.Create(localPath)
	if err != nil {
		return 0, "", fmt.Errorf("创建本地文件失败: %v", err)
	}

	// 在发生错误时删除本地文件
	var downloadError error
	defer func() {
		localFile.Close()
		if downloadError != nil {
			// 发生错误时删除未完成的文件
			if !config.JSONOutput {
				fmt.Printf("删除不完整的下载文件: %s\n", localPath)
			}
			os.Remove(localPath)
		}
	}()

	// 创建进度条
	bar := newProgressBar(fileSize, remotePath)

	// 创建MD5哈希计算器
	hash := md5.New()

	// 创建多写入器，同时写入到文件和哈希计算器
	multiWriter := io.MultiWriter(localFile, hash)

	// 设置缓冲区大小
	bufSize := config.BufferSize * 1024 * 1024 // 将MB转换为字节
	if bufSize <= 0 {
		bufSize = 32 * 1024 * 1024 // 默认32MB
	}
	buf := make([]byte, bufSize)

	// 初始化已下载字节数
	var downloaded int64 = 0
	lastProgressUpdate := time.Now()

	// 设置下载通道和完成通道
	done := make(chan error, 1)

	// 启动下载协程
	go func() {
		// 读取文件并计算MD5
		for {
			nr, er := remoteFile.Read(buf)
			if nr > 0 {
				nw, ew := multiWriter.Write(buf[0:nr])
				if nw > 0 {
					downloaded += int64(nw)

					// 更新进度条，限制更新频率
					if !config.JSONOutput && time.Since(lastProgressUpdate) > 100*time.Millisecond {
						bar.updateProgress(downloaded)
						lastProgressUpdate = time.Now()
					}
				}
				if ew != nil {
					done <- ew
					return
				}
				if nr != nw {
					done <- io.ErrShortWrite
					return
				}
			}
			if er != nil {
				if er != io.EOF {
					done <- er
				} else {
					done <- nil // 成功完成
				}
				return
			}
		}
	}()

	// 处理下载超时
	if config.Timeout > 0 {
		select {
		case downloadError = <-done:
			// 下载完成或发生错误
		case <-time.After(time.Duration(config.Timeout) * time.Second):
			downloadError = fmt.Errorf("文件下载超时，超过 %d 秒", config.Timeout)

			// 尝试手动关闭远程文件，减少资源泄漏
			remoteFile.Close()

			if !config.JSONOutput {
				fmt.Printf("\n下载超时，已中断下载: %s\n", remotePath)
			}
		}
	} else {
		// 超时为0表示不限制超时时间
		downloadError = <-done
	}

	// 如果发生错误，返回
	if downloadError != nil {
		return downloaded, "", downloadError
	}

	// 完成进度条
	if !config.JSONOutput {
		bar.finish()
	}

	// 计算MD5校验和
	md5sum := fmt.Sprintf("%x", hash.Sum(nil))

	return fileSize, md5sum, nil
}

// downloadDirectory 递归下载目录
func downloadDirectory(sftpClient *sftp.Client, remotePath, localPath, host string, config *pkg.Config, logWriter io.Writer, cmdLogger *logger.Logger) error {
	// 创建本地目录
	localDirPath := filepath.Join(localPath, filepath.Base(remotePath))
	err := os.MkdirAll(localDirPath, 0755)
	if err != nil {
		return fmt.Errorf("创建本地目录失败: %v", err)
	}

	// 读取远程目录内容
	remoteFiles, err := sftpClient.ReadDir(remotePath)
	if err != nil {
		return fmt.Errorf("读取远程目录失败: %v", err)
	}

	// 遍历目录内容
	for _, remoteFile := range remoteFiles {
		remoteFilePath := filepath.Join(remotePath, remoteFile.Name())
		localFilePath := filepath.Join(localDirPath, remoteFile.Name())

		if remoteFile.IsDir() {
			// 递归下载子目录
			err = downloadDirectory(sftpClient, remoteFilePath, localDirPath, host, config, logWriter, cmdLogger)
			if err != nil {
				return err
			}
		} else {
			// 下载文件
			fileSize, md5sum, err := downloadFile(sftpClient, remoteFilePath, localFilePath, host, config, logWriter)
			if err != nil {
				return err
			}

			// 记录文件下载结果
			result := &pkg.DownloadResult{
				Host:       host,
				Type:       "download",
				Status:     "success",
				RemotePath: remoteFilePath,
				LocalPath:  localFilePath,
				Size:       fileSize,
				MD5:        md5sum,
				Duration:   "0s", // 这里不记录单个文件的下载时间
				SSHUser:    config.User,
				Timestamp:  time.Now().Format("2006-01-02 15:04:05"),
			}
			cmdLogger.LogDownload(result)

			// 非JSON模式下不在这里输出结果，避免大量输出
			if config.JSONOutput {
				output.OutputDownloadResult(host, "success", remoteFilePath, localFilePath, fileSize, "0s", "", config.User, config.JSONOutput, logWriter)
			}
		}
	}

	return nil
}

// progressBar 简单的进度条结构
type progressBar struct {
	total      int64
	current    int64
	startTime  time.Time
	lastOutput time.Time
	fileName   string
}

// newProgressBar 创建新的进度条
func newProgressBar(total int64, fileName string) *progressBar {
	return &progressBar{
		total:      total,
		current:    0,
		startTime:  time.Now(),
		lastOutput: time.Now(),
		fileName:   filepath.Base(fileName),
	}
}

// updateProgress 更新进度条
func (p *progressBar) updateProgress(current int64) {
	p.current = current

	// 限制更新频率
	if time.Since(p.lastOutput) < 100*time.Millisecond {
		return
	}
	p.lastOutput = time.Now()

	percent := float64(p.current) * 100 / float64(p.total)

	// 计算速度
	elapsed := time.Since(p.startTime).Seconds()
	speed := float64(p.current) / elapsed / 1024 // KB/s

	// 估计剩余时间
	var eta string
	if speed > 0 {
		etaSeconds := float64(p.total-p.current) / (speed * 1024)
		if etaSeconds < 60 {
			eta = fmt.Sprintf("%.1f秒", etaSeconds)
		} else if etaSeconds < 3600 {
			eta = fmt.Sprintf("%.1f分钟", etaSeconds/60)
		} else {
			eta = fmt.Sprintf("%.1f小时", etaSeconds/3600)
		}
	} else {
		eta = "计算中..."
	}

	// 绘制进度条
	width := 50
	completed := int(float64(width) * float64(p.current) / float64(p.total))

	fmt.Printf("\r%s [", p.fileName)
	for i := 0; i < width; i++ {
		if i < completed {
			fmt.Print("=")
		} else if i == completed {
			fmt.Print(">")
		} else {
			fmt.Print(" ")
		}
	}

	// 格式化大小
	var totalSizeStr, currentSizeStr string
	if p.total < 1024 {
		totalSizeStr = fmt.Sprintf("%dB", p.total)
	} else if p.total < 1024*1024 {
		totalSizeStr = fmt.Sprintf("%.1fKB", float64(p.total)/1024)
	} else if p.total < 1024*1024*1024 {
		totalSizeStr = fmt.Sprintf("%.1fMB", float64(p.total)/(1024*1024))
	} else {
		totalSizeStr = fmt.Sprintf("%.1fGB", float64(p.total)/(1024*1024*1024))
	}

	if p.current < 1024 {
		currentSizeStr = fmt.Sprintf("%dB", p.current)
	} else if p.current < 1024*1024 {
		currentSizeStr = fmt.Sprintf("%.1fKB", float64(p.current)/1024)
	} else if p.current < 1024*1024*1024 {
		currentSizeStr = fmt.Sprintf("%.1fMB", float64(p.current)/(1024*1024))
	} else {
		currentSizeStr = fmt.Sprintf("%.1fGB", float64(p.current)/(1024*1024*1024))
	}

	fmt.Printf("] %.1f%% %s/%s %.1fKB/s ETA:%s", percent, currentSizeStr, totalSizeStr, speed, eta)
}

// finish 完成进度条
func (p *progressBar) finish() {
	// 更新最终进度
	p.updateProgress(p.total)
	fmt.Println()
}
