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
	"strconv"
	"strings"
	"sync"
	"time"

	"dmshx/internal/logger"
	"dmshx/internal/output"
	"dmshx/pkg"

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
			defer client.Close()

			// 创建会话
			session, err := client.NewSession()
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
			defer session.Close()

			// 获取命令输出
			var stdout, stderr strings.Builder
			session.Stdout = &stdout
			session.Stderr = &stderr

			// 执行命令
			err = session.Start(config.Cmd)
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

			// 设置超时
			done := make(chan error, 1)
			go func() {
				done <- session.Wait()
			}()

			var cmdErr error
			select {
			case cmdErr = <-done:
				// 命令正常完成
			case <-time.After(time.Duration(config.Timeout) * time.Second):
				session.Signal(ssh.SIGTERM)
				cmdErr = fmt.Errorf("command timed out after %d seconds", config.Timeout)
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
				Host:     host,
				Type:     "cmd",
				Status:   status,
				Stdout:   stdout.String(),
				Stderr:   stderr.String(),
				Duration: duration,
				Error:    errMsg,
			}

			// 记录命令执行日志
			cmdLogger.LogCommand(result)

			output.OutputCmdResult(host, status, stdout.String(), stderr.String(), "cmd", duration, errMsg, config.JSONOutput, logWriter)
		}(host)
	}

	wg.Wait()
}
