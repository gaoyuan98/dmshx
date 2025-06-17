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
	result := pkg.CmdResult{
		Host:      host,
		Type:      cmdType,
		Status:    status,
		Stdout:    stdout,
		Stderr:    stderr,
		Duration:  duration,
		Timestamp: time.Now().Format("2006-01-02 15:04:05"),
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
		fmt.Fprintf(writer, "Host: %s\nType: %s\nStatus: %s\nTimestamp: %s\nStdout: %s\nStderr: %s\nDuration: %s\n",
			result.Host, result.Type, result.Status, result.Timestamp, result.Stdout, result.Stderr, result.Duration)
		if errMsg != "" {
			fmt.Fprintf(writer, "Error: %s\n", errMsg)
		}
	}
}

// OutputSQLResult 输出SQL执行结果
func OutputSQLResult(host, status, dbType string, rows []interface{}, duration, errMsg string, jsonOutput bool, writer io.Writer) {
	result := pkg.SQLResult{
		Host:      host,
		Type:      "sql",
		DB:        dbType,
		Status:    status,
		Rows:      rows,
		Duration:  duration,
		Timestamp: time.Now().Format("2006-01-02 15:04:05"),
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
		fmt.Fprintf(writer, "Host: %s\nType: sql\nDB: %s\nStatus: %s\nTimestamp: %s\nDuration: %s\n",
			result.Host, result.DB, result.Status, result.Timestamp, result.Duration)
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
