# wueasy-go-tools

`wueasy-go-tools` 是一个为 Go 语言开发者打造的高效、易用的工具包集合。它封装了后端开发中常用的各种组件和功能模块，旨在减少重复代码编写，提高开发效率并统一代码规范。

## 📦 核心功能模块

| 模块 | 路径 | 说明 |
|------|------|------|
| **配置管理** | [`config`](#-配置管理-config) | 统一的结构化配置定义与管理 |
| **日志系统** | [`log`](#-日志系统-log) | 高性能日志 + 自动脱敏 + 链路追踪 + Gin 中间件 |
| **敏感信息脱敏** | [`log`](#-日志脱敏) | 字段名匹配 / 内容正则匹配 / JSON 自动检测 / 自动输出 |
| **数据库客户端** | [`db-client`](#-数据库连接-db-client) | sqlx 封装，支持 MySQL / PostgreSQL |
| **Nacos 客户端** | [`nacos`](#-nacos-客户端) | 服务注册发现 + 配置中心 + 负载均衡 |
| **Redis 客户端** | [`redis`](#-redis-客户端) | go-redis/v9 封装 |
| **API 响应封装** | [`result`](#-统一返回结果-result) | 统一 Result 结构 + 标准状态码 |
| **国际化 i18n** | [`i18n`](#-国际化-i18n) | 多语言支持（zh/en/ru） |
| **文件服务客户端** | [`file-client`](#-文件服务-client) | 上传/下载/删除 + 分片传输 |
| **用户工具** | [`user`](#-用户工具-user) | 从请求头获取 Session 数据 |
| **系统服务** | [`system-service`](#-系统服务-system-service) | Go 程序安装为系统服务 |
| **路径匹配** | [`ant-path-matcher`](#-路径匹配-ant-path-matcher) | Ant 风格路径匹配 |
| **启动参数** | [`startup-parameter`](#-启动参数解析-startup-parameter) | 命令行参数解析 |
| **实用工具** | [`utils`](#-实用工具-utils) | 加密解密 / Base64 / 文件处理 / 随机数等 |

---

## 🚀 快速开始

### 引入依赖

```bash
go get github.com/wueasy/wueasy-go-tools
```

```go
import "github.com/wueasy/wueasy-go-tools/log"
import "github.com/wueasy/wueasy-go-tools/utils"
```

---

## 📌 使用示例

### 🔧 配置管理 (`config`)

统一的结构化配置定义，所有模块共用。

```go
import "github.com/wueasy/wueasy-go-tools/config"

cfg := config.LogConfig{
    Level:      "debug",
    MaxSize:    100,   // MB
    MaxBackups: 30,
    MaxAge:     7,     // 天
    Sensitive: config.SensitiveConfig{
        FieldRules: []config.FieldRule{...},
        ContentRules: []config.ContentRule{...},
    },
}
```

---

### � 日志系统 (`log`)

基于 `zap` 和 `lumberjack` 的高性能日志，支持日志切割、链路追踪 (TraceId) 和 Gin 中间件。

#### 初始化

```go
import "github.com/wueasy/wueasy-go-tools/log"

func main() {
    log.Init("./", config.LogConfig{
        Level:      "debug",
        MaxSize:    100,
        MaxBackups: 30,
        MaxAge:     7,
    })
    log.Info("服务启动成功")
}
```

#### 基本使用

```go
// 普通日志
log.Info("消息")
log.Infof("格式化 %s", "消息")
log.Infow("结构化", "key", "value")

log.Debug("调试信息")
log.Warn("警告信息")
log.Error("错误信息")

// 带 TraceId 的上下文日志
ctx := log.NewContext(context.Background(), "trace-001")
log.Ctx(ctx).Infof("收到新请求, userId=%d", 123)

// 日志刷新
defer log.Sync()
```

#### 级别查询

```go
log.IsDebugEnabled()  // bool
log.IsInfoEnabled()   // bool
log.GetLevel()        // zapcore.Level
```

#### 动态配置

```go
log.UpdateLogLevel("info")
log.UpdateServiceName("my-service")
log.UpdateMaxSize(200)
log.UpdateMaxBackups(50)
log.UpdateMaxAge(30)
log.GetLogConfig()    // 获取当前配置
```

#### Gin 中间件

```go
r := gin.Default()
r.Use(log.GinLogger())    // 请求日志自动包含 TraceId
r.Use(log.GinRecovery())  // panic 恢复
```

#### 断点日志

```go
breakpointCfg := &log.BreakpointConfig{
    Url: "http://log-server/api/logs",
}
r.Use(log.GinLogger(log.WithBreakpointConfig(breakpointCfg, "my-service")))
```

---

### 🔒 日志脱敏

日志脱敏在 `Init()` 时配置生效，通过 `Ctx(ctx)` 输出的日志会**自动脱敏**，无需手动调用转换函数。

#### 配置

```go
log.Init("./", config.LogConfig{
    Level: "debug",
    Sensitive: config.SensitiveConfig{
        MaxLength: 100,
        FieldRules: []config.FieldRule{
            {FieldNames: []string{"mobile", "phone"}, Type: "mobile"},
            {FieldNames: []string{"password", "pwd"}, Type: "password"},
            {FieldNames: []string{"bankcard", "cardno"}, Type: "bankcard"},
            {FieldNames: []string{"email"}, Type: "email"},
            {FieldNames: []string{"idcard"}, Type: "idcard"},
            {FieldNames: []string{"name", "realname"}, Type: "name"},
            // 自定义掩码策略
            {FieldNames: []string{"custom_no"},
                Mask: &config.MaskConfig{
                    Strategy: "border", PrefixKeep: 2, SuffixKeep: 3, MaskChar: "#",
                }},
        },
        ContentRules: []config.ContentRule{
            {Type: "mobile"},
            {Type: "email"},
            {Type: "ip"},
        },
    },
})
```

#### 自动脱敏

```go
ctx := log.NewContext(context.Background(), "trace-001")

// ✅ Infow 结构化字段——走字段名匹配
log.Ctx(ctx).Infow("用户注册",
    "mobile", "13800138000",
    "password", "abc123",
)
// → {"mobile":"138****8000","password":"******"}

// ✅ Infof 消息体——走内容正则匹配
log.Ctx(ctx).Infof("用户13800138000登录，邮箱test@example.com")
// → 用户138****8000登录，邮箱tes***@example.com

// ✅ JSON 响应体自动检测
body := `{"user":{"mobile":"13800138000"},"password":"abc123"}`
log.Ctx(ctx).Infow("HTTP响应", "response", body)
// → {"response":"{\"password\":\"******\",\"user\":{\"mobile\":\"138****8000\"}}"}
```

> **Infof vs Infow**：`Infof` 将参数格式化为一行字符串，丢失字段 key，只能走内容正则。`Infow` 保留 key-value 结构，可同时走字段名匹配 + JSON 自动检测，脱敏更精确。**建议敏感字段使用 Infow**。

#### 手动脱敏 API

```go
log.Desensitize("13800138000", log.Mobile)       // 138****8000
log.DesensitizeJSON(body)                         // JSON 脱敏（含截断）
log.DesensitizeJSON2(body)                        // JSON 脱敏（不截断）
log.DesensitizeQuery("mobile=13800138000&name=张三")
log.DesensitizeText("用户13800138000登录")
```

#### 四种掩码策略

| 策略 | 说明 | 输入 | 输出 |
|------|------|------|------|
| `border` | 保留首尾 | `13800138000` | `138****8000` |
| `replace` | 全部替换 | `abc123` | `******` |
| `prefix` | 仅保留前缀 | `wxid_abc` | `wxi******` |
| `suffix` | 仅保留后缀 | `123456` | `***456` |

#### 预设类型及内置正则

| 类型 | 内容正则 | 说明 |
|------|----------|------|
| `mobile` | `1[3-9]\d{9}` | 手机号 |
| `idcard` | `\d{17}[\dXx]` | 身份证号 |
| `bankcard` | `\d{16,19}` | 银行卡号 |
| `email` | `[a-zA-Z0-9._%+-]+@…` | 邮箱 |
| `ip` | `\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}` | IP 地址 |
| `creditcode` | `[0-9A-HJ-NPQRTUWXY]{2}\d{6}…` | 统一信用代码 |
| `qq` | `[1-9]\d{4,10}` | QQ 号 |
| `password` | — | 无正则，仅字段名匹配 |
| `name` | — | 无正则，仅字段名匹配 |

---

### �️ 数据库连接 (`db-client`)

基于 `sqlx` 封装，支持 MySQL 和 PostgreSQL，内置 SQL 日志打印。

```go
import (
    "github.com/wueasy/wueasy-go-tools/config"
    dbClient "github.com/wueasy/wueasy-go-tools/db-client"
)

db, err := dbClient.Init(config.DbConfig{
    DriverName: "mysql",   // mysql | postgres
    Uri:        "127.0.0.1:3306/mydb?charset=utf8mb4&parseTime=True&loc=Local",
    Username:   "root",
    Password:   "123456",
    ShowSql:    true,       // 开启 SQL 日志
})
```

---

### 🌐 Nacos 客户端

集成 Nacos 服务注册与发现、配置中心，内置加权轮询负载均衡。

#### 服务注册

```go
import nacosClient "github.com/wueasy/wueasy-go-tools/nacos"

nacosClient.RegisterNacos(config.NacosConfig{
    ServerAddr: "127.0.0.1:8848",
    Discovery:  config.DiscoveryConfig{Enabled: true},
}, "my-service", "./", "8080")

defer nacosClient.CloseClient()
```

#### 服务发现

```go
// 获取一个健康的实例
instance, err := nacosClient.GetHealthyInstanceWithGroup("service-name", "DEFAULT_GROUP", "")

// 获取带元数据过滤的实例
instance, err := nacosClient.GetHealthyInstanceWithGroupAndMetadata("service-name", "DEFAULT_GROUP", "", map[string]string{"version": "v2"})

// 获取所有健康实例
instances, err := nacosClient.GetAllHealthyInstances("service-name", "DEFAULT_GROUP")

// 取消订阅
nacosClient.UnsubscribeService("service-name", "DEFAULT_GROUP")
```

#### 配置中心

```go
// 初始化
nacosClient.InitConfig(ctx, des3Key, sm4Key, nacosCfg, "my-server", "./", callback, &myConfig)

// 获取配置
configStr, err := nacosClient.GetConfig("dataId", "DEFAULT_GROUP")

// 监听配置变更
nacosClient.ListenConfig(ctx, des3Key, sm4Key, "dataId", "DEFAULT_GROUP", callback, &myConfig)

defer nacosClient.CloseConfigClient()
```

---

### � Redis 客户端

基于 `go-redis/v9` 封装。

```go
import "github.com/wueasy/wueasy-go-tools/redis"

client := redis.Init(config.RedisConfig{
    Addr:     "127.0.0.1:6379",
    Password: "",
    DB:       0,
})

// 使用 go-redis/v9 原生 API
client.Set(ctx, "key", "value", 0)
val, err := client.Get(ctx, "key").Result()
```

---

### 📋 统一返回结果 (`result`)

统一的 HTTP 接口返回结构，内置标准状态码。

```go
import "github.com/wueasy/wueasy-go-tools/result"

// 成功
c.JSON(200, result.SuccessData(data))

// 成功（自定义消息）
c.JSON(200, result.SuccessDataMsg(data, "操作成功"))

// 失败
c.JSON(200, result.Error("参数错误"))

// 分页成功
c.JSON(200, result.SuccessPage(data, totalCount, pageSize, pageNum))

// Session 数据模型
type SessionData struct {
    UserId            string
    Username          string
    Nickname          string
    IsSystem          bool
    CustomParameterMap map[string]string
}
```

---

### 🌍 国际化 (`i18n`)

多语言支持，内置中文 (zh) / 英文 (en) / 俄语 (ru)。

```go
import "github.com/wueasy/wueasy-go-tools/i18n"

func init() {
    i18n.Init(i18n.Config{
        BundleDir: "./i18n/",
    })
}

// 翻译（带模板参数）
msg := i18n.Translate("login.account.password.error", "zh", map[string]interface{}{"Count": 3})

// 翻译（无模板参数）
msg := i18n.TranslateWithoutData("error", "zh")

// 快捷方法
msg := i18n.TL("en", "error")           // 指定语言
msg := i18n.T("error")                   // 默认语言

// 返回失败 Result
result := i18n.TranslateFailResult("error", "zh")

// 动态注册消息
i18n.RegisterMessage("ru", "custom.key", "Пользовательское сообщение")
i18n.RegisterMessages(map[string]map[string]string{...})
```

---

### � 文件服务 (`file-client`)

针对 wueasy-file-server 封装的客户端，支持 Nacos 服务发现和直接 HTTP。

```go
import fileClient "github.com/wueasy/wueasy-go-tools/file-client"

client := fileClient.NewFileClient("file-server", "DEFAULT_GROUP").
    SetBaseUrl("http://127.0.0.1:9830").  // 可选，不设置则走 Nacos 发现
    SetTimeout(30 * time.Second)

ctx := context.Background()

// 上传本地文件
resp, err := client.UploadLocalFile(ctx, "document", "/path/to/file.pdf")

// 流式上传
resp, err := client.UploadStream(ctx, "images", "photo.jpg", reader, fileSize)

// 分片上传（大文件）
resp, err := client.UploadChunk(ctx, "videos", "movie.mp4", chunkReader, chunkSize, totalSize, chunkIndex, totalChunks)

// 下载文件
data, err := client.Download(ctx, "document", "fileKey")

// 分片下载
data, err := client.DownloadChunk(ctx, "videos", "fileKey", offset, chunkSize)

// 删除文件
_, err = client.Delete(ctx, "document", "fileKey")
```

---

### 👤 用户工具 (`user`)

从 Gin 请求头获取 Session 会话数据。

```go
import "github.com/wueasy/wueasy-go-tools/user"

func handler(c *gin.Context) {
    // 获取用户 ID
    userId := user.GetUserId(c)           // string
    userIdInt := user.GetUserIdInt(c)     // *int64，未获取到返回 nil

    // 获取昵称
    nickname := user.GetNickname(c)

    // 判断是否为超级管理员
    isAdmin := user.IsSystemUser(c)

    // 自定义参数
    val := user.GetCustomParameter(c, "tenantId")
    all := user.GetCustomParameterMap(c)

    // 请求信息
    requestId := user.GetRequestId(c)
    ip := user.GetIp(c)
}
```

---

### 🔄 系统服务 (`system-service`)

将 Go 程序安装为 Windows/Linux 后台服务。

```go
import systemService "github.com/wueasy/wueasy-go-tools/system-service"

func main() {
    systemService.Run("my-service", "我的服务", run)
}

func run() {
    // 服务启动后的主逻辑
}
```

```bash
# 安装为系统服务
./app install
# 启动
./app start
# 停止
./app stop
# 卸载
./app uninstall
```

---

### 🧩 路径匹配 (`ant-path-matcher`)

Ant 风格路径匹配，类似 Spring AntPathMatcher。

```go
import antmatcher "github.com/wueasy/wueasy-go-tools/ant-path-matcher"

antmatcher.Match("/api/**", "/api/user/list")     // true
antmatcher.Match("/api/*.go", "/api/main.go")     // true
antmatcher.Match("/api/{id}", "/api/123")         // true

// 批量匹配
antmatcher.Matchs([]string{"/api/**", "/admin/**"}, "/api/user")

// 清除缓存
antmatcher.ClearCache()
```

---

### 🚩 启动参数解析 (`startup-parameter`)

统一的命令行启动参数解析。

```go
import startup "github.com/wueasy/wueasy-go-tools/startup-parameter"

startup.Init(map[string]string{
    "port": "8080",
    "env":  "dev",
})

port := startup.Get("port")  // "8080"
```

---

### 🛠️ 实用工具 (`utils`)

#### 加密解密

```go
// RSA
enc, _ := utils.RsaEncrypt("data", publicKey)
dec, _ := utils.RsaDecrypt(enc, privateKey)
enc, _ := utils.RsaEncryptOAEP("data", publicKey)
dec, _ := utils.RsaDecryptOAEP(enc, privateKey)

// SM4 国密
cipher, _ := utils.EncryptSM4(plaintext, key)
plain, _ := utils.DecryptSM4(cipher, key)

// DES3
cipher, _ := utils.Encrypt3DES(plaintext, key)
plain, _ := utils.Decrypt3DES(cipher, key)
cipher, _ := utils.Encrypt3DESECB(plaintext, key)
plain, _ := utils.Decrypt3DESECB(cipher, key)

// XOR 异或
enc := utils.XorEncrypt("data", "key")
dec := utils.XorDecrypt(enc, "key")

// 密钥混淆
obfuscated := utils.ObfuscateKey("raw-key")

// 解密配置中的加密内容
utils.DecryptEncryptedContent(ctx, content, utils.CreateEncryptionConfigs(des3Key, sm4Key))
```

#### Base64

```go
enc := utils.Base64Encode([]byte("data"))
enc := utils.Base64EncodeString("data")
dec, _ := utils.Base64Decode(enc)
dec, _ := utils.Base64DecodeString(enc)

enc := utils.Base64URLEncode([]byte("data"))
enc := utils.Base64URLEncodeString("data")
dec, _ := utils.Base64URLDecode(enc)
dec, _ := utils.Base64URLDecodeString(enc)
```

#### 文件处理

```go
size, _ := utils.ParseFileBytes("10MB")   // 10485760
str := utils.FormatFileSize(1048576)       // "1.00 MB"
ext := utils.GetFileExt("photo.jpg")       // ".jpg"
```

#### 网络请求

```go
// 获取请求 IP
ip := utils.GetIpAddr(r)

// 读取请求体
body, _ := utils.GetBody(r)

// 签名参数提取
signMap := utils.GetSignMap(body, "application/json", "MD5")

// 响应写入
utils.ResponseWrite(data, resp)
utils.ResponseWrite2(data, w)
utils.ResponseWrite3(data, w, 500)
```

#### 字符串 / 数据工具

```go
uuid := utils.GetUuid()                          // UUID v7
random := utils.GenerateRandomString(32)          // 随机字符串
val := utils.GetDefaultString("", "default")      // 空值时取默认
root := utils.GetRootPath("ROOT_PATH")             // 从环境变量获取根路径
ver, _ := utils.VersionToNumber("1.2.3")           // 版本号转数字
pageSize, _ := utils.GetPageSize(20)               // 分页大小转换

// 正则匹配
matched, _ := utils.RegexMatch(`\d+`, "abc123")

// Ant 路径匹配
utils.MatchPattern("/api/**", "/api/user/list")

// 环境变量替换 ${VAR_NAME}
utils.ExpandEnv("${HOME}/config.yml")

// SQL 方言转换（? → $1）
pgSQL := utils.GetTransformSql("postgres", mysqlSQL)
pgSQL := utils.ReplaceQuestionToDollar(mysqlSQL)
```

#### 配置读取

```go
utils.ReadConfig(ctx, "config.yml", &myConfig, des3Key, sm4Key)
```

---

## 📄 开源协议

本项目遵循 [Apache License 2.0](LICENSE) 协议。
