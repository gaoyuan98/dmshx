# DMSHX - DM专用远程指令与数据库执行工具

DMSHX (D(M) + (S)SH + (H)ost e(X)ecutor) 是一个跨平台、零依赖的命令行工具，用于远程主机批量命令执行、配置文件处理、达梦数据库SQL查询，并统一输出为JSON，适合运维集成或Python自动化调用。

## 功能特点

### SSH命令执行
- 支持用户名+密码/私钥认证
- 支持per-host SSH端口配置（例如10.0.0.1:2222）
- 支持传参方式执行任意shell命令（如cat, sed, echo）
- 支持命令超时控制（单位秒）

### 文件上传功能
- 支持SFTP文件上传到远程主机
- 支持自动创建远程目录结构
- 支持设置上传文件的权限
- 支持上传超时控制
- 支持多主机并行上传

### 文件下载功能
- 支持从远程主机下载单个文件或整个目录
- 支持MD5校验确保文件完整性
- 提供实时进度显示，包括下载速度、剩余时间等
- 支持多主机并行下载
- 支持下载超时控制

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
- Windows x86_64: `dmshx-windows-x86_64.exe`
- Linux x86_64: `dmshx-linux-x86_64`
- Linux ARM64: `dmshx-linux-arm64`

### 从源码编译
需要Go 1.19或更高版本。

```bash
git clone https://github.com/gaoyuan98/dmshx/dmshx.git
cd dmshx
go build -o dmshx ./cmd/dmshx
```

在Windows环境下，可以使用提供的构建脚本一键编译多平台版本：

```powershell
# 或者使用批处理文件
.\build_dmshx.bat
```

### 优化可执行文件大小

默认编译的Go程序通常较大，这是因为它们是静态链接的。build_dmshx.bat脚本已包含以下优化：

1. 使用 `-ldflags "-s -w"` 移除调试信息和符号表
2. 使用 `-trimpath` 移除编译路径信息
3. 支持UPX工具进行可执行文件压缩（需单独安装）

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

# 以root用户连接，但以dmdba用户执行命令
dmshx -hosts "192.168.1.10,192.168.1.11" -user "root" -password "rootpassword" -cmd "cat /opt/dmdata/5236/DMDB/dm.ini" -exec-user "dmdba"
```

### SQL查询执行

```bash
# 达梦数据库查询
dmshx -db-type "dm" -db-host "192.168.112.168" -db-port 5236 -db-user "SYSDBA" -db-pass "Dameng123#" -sql "SELECT * FROM V$INSTANCE" -timeout 60

# 设置查询超时
dmshx -db-type "dm" -db-host "192.168.1.20" -db-user "SYSDBA" -db-pass "SYSDBA" -sql "SELECT * FROM LARGE_TABLE" -timeout 60

# 执行不限制超时的查询（适用于大型报表查询）
dmshx -db-type "dm" -db-host "192.168.112.168" -db-port 5236 -db-user "SYSDBA" -db-pass "Dameng123#" -sql "SELECT * FROM LARGE_TABLE JOIN ANOTHER_TABLE" -timeout 0
```

### 输出格式控制

```bash
# 输出为JSON格式（默认）
dmshx -hosts "192.168.1.10" -user "root" -password "password" -cmd "ls -la" -json-output

# 输出到日志文件
dmshx -hosts "192.168.1.10" -user "root" -password "password" -cmd "ls -la" -log-file "output.log"

# 关闭UTF-8编码（适用于特殊终端环境）
dmshx -hosts "192.168.1.10" -user "root" -password "password" -cmd "ls -la" -enable-utf8=false

# 启用命令执行日志记录并设置保留天数
dmshx -hosts "192.168.1.10" -user "root" -password "password" -cmd "ls -la" -enable-command-log -log-retention 30

# 显示版本信息（使用短参数）
dmshx -v

