/*
 * @Author: gaoyuan
 * @Date: 2025-06-17
 * @Description: SSH连接和命令执行模块，支持多主机并发执行SSH命令，包括密码和私钥认证方式
 */

package ssh

import (
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
				Stdout:         pkg.CleanAnsiSequences(stdout.String()),
				Stderr:         pkg.CleanAnsiSequences(stderr.String()),
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
				fmt.Printf("命令执行完成 [%s]: %s (耗时: %s)\n", host, cmdToExecute, duration)
				fmt.Println("----------------------------------------")
			}

			output.OutputCmdResultComplete(host, status, stdout.String(), stderr.String(), "cmd", duration, errMsg, config.User, execUser, cmdToExecute, timeoutSetting, config.JSONOutput, logWriter)
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
