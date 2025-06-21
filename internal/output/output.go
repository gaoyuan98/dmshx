/*
 * @Author: gaoyuan
 * @Date: 2025-06-17
 * @Description: 输出格式化模块，负责将SSH命令执行结果和SQL查询结果格式化为JSON或文本格式输出
 */

package output

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"

	"dmshx/pkg"
)

// OutputCmdResult 输出命令执行结果
func OutputCmdResult(host, status, stdout, stderr, cmdType, duration, errMsg string, jsonOutput bool, writer io.Writer) {
	OutputCmdResultWithUsers(host, status, stdout, stderr, cmdType, duration, errMsg, "", "", jsonOutput, writer)
}

// OutputCmdResultWithUsers 输出带有用户信息的命令执行结果
func OutputCmdResultWithUsers(host, status, stdout, stderr, cmdType, duration, errMsg, sshUser, execUser string, jsonOutput bool, writer io.Writer) {
	OutputCmdResultFull(host, status, stdout, stderr, cmdType, duration, errMsg, sshUser, execUser, "", jsonOutput, writer)
}

// OutputCmdResultFull 输出完整的命令执行结果，包括实际执行的命令
func OutputCmdResultFull(host, status, stdout, stderr, cmdType, duration, errMsg, sshUser, execUser, actualCmd string, jsonOutput bool, writer io.Writer) {
	OutputCmdResultComplete(host, status, stdout, stderr, cmdType, duration, errMsg, sshUser, execUser, actualCmd, "", jsonOutput, writer)
}

// OutputCmdResultComplete 输出完整的命令执行结果，包括实际执行的命令和超时设置
func OutputCmdResultComplete(host, status, stdout, stderr, cmdType, duration, errMsg, sshUser, execUser, actualCmd, timeoutSetting string, jsonOutput bool, writer io.Writer) {
	result := pkg.CmdResult{
		Host:           host,
		Type:           cmdType,
		Status:         status,
		Stdout:         stdout,
		Stderr:         stderr,
		Duration:       duration,
		Timestamp:      time.Now().Format("2006-01-02 15:04:05"),
		SSHUser:        sshUser,
		ExecUser:       execUser,
		ActualCmd:      actualCmd,
		TimeoutSetting: timeoutSetting,
	}

	if errMsg != "" {
		result.Error = errMsg
	}

	if jsonOutput {
		jsonData, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error marshaling JSON: %v\n", err)
			return
		}
		fmt.Fprintln(writer, string(jsonData))
	} else {
		fmt.Fprintf(writer, "Host: %s\nType: %s\nStatus: %s\nTimestamp: %s\n",
			result.Host, result.Type, result.Status, result.Timestamp)

		if result.SSHUser != "" {
			fmt.Fprintf(writer, "SSH用户: %s\n", result.SSHUser)
		}

		if result.ExecUser != "" && result.ExecUser != result.SSHUser {
			fmt.Fprintf(writer, "执行用户: %s\n", result.ExecUser)
		}

		if result.ActualCmd != "" {
			fmt.Fprintf(writer, "实际命令: %s\n", result.ActualCmd)
		}

		if result.TimeoutSetting != "" {
			fmt.Fprintf(writer, "超时设置: %s\n", result.TimeoutSetting)
		}

		fmt.Fprintf(writer, "Stdout: %s\nStderr: %s\nDuration: %s\n",
			pkg.CleanAnsiSequences(result.Stdout), pkg.CleanAnsiSequences(result.Stderr), result.Duration)

		if errMsg != "" {
			fmt.Fprintf(writer, "Error: %s\n", errMsg)
		}
	}
}

// OutputSQLResult 输出SQL执行结果
func OutputSQLResult(host, status, dbType string, rows []interface{}, duration, errMsg string, jsonOutput bool, writer io.Writer) {
	OutputSQLResultWithTimeout(host, status, dbType, rows, duration, errMsg, "", jsonOutput, writer)
}

// OutputSQLResultWithTimeout 输出带有超时设置信息的SQL执行结果
func OutputSQLResultWithTimeout(host, status, dbType string, rows []interface{}, duration, errMsg, timeoutSetting string, jsonOutput bool, writer io.Writer) {
	result := pkg.SQLResult{
		Host:           host,
		Type:           "sql",
		DB:             dbType,
		Status:         status,
		Rows:           rows,
		Duration:       duration,
		Timestamp:      time.Now().Format("2006-01-02 15:04:05"),
		TimeoutSetting: timeoutSetting,
	}

	if errMsg != "" {
		result.Error = errMsg
	}

	if jsonOutput {
		jsonData, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error marshaling JSON: %v\n", err)
			return
		}
		fmt.Fprintln(writer, string(jsonData))
	} else {
		fmt.Fprintf(writer, "Host: %s\nType: sql\nDB: %s\nStatus: %s\nTimestamp: %s\n",
			result.Host, result.DB, result.Status, result.Timestamp)

		if result.TimeoutSetting != "" {
			fmt.Fprintf(writer, "超时设置: %s\n", result.TimeoutSetting)
		}

		fmt.Fprintf(writer, "Duration: %s\n", result.Duration)

		if len(rows) > 0 {
			fmt.Fprintf(writer, "Rows: %d\n", len(rows))
			for i, row := range rows {
				fmt.Fprintf(writer, "  Row %d: %v\n", i+1, row)
			}
		}
		if errMsg != "" {
			fmt.Fprintf(writer, "Error: %s\n", errMsg)
		}
	}
}

