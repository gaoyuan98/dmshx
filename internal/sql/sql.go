/*
 * @Author: gaoyuan
 * @Date: 2025-06-17
 * @Description: SQL查询执行模块，支持达梦数据库连接和查询，提供超时控制和结果格式化功能
 */

package sql

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"dmshx/internal/logger"
	"dmshx/internal/output"
	"dmshx/pkg"

	_ "github.com/gaoyuan98/dm"
)

// ExecuteQuery 执行SQL查询
func ExecuteQuery(config *pkg.Config, logWriter io.Writer, cmdLogger *logger.Logger) {
	if config.DBType == "" || config.DBHost == "" || config.DBUser == "" {
		fmt.Fprintf(os.Stderr, "Database type, host and user are required for SQL queries\n")
		return
	}

	startTime := time.Now()

	var db *sql.DB
	var err error
	var connStr string

	// 连接数据库
	switch strings.ToLower(config.DBType) {
	case "dm":
		port := 5236
		if config.DBPort > 0 {
			port = config.DBPort
		}
		// 使用安全的DSN构建函数
		connStr = buildDSN(config.DBUser, config.DBPass, config.DBHost, port)
		db, err = sql.Open("dm", connStr)
	case "oracle":
		// 注意：这里需要导入Oracle驱动，但由于依赖问题，本示例不包含Oracle支持
		errMsg := "Oracle support not implemented in this version"
		result := &pkg.SQLResult{
			Host:   config.DBHost,
			Type:   "sql",
			DB:     config.DBType,
			Status: "error",
			Error:  errMsg,
		}
		cmdLogger.LogSQL(result)
		fmt.Fprintf(os.Stderr, "%s\n", errMsg)
		return
	default:
		errMsg := fmt.Sprintf("Unsupported database type: %s", config.DBType)
		result := &pkg.SQLResult{
			Host:   config.DBHost,
			Type:   "sql",
			DB:     config.DBType,
			Status: "error",
			Error:  errMsg,
		}
		cmdLogger.LogSQL(result)
		fmt.Fprintf(os.Stderr, "%s\n", errMsg)
		return
	}

	if err != nil {
		result := &pkg.SQLResult{
			Host:   config.DBHost,
			Type:   "sql",
			DB:     config.DBType,
			Status: "error",
			Error:  err.Error(),
		}
		cmdLogger.LogSQL(result)
		output.OutputSQLResult(config.DBHost, "error", config.DBType, nil, "0s", err.Error(), config.JSONOutput, logWriter)
		return
	}
	defer db.Close()

	// 设置超时
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(config.Timeout)*time.Second)
	defer cancel()

	// 执行查询
	rows, err := db.QueryContext(ctx, config.SQL)
	if err != nil {
		result := &pkg.SQLResult{
			Host:   config.DBHost,
			Type:   "sql",
			DB:     config.DBType,
			Status: "error",
			Error:  err.Error(),
		}
		cmdLogger.LogSQL(result)
		output.OutputSQLResult(config.DBHost, "error", config.DBType, nil, "0s", err.Error(), config.JSONOutput, logWriter)
		return
	}
	defer rows.Close()

	// 获取列名
	columns, err := rows.Columns()
	if err != nil {
		result := &pkg.SQLResult{
			Host:   config.DBHost,
			Type:   "sql",
			DB:     config.DBType,
			Status: "error",
			Error:  err.Error(),
		}
		cmdLogger.LogSQL(result)
		output.OutputSQLResult(config.DBHost, "error", config.DBType, nil, "0s", err.Error(), config.JSONOutput, logWriter)
		return
	}

	// 准备结果集
	var results []interface{}

	// 遍历结果集
	for rows.Next() {
		// 创建一个切片，用于存储每一行的值
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))

		// 初始化指针
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		// 扫描当前行
		if err := rows.Scan(valuePtrs...); err != nil {
			result := &pkg.SQLResult{
				Host:   config.DBHost,
				Type:   "sql",
				DB:     config.DBType,
				Status: "error",
				Error:  err.Error(),
			}
			cmdLogger.LogSQL(result)
			output.OutputSQLResult(config.DBHost, "error", config.DBType, nil, "0s", err.Error(), config.JSONOutput, logWriter)
			return
		}

		// 创建一个map来存储当前行的数据
		row := make(map[string]interface{})
		for i, col := range columns {
			var v interface{}
			val := values[i]

			// 处理不同类型的值
			switch val.(type) {
			case []byte:
				v = string(val.([]byte))
			default:
				v = val
			}

			row[col] = v
		}

		results = append(results, row)
	}

	// 检查遍历过程中是否有错误
	if err := rows.Err(); err != nil {
		result := &pkg.SQLResult{
			Host:   config.DBHost,
			Type:   "sql",
			DB:     config.DBType,
			Status: "error",
			Error:  err.Error(),
		}
		cmdLogger.LogSQL(result)
		output.OutputSQLResult(config.DBHost, "error", config.DBType, nil, "0s", err.Error(), config.JSONOutput, logWriter)
		return
	}

	duration := time.Since(startTime).String()

	// 记录SQL执行结果
	result := &pkg.SQLResult{
		Host:     config.DBHost,
		Type:     "sql",
		DB:       config.DBType,
		Status:   "success",
		Rows:     results,
		Duration: duration,
	}
	cmdLogger.LogSQL(result)

	output.OutputSQLResult(config.DBHost, "success", config.DBType, results, duration, "", config.JSONOutput, logWriter)
}