# 显示版本信息（使用完整参数）
dmshx -version
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
| -exec-user | string | "" | 执行命令的用户，如果设置且与SSH登录用户不同，将使用su切换到该用户执行命令 |
| -upload-file | string | "" | 要上传到远程主机的本地文件路径 |
| -upload-dir | string | "" | 远程主机上的目标目录，文件将上传到此目录下 |
| -upload-perm | int | 0644 | 上传文件的权限设置（八进制），默认为0644 |
| -remote-path | string | "" | 要从远程主机下载的文件或目录路径 |
| -local-path | string | "" | 下载文件保存到本地的目录路径 |
| -verify-md5 | bool | true | 是否验证下载文件的MD5校验和，确保文件完整性 |
| -buffer-size | int64 | 32 | 下载文件时使用的缓冲区大小，单位为MB |
| -db-type | string | "" | 数据库类型，当前支持 "dm"（达梦数据库），未来计划支持 "oracle" |
| -db-host | string | "" | 数据库服务器主机名或IP地址 |
| -db-port | int | 0 | 数据库服务端口，达梦数据库默认为5236 |
| -db-user | string | "" | 数据库连接用户名 |
| -db-pass | string | "" | 数据库连接密码 |
| -db-name | string | "" | 数据库名称或SID（Oracle） |
| -sql | string | "" | 要执行的SQL查询语句，例如 "SELECT * FROM V$INSTANCE" |
| -json-output | bool | true | 是否以JSON格式输出结果，便于程序解析，默认开启 |
| -log-file | string | "" | 执行结果输出日志文件路径，若指定则同时输出到屏幕和文件 |
| -version, -v | bool | false | 显示程序版本号、构建时间、作者和构建日期信息 |
| -real-time | bool | false | 启用命令执行实时输出功能，只在非JSON输出模式下有效（-json-output=false） |
| -enable-utf8 | bool | true | 启用UTF-8编码输出，在Windows环境下自动设置控制台代码页为65001(UTF-8)，确保中文正确显示 |
| -enable-command-log | bool | true | 是否启用命令执行日志记录功能，默认开启 |
| -command-log-path | string | "./logs" | 命令执行日志存储目录 |
| -log-retention | int | 7 | 日志文件保留天数，同时也是清理检查的间隔天数 |

## 输出格式详解

dmshx支持两种输出格式：JSON格式（默认）和文本格式。所有输出都包含统一的字段结构，便于程序解析和日志归档。

### JSON格式输出（默认）

#### SSH命令执行结果

**成功执行示例：**
```json
{
  "host": "192.168.112.168",
  "type": "cmd",
  "status": "success",
  "stdout": "total 8\ndrwxr-xr-x 2 root root 4096 Jun 17 08:45 .\ndrwxr-xr-x 3 root root 4096 Jun 17 08:44 ..\n-rw-r--r-- 1 root root  123 Jun 17 08:45 test.txt",
  "stderr": "",
  "duration": "2.45s",
  "timestamp": "2025-06-17 08:45:12",
  "error": "",
  "ssh_user": "root",
  "exec_user": "dmdba",
  "actual_cmd": "su - dmdba -c 'ls -la'",
  "timeout_setting": "30秒"
}
```

**执行失败示例：**
```json
{
  "host": "192.168.112.168",
  "type": "cmd",
  "status": "error",
  "stdout": "",
  "stderr": "ls: cannot access '/nonexistent': No such file or directory",
  "duration": "0.12s",
  "timestamp": "2025-06-17 08:45:12",
  "error": "exit status 2"
}
```

**连接失败示例：**
```json
{
  "host": "192.168.112.168",
  "type": "cmd",
  "status": "error",
  "stdout": "",
  "stderr": "",
  "duration": "0s",
  "timestamp": "2025-06-17 08:45:12",
  "error": "dial tcp 192.168.112.168:22: connect: connection refused"
}
```

#### SQL查询执行结果

**成功查询示例：**
```json
{
  "host": "192.168.112.168",
  "type": "sql",
  "db": "dm",
  "status": "success",
  "rows": [
    {
      "INSTANCE_NAME": "DAMENG",
      "VERSION": "8.0.0.128",
      "STATUS": "ACTIVE",
      "STARTUP_TIME": "2025-06-17 08:00:00"
    },
    {
      "INSTANCE_NAME": "DAMENG2",
      "VERSION": "8.0.0.128",
      "STATUS": "ACTIVE",
      "STARTUP_TIME": "2025-06-17 08:30:00"
    }
  ],
  "duration": "0.91s",
  "timestamp": "2025-06-17 08:45:12",
  "error": "",
  "timeout_setting": "30秒"
}
```

**查询失败示例：**
```json
{
  "host": "192.168.112.168",
  "type": "sql",
  "db": "dm",
  "status": "error",
  "rows": [],
  "duration": "0.05s",
  "timestamp": "2025-06-17 08:45:12",
  "error": "table or view does not exist: NONEXISTENT_TABLE"
}
```