// OutputUploadResult 输出文件上传结果
func OutputUploadResult(host, status, localFile, remoteFile string, size int64, duration, errMsg, sshUser string, jsonOutput bool, writer io.Writer) {
	OutputUploadResultWithTimeout(host, status, localFile, remoteFile, size, duration, errMsg, sshUser, "", jsonOutput, writer)
}

// OutputUploadResultWithTimeout 输出带有超时设置信息的文件上传结果
func OutputUploadResultWithTimeout(host, status, localFile, remoteFile string, size int64, duration, errMsg, sshUser, timeoutSetting string, jsonOutput bool, writer io.Writer) {
	result := pkg.UploadResult{
		Host:           host,
		Type:           "upload",
		Status:         status,
		LocalFile:      localFile,
		RemoteFile:     remoteFile,
		Size:           size,
		Duration:       duration,
		Timestamp:      time.Now().Format("2006-01-02 15:04:05"),
		SSHUser:        sshUser,
		TimeoutSetting: timeoutSetting,
	}

	if errMsg != "" {
		result.Error = errMsg
	}

	if jsonOutput {
		jsonData, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error marshaling JSON: %v\n", err)
			return
		}
		fmt.Fprintln(writer, string(jsonData))
	} else {
		fmt.Fprintf(writer, "Host: %s\nType: upload\nStatus: %s\nTimestamp: %s\n",
			result.Host, result.Status, result.Timestamp)

		if result.SSHUser != "" {
			fmt.Fprintf(writer, "SSH用户: %s\n", result.SSHUser)
		}

		fmt.Fprintf(writer, "本地文件: %s\n远程文件: %s\n文件大小: %d字节\n",
			result.LocalFile, result.RemoteFile, result.Size)

		if result.TimeoutSetting != "" {
			fmt.Fprintf(writer, "超时设置: %s\n", result.TimeoutSetting)
		}

		fmt.Fprintf(writer, "Duration: %s\n", result.Duration)

		if errMsg != "" {
			fmt.Fprintf(writer, "Error: %s\n", errMsg)
		}
	}
}

// OutputDownloadResult 输出下载文件结果
func OutputDownloadResult(host, status, remotePath, localPath string, size int64, duration, errMsg, sshUser string, jsonOutput bool, writer io.Writer) {
	if jsonOutput {
		// JSON格式输出
		result := map[string]interface{}{
			"host":        host,
			"type":        "download",
			"status":      status,
			"remote_path": remotePath,
			"local_path":  localPath,
			"size":        size,
			"duration":    duration,
			"timestamp":   time.Now().Format("2006-01-02 15:04:05"),
			"ssh_user":    sshUser,
		}

		if errMsg != "" {
			result["error"] = errMsg
		}

		jsonData, _ := json.Marshal(result)
		fmt.Fprintln(writer, string(jsonData))
	} else {
		// 普通文本输出
		timeStr := time.Now().Format("2006-01-02 15:04:05")
		if status == "success" {
			// 计算文件大小单位
			sizeStr := formatFileSize(size)
			fmt.Fprintf(writer, "[%s] %s 成功下载文件 %s 到 %s (大小: %s, 用时: %s, 用户: %s)\n",
				timeStr, host, remotePath, localPath, sizeStr, duration, sshUser)
		} else {
			fmt.Fprintf(writer, "[%s] %s 下载文件失败 %s -> %s (%s, 用户: %s)\n",
				timeStr, host, remotePath, localPath, errMsg, sshUser)
		}
	}
}

// formatFileSize 格式化文件大小
func formatFileSize(size int64) string {
	if size < 1024 {
		return fmt.Sprintf("%dB", size)
	} else if size < 1024*1024 {
		return fmt.Sprintf("%.1fKB", float64(size)/1024)
	} else if size < 1024*1024*1024 {
		return fmt.Sprintf("%.1fMB", float64(size)/(1024*1024))
	} else {
		return fmt.Sprintf("%.1fGB", float64(size)/(1024*1024*1024))
	}
}
