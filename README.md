# DMSHX - DM专用远程指令与数据库执行工具

DMSHX (D(M) + (S)SH + (H)ost e(X)ecutor) 是一个跨平台、零依赖的命令行工具，用于远程主机批量命令执行、配置文件处理、达梦数据库SQL查询，并统一输出为JSON，适合运维集成或Python自动化调用。

## 功能特点

### SSH命令执行
- 支持用户名+密码/私钥认证
- 支持per-host SSH端口配置（例如10.0.0.1:2222）
- 支持传参方式执行任意shell命令（如cat, sed, echo）
- 支持命令超时控制（单位秒）

### SQL查询功能
- 支持数据库类型：
  - 达梦数据库（DM）
  - Oracle（计划支持）
- 支持SQL查询语句，返回结构封装为JSON
- 支持SQL执行超时设置

### JSON输出封装
- 所有命令或SQL执行统一返回结构，便于机器解析与日志归档

### 日志记录功能
- 记录每条命令的接收时间和执行结果
- 支持日志定期自动清理
- 可配置日志保留天数
- 按日期自动组织日志文件，便于查找和管理
- 支持UTF-8编码，确保中文正确显示

## 安装

### 下载预编译版本
从[Releases](https://github.com/yourusername/dmshx/releases)页面下载适合您平台的可执行文件：
- Windows: `dmshx.exe`
- Linux x86_64: `dmshx-linux`
- Linux ARM64: `dmshx-arm`

### 从源码编译
需要Go 1.19或更高版本。

```bash
git clone https://github.com/yourusername/dmshx.git
cd dmshx
go build -o dmshx ./cmd/dmshx
```

在Windows环境下，可以使用提供的构建脚本一键编译多平台版本：

```powershell
# 或者使用批处理文件
.\build_dmshx.bat
```

### 优化可执行文件大小

默认编译的Go程序通常较大，这是因为它们是静态链接的。build_dmshx.ps1脚本已包含以下优化：

1. 使用 `-ldflags "-s -w"` 移除调试信息和符号表
2. 使用 `-trimpath` 移除编译路径信息
3. 使用 UPX 工具进行可执行文件压缩（需单独安装）

如果需要进一步减小文件大小，可以手动安装 UPX 工具：
1. 从 [UPX官网](https://github.com/upx/upx/releases) 下载适合您系统的版本
2. 将 upx 可执行文件放入系统 PATH 目录或与构建脚本相同目录
3. 重新运行构建脚本

通过这些优化，可执行文件通常可以减小 50%-70%。

## 使用方法

> **注意**：命令行参数使用Go标准flag包格式，使用单横线(`-`)作为参数前缀。

### SSH命令执行

```bash
# 使用密码在单台主机上执行命令
dmshx -host "192.168.112.168" -port 22 -user "root" -password "gaoyuan123#" -cmd "cat /opt/dmdata/5236/DMDB/dm.ini"

# 使用私钥在多台主机上执行命令
dmshx -hosts "192.168.1.10,192.168.1.11:2222" -user "root" -key "/path/to/id_rsa" -cmd "cat /etc/hosts" -timeout 30

# 从文件读取主机列表
dmshx -host-file "hosts.txt" -user "admin" -password "password" -cmd "uptime"
```

### SQL查询执行

```bash
# 达梦数据库查询
dmshx -db-type "dm" -db-host "192.168.112.168" -db-port 5236 -db-user "SYSDBA" -db-pass "Dameng123#" -sql "SELECT * FROM V$INSTANCE"

# 设置查询超时
dmshx -db-type "dm" -db-host "192.168.1.20" -db-user "SYSDBA" -db-pass "SYSDBA" -sql "SELECT * FROM LARGE_TABLE" -timeout 60
```

### 输出格式控制

```bash
# 输出为JSON格式（默认）
dmshx -hosts "192.168.1.10" -user "root" -password "password" -cmd "ls -la" -json-output

# 输出到日志文件
dmshx -hosts "192.168.1.10" -user "root" -password "password" -cmd "ls -la" -log-file "output.log"

# 启用命令执行日志记录并设置保留天数
dmshx -hosts "192.168.1.10" -user "root" -password "password" -cmd "ls -la" -enable-command-log -log-retention 30
```

## 命令行参数说明

| 参数名 | 类型 | 默认值 | 说明 |
|--------|------|--------|------|
| -hosts | string | "" | 多主机逗号分隔列表，支持格式 ip[:port]，例如 "192.168.1.10,192.168.1.11:2222" |
| -host | string | "" | 单主机设置，支持格式 ip[:port]，与-hosts功能相同但只接受单个主机 |
| -host-file | string | "" | 主机列表文件路径，文件中每行包含一个主机，格式为 ip[:port] |
| -port | int | 22 | 默认SSH连接端口（全局设置），在hosts未指定端口时使用 |
| -user | string | "" | SSH登录用户名，用于远程主机认证 |
| -key | string | "" | SSH私钥文件路径，优先级高于密码认证 |
| -password | string | "" | SSH登录密码，仅在未提供私钥时使用（不推荐在生产环境直接使用） |
| -cmd | string | "" | 在远程主机执行的Shell命令，例如 "ls -la /opt" 或 "cat /etc/hosts" |
| -timeout | int | 30 | 命令或SQL执行超时时间，单位为秒，超时后会终止执行 |
| -db-type | string | "" | 数据库类型，当前支持 "dm"（达梦数据库），未来计划支持 "oracle" |
| -db-host | string | "" | 数据库服务器主机名或IP地址 |
| -db-port | int | 0 | 数据库服务端口，达梦数据库默认为5236 |
| -db-user | string | "" | 数据库连接用户名 |
| -db-pass | string | "" | 数据库连接密码 |
| -db-name | string | "" | 数据库名称或SID（Oracle） |
| -sql | string | "" | 要执行的SQL查询语句，例如 "SELECT * FROM V$INSTANCE" |
| -json-output | bool | true | 是否以JSON格式输出结果，便于程序解析，默认开启 |
| -log-file | string | "" | 执行结果输出日志文件路径，若指定则同时输出到屏幕和文件 |
| -version | bool | false | 显示程序版本号和构建时间信息 |
| -enable-command-log | bool | true | 是否启用命令执行日志记录功能，默认开启 |
| -command-log-path | string | "./logs" | 命令执行日志存储目录 |
| -log-retention | int | 7 | 日志文件保留天数，超过此天数的日志将被自动清理 |

## 输出格式（JSON）

### 执行命令返回
```json
{
  "host": "192.168.112.168",
  "type": "cmd",
  "status": "success",
  "stdout": "...",
  "stderr": "",
  "duration": "2.45s",
  "timestamp": "2023-06-17 08:45:12"
}
```

### SQL查询返回
```json
{
  "host": "192.168.112.168",
  "type": "sql",
  "db": "dm",
  "status": "success",
  "rows": [
    {"INSTANCE_NAME": "DAMENG", "VERSION": "8.0.0.128", "STATUS": "ACTIVE"}
  ],
  "duration": "0.91s",
  "timestamp": "2023-06-17 08:45:12"
}
```

## 日志记录

启用命令执行日志记录功能后，系统将自动为每条命令创建日志文件，格式如下：

```
日志文件位置: {command-log-path}/{yyyy-MM-dd}/command_{timestamp}.log

日志内容格式:
执行时间: 2023-06-17 08:45:12
命令类型: SSH/SQL
目标主机: 192.168.1.10
执行命令: ls -la
执行状态: 成功/失败
执行耗时: 2.45s
执行结果: ...
```

### 日志清理
系统会在每次启动时以及每24小时自动检查并清理过期日志文件（超过设定的保留天数）。清理时会按照目录日期进行判断，自动删除整个过期日期的目录。

## 高级配置

### 配置示例

以下是一些常用配置场景的示例：

1. 批量执行命令并记录日志：
```bash
dmshx -hosts "server1,server2,server3" -user "admin" -key "/path/to/id_rsa" -cmd "uptime" -enable-command-log -log-retention 30
```

2. 执行达梦数据库查询并输出到文件：
```bash
dmshx -db-type "dm" -db-host "192.168.112.168" -db-port 5236 -db-user "SYSDBA" -db-pass "Dameng123#" -sql "SELECT * FROM V$VERSION" -log-file "dm_version.json"
```

3. 读取主机列表文件并执行命令：
```bash
dmshx -host-file "production_servers.txt" -user "ops" -password "secure_pass" -cmd "systemctl status nginx" -timeout 60
```