**连接失败示例：**
```json
{
  "host": "192.168.112.168",
  "type": "sql",
  "db": "dm",
  "status": "error",
  "rows": [],
  "duration": "0s",
  "timestamp": "2025-06-17 08:45:12",
  "error": "dial tcp 192.168.112.168:5236: connect: connection refused"
}
```

#### 文件上传结果

**上传成功示例：**
```json
{
  "host": "192.168.1.10",
  "type": "upload",
  "status": "success",
  "local_file": "/path/to/localfile.txt",
  "remote_file": "/opt/destination/localfile.txt",
  "size": 12345,
  "duration": "1.23s",
  "timestamp": "2025-06-17 08:45:12",
  "ssh_user": "root",
  "timeout_setting": "30秒"
}
```

**上传失败示例：**
```json
{
  "host": "192.168.1.10",
  "type": "upload",
  "status": "error",
  "local_file": "/path/to/localfile.txt",
  "remote_file": "/opt/destination/localfile.txt",
  "size": 0,
  "duration": "0.05s",
  "timestamp": "2025-06-17 08:45:12",
  "ssh_user": "root",
  "error": "创建远程目录失败: permission denied"
}
```

#### 文件下载结果

**下载成功示例：**
```json
{
  "host": "192.168.1.10",
  "type": "download",
  "status": "success",
  "remote_path": "/opt/source/file.txt",
  "local_path": "/downloads/file.txt",
  "size": 12345,
  "md5": "a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6",
  "duration": "1.23s",
  "timestamp": "2025-06-17 08:45:12",
  "ssh_user": "root",
  "timeout_setting": "30秒"
}
```

**下载失败示例：**
```json
{
  "host": "192.168.1.10",
  "type": "download",
  "status": "error",
  "remote_path": "/opt/source/file.txt",
  "local_path": "/downloads/file.txt",
  "size": 0,
  "duration": "0.05s",
  "timestamp": "2025-06-17 08:45:12",
  "ssh_user": "root",
  "error": "远程文件不存在或无法访问: no such file or directory"
}
```

### 文本格式输出

当设置 `-json-output=false` 时，输出为易读的文本格式：

#### SSH命令执行结果（文本格式）

**成功执行：**
```
Host: 192.168.112.168
Type: cmd
Status: success
Timestamp: 2025-06-17 08:45:12
Stdout: total 8
drwxr-xr-x 2 root root 4096 Jun 17 08:45 .
drwxr-xr-x 3 root root 4096 Jun 17 08:44 ..
-rw-r--r-- 1 root root  123 Jun 17 08:45 test.txt
Stderr: 
Duration: 2.45s
```

**执行失败：**
```
Host: 192.168.112.168
Type: cmd
Status: error
Timestamp: 2025-06-17 08:45:12
Stdout: 
Stderr: ls: cannot access '/nonexistent': No such file or directory
Duration: 0.12s
Error: exit status 2
```

#### SQL查询结果（文本格式）

**成功查询：**
```
Host: 192.168.112.168
Type: sql
DB: dm
Status: success
Timestamp: 2025-06-17 08:45:12
Duration: 0.91s
Rows: 2
  Row 1: map[INSTANCE_NAME:DAMENG VERSION:8.0.0.128 STATUS:ACTIVE]
  Row 2: map[INSTANCE_NAME:DAMENG2 VERSION:8.0.0.128 STATUS:ACTIVE]
```

**查询失败：**
```
Host: 192.168.112.168
Type: sql
DB: dm
Status: error
Timestamp: 2025-06-17 08:45:12
Duration: 0.05s
Rows: 0
Error: table or view does not exist: NONEXISTENT_TABLE
```

### 输出字段说明

#### 通用字段

| 字段名 | 类型 | 说明 |
|--------|------|------|
| `host` | string | 目标主机IP地址或主机名 |
| `type` | string | 执行类型，SSH命令为"cmd"，SQL查询为"sql" |
| `status` | string | 执行状态，"success"表示成功，"error"表示失败 |
| `duration` | string | 执行耗时，格式为"Xs"（如"2.45s"） |
| `timestamp` | string | 执行完成时间戳，格式为"YYYY-MM-DD HH:MM:SS" |
| `ssh_user` | string | SSH连接使用的用户名 |
| `exec_user` | string | 实际执行命令的用户名，当使用-exec-user参数时会与ssh_user不同 |
| `actual_cmd` | string | 实际执行的命令字符串，当使用-exec-user参数时会与原始命令不同 |
| `timeout_setting` | string | 执行命令的超时设置，如"30秒"或"无限制" |

#### SSH命令执行特有字段

| 字段名 | 类型 | 说明 |
|--------|------|------|
| `stdout` | string | 命令的标准输出内容 |
| `stderr` | string | 命令的标准错误输出内容 |
| `error` | string | 执行过程中的错误信息（仅在失败时存在） |

#### SQL查询特有字段

| 字段名 | 类型 | 说明 |
|--------|------|------|
| `db` | string | 数据库类型，如"dm"、"oracle" |
| `rows` | array | 查询结果行数组，每行为一个对象，键为列名，值为列值 |
| `error` | string | 查询过程中的错误信息（仅在失败时存在） |
| `timeout_setting` | string | 执行SQL查询的超时设置，如"30秒"或"无限制" |

### 多主机并发执行

当指定多个主机时，每个主机的执行结果会分别输出：

```bash
# 执行命令
dmshx -hosts "192.168.1.10,192.168.1.11" -user "root" -password "password" -cmd "uptime"
```

**输出示例：**
```json
{
  "host": "192.168.1.10",
  "type": "cmd",
  "status": "success",
  "stdout": " 08:45:12 up 5 days, 12:30,  1 user,  load average: 0.52, 0.48, 0.45",
  "stderr": "",
  "duration": "1.23s",
  "timestamp": "2025-06-17 08:45:12"
}
{
  "host": "192.168.1.11",
  "type": "cmd",
  "status": "success",
  "stdout": " 08:45:13 up 3 days, 8:15,  2 users,  load average: 0.78, 0.65, 0.52",
  "stderr": "",
  "duration": "1.45s",
  "timestamp": "2025-06-17 08:45:13"
}
```

### 错误处理

dmshx对不同类型的错误提供详细的错误信息：

1. **连接错误**：网络连接失败、认证失败等
2. **执行错误**：命令执行失败、SQL语法错误等
3. **超时错误**：执行时间超过设定的超时时间
4. **权限错误**：SSH认证失败、数据库权限不足等

所有错误都会在`error`字段中提供具体的错误描述，便于问题诊断和调试。

## 日志记录

启用命令执行日志记录功能后，系统将自动为每条命令创建日志文件，格式如下：

```
日志文件位置: {command-log-path}/{yyyy-MM-dd}/command_{timestamp}.log

日志内容格式:
执行时间: 2023-06-17 08:45:12
命令类型: SSH
目标主机: 192.168.1.10
SSH用户: root
执行用户: dmdba  (仅当与SSH用户不同时显示)
原始命令: ls -la
实际命令: su - dmdba -c 'ls -la'  (仅当与原始命令不同时显示)
超时设置: 30秒
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

4. 以root用户连接但以dmdba用户执行命令：
```bash
dmshx -hosts "192.168.1.10,192.168.1.11" -user "root" -password "rootpassword" -cmd "cat /opt/dmdata/5236/DMDB/dm.ini" -exec-user "dmdba"

dmshx -hosts "192.168.1.10,192.168.1.11" -user "root" -password "rootpassword" -cmd "/opt/dmdbms/bin/DmServiceDM01 restart" -exec-user "dmdba"
 
```

这种方式实际执行的命令是：`su - dmdba -c 'ps -ef | grep dms'`，适用于需要以特定用户身份执行命令的场景，例如操作达梦数据库时需要使用dmdba用户权限。

5. 设置超时时间为0（不限制超时）：
```bash
dmshx -hosts "192.168.112.168" -user "root" -password "gaoyuan123#" -cmd "tar -czf backup.tar.gz /opt/dmdata" -timeout 0
```

这种设置适用于执行时间不可预测的长时间运行命令，如备份、大文件传输等。

# 输出到日志文件
dmshx -hosts "192.168.1.10" -user "root" -password "password" -cmd "ls -la" -log-file "output.log"

# 以文本模式实时显示命令执行过程
dmshx -hosts "192.168.112.168" -user "root" -password "gaoyuan123#" -cmd "/opt/dmdbms/bin/DmServiceDM01 restart" -json-output=false -real-time

# 启用命令执行日志记录并设置保留天数
dmshx -hosts "192.168.1.10" -user "root" -password "password" -cmd "ls -la" -enable-command-log -log-retention 30
```

### 实时输出模式

dmshx支持实时输出模式，可在执行耗时较长的命令时提供更好的用户体验：

```bash
# 启用实时输出模式（需要设置json-output=false）
dmshx -hosts "192.168.112.168" -user "root" -password "gaoyuan123#" -cmd "/opt/dmdbms/bin/DmServiceDM01 restart" -json-output=false -real-time
```

在实时输出模式下：
1. 命令开始执行时显示提示信息
2. 命令执行过程中的输出会实时显示在终端
3. 命令完成后显示执行结果摘要
4. 最终以文本格式输出完整结果

此模式特别适合执行耗时较长的操作（如数据库启停、备份还原等），使用户可以实时查看执行进度。

**注意**: 实时输出模式只在`-json-output=false`时有效，因为JSON格式必须作为完整结构输出。

### 文件上传

```bash
# 上传单个文件到远程主机
dmshx -hosts "192.168.1.10" -user "root" -password "password" -upload-file "/path/to/localfile.txt" -upload-dir "/opt/destination/"

# 使用私钥上传文件到多台主机
dmshx -hosts "192.168.1.10,192.168.1.11" -user "root" -key "/path/to/id_rsa" -upload-file "/path/to/localfile.txt" -upload-dir "/opt/destination/" -timeout 60

# 设置上传文件的权限
dmshx -hosts "192.168.1.10" -user "root" -password "password" -upload-file "/path/to/script.sh" -upload-dir "/opt/scripts/" -upload-perm 0755

# 从文件读取主机列表上传文件
dmshx -host-file "hosts.txt" -user "root" -password "password" -upload-file "/path/to/config.conf" -upload-dir "/etc/app/"
```

### 文件下载

```bash
# 从远程主机下载单个文件
dmshx -hosts "192.168.1.10" -user "root" -password "password" -remote-path "/opt/source/file.txt" -local-path "/downloads/"

# 从远程主机下载整个目录
dmshx -hosts "192.168.1.10" -user "root" -password "password" -remote-path "/opt/source/dir" -local-path "/downloads/"

# 使用私钥从多台主机下载文件
dmshx -hosts "192.168.1.10,192.168.1.11" -user "root" -key "/path/to/id_rsa" -remote-path "/opt/logs/app.log" -local-path "/backup/logs/" -timeout 60

# 下载并验证MD5校验和
dmshx -hosts "192.168.1.10" -user "root" -password "password" -remote-path "/opt/important-data.zip" -local-path "/backup/" -verify-md5 true

# 设置更大的缓冲区加速下载大文件
dmshx -hosts "192.168.1.10" -user "root" -password "password" -remote-path "/opt/large-file.tar.gz" -local-path "/backup/" -buffer-size 100 -timeout 300

# 从文件读取主机列表下载文件
dmshx -host-file "hosts.txt" -user "root" -password "password" -remote-path "/var/log/syslog" -local-path "/backup/logs/"

# 自己测试用
```

-host "192.168.112.168" -user "root" -password "gaoyuan123#" -cmd "ls -la"

-host "192.168.112.168" -user "root" -password "gaoyuan123#" -upload-file "E:\go_code\dmshx\build_dmshx.bat" -upload-dir "/opt/"

-host "192.168.112.168" -user "root" -password "gaoyuan123#" -remote-path "/opt/build_dmshx.bat" -local-path "E:\downloads\"

# 下载大文件并指定缓冲区大小
-hosts "192.168.112.168" -user "root" -password "gaoyuan123#" -remote-path "/opt/dm_soft/DMDB_INSTALL_SCRIPTS/dm8_20240301_x86_kylin10_64_ent_8.1.3.26_pack26.iso" -local-path "E:\go_code\dmshx" -verify-md5 true -buffer-size 100

# 禁用JSON输出以显示实时进度条
-hosts "192.168.112.168" -user "root" -password "gaoyuan123#" -remote-path "/opt/dm_soft/DMDB_INSTALL_SCRIPTS/dm8_20240301_x86_kylin10_64_ent_8.1.3.26_pack26.iso" -local-path "E:\go_code\dmshx" -buffer-size 100 -json-output=false -verify-md5 true

-host "192.168.112.168" -user "root" -password "gaoyuan123#" -cmd "/opt/dmdbms/bin/DmServiceDM01 restart" -exec-user "dmdba" -json-output=false -real-time

-db-type "dm" -db-host "192.168.112.168" -db-port 5236 -db-user "SYSDBA" -db-pass "Dameng123#" -sql "SELECT * FROM V$INSTANCE" -timeout 60


```