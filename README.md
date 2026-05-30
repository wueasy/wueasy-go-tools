# wueasy-go-tools

`wueasy-go-tools` 是一个为 Go 语言开发者打造的高效、易用的工具包集合。它封装了后端开发中常用的各种组件和功能模块，旨在减少重复代码编写，提高开发效率并统一代码规范。

## 📦 核心功能模块

*   **配置管理 (`config`)**: 统一的结构化配置定义与管理。
*   **日志系统 (`log`)**: 
    *   基于 `zap` 和 `lumberjack` 的高性能日志组件。
    *   支持日志切割、链路追踪 (TraceId) 并在 Gin 中间件中无缝接入。
    *   **敏感信息脱敏**：支持字段名匹配（JSON / Query）和内容正则匹配（纯文本）两种模式。
    *   四种通用掩码策略（`border` / `replace` / `prefix` / `suffix`），通过配置即可扩展。
*   **数据库客户端 (`db-client`)**: 
    *   基于 `sqlx` 封装的数据库客户端。
    *   支持 MySQL 和 PostgreSQL。
    *   内置原生 SQL 执行日志拦截打印功能，方便开发调试。
*   **Nacos 客户端 (`nacos`)**: 
    *   集成 Nacos 服务注册与发现、配置中心。
    *   内置支持多种负载均衡策略（加权等），实现平滑的服务调用。
*   **Redis 客户端 (`redis`)**: 
    *   基于 `go-redis/v9` 的 Redis 客户端封装，支持简易配置与连接池管理。
*   **API 响应封装 (`result`)**: 
    *   提供统一的 HTTP 接口返回结构 (`Result`)，内置标准状态码、分页模型。
    *   包含用户 Session、登录模型和验证码结构定义。
*   **国际化 (`i18n`)**: 基于 `go-i18n` 的多语言支持工具。
*   **文件服务客户端 (`file-client`)**: 
    *   针对 `wueasy-file-server` 封装的专属客户端工具。
    *   支持 Nacos 服务发现和直接 HTTP 访问。
    *   提供文件的高效流式上传、下载、删除功能，并完整支持大文件分片上传及分片下载。
*   **系统服务 (`system-service`)**: 
    *   基于 `kardianos/service` 封装。
    *   支持将 Go 编译后的程序一键安装、启动、停止和卸载为 Windows/Linux 后台系统服务。
*   **路径匹配 (`ant-path-matcher`)**: 提供类似 Spring 的 Ant 风格路径匹配工具（例如 `/api/**/*.go`）。
*   **启动参数解析 (`startup-parameter`)**: 统一的命令行或启动参数解析封装。
*   **实用工具 (`utils`)**:
    *   **加密解密**: 支持 RSA、SM4 (国密)、DES3 等常见加解密算法。
    *   **文件处理**: 文件大小格式化转换等。
    *   **字符串/基础工具**: 常用数据处理、随机数生成、Base64 等转换函数。

---

## 🚀 快速开始

### 1. 引入依赖

如果是本地项目，可以在 `go.mod` 中使用 `replace` 引入，或者如果已推送到远程仓库，可直接 `go get`：

```bash
# 假设有远程仓库
go get github.com/wueasy/wueasy-go-tools

# 或者在本地使用 replace
# replace github.com/wueasy/wueasy-go-tools => ../wueasy-go-tools
```

代码中统一使用 `github.com/wueasy/wueasy-go-tools/...` 的路径进行导入：
```go
import "github.com/wueasy/wueasy-go-tools/log"
import "github.com/wueasy/wueasy-go-tools/utils"
```

### 2. 常见使用示例

#### 📌 日志记录 (Log)
```go
import "github.com/wueasy/wueasy-go-tools/log"

func main() {
    // 记录普通日志
    log.Info("服务启动成功")
    
    // 带有 TraceId 的上下文日志
    log.CtxInfo(ctx, "收到新请求", "userId", 123)
}
```

#### 📌 日志脱敏 (Sensitive Data Masking)

日志脱敏在 `Init()` 时配置生效，通过 `Ctx(ctx)` 输出的日志会**自动脱敏**，无需手动调用任何转换函数。

**初始化配置**：

```go
import (
    "github.com/wueasy/wueasy-go-tools/config"
    "github.com/wueasy/wueasy-go-tools/log"
)

func main() {
    log.Init("./", config.LogConfig{
        Level: "debug",
        Sensitive: config.SensitiveConfig{
            MaxLength: 100,
            // 字段名匹配——用于 Infow 结构化字段、JSON 响应体
            FieldRules: []config.FieldRule{
                {FieldNames: []string{"mobile", "phone"}, Type: "mobile"},
                {FieldNames: []string{"password", "pwd"}, Type: "password"},
                {FieldNames: []string{"bankcard", "cardno"}, Type: "bankcard"},
                {FieldNames: []string{"email"}, Type: "email"},
                {FieldNames: []string{"idcard"}, Type: "idcard"},
                {FieldNames: []string{"name", "realname"}, Type: "name"},
            },
            // 内容正则匹配——用于 Infof 消息体、纯文本日志
            ContentRules: []config.ContentRule{
                {Type: "mobile"},
                {Type: "email"},
                {Type: "ip"},
            },
        },
    })
}
```

> **YAML 配置**（`application.yml`）：
> ```yaml
> log:
>   level: debug
>   sensitive:
>     max-length: 100
>     field-rules:
>       - field-names: ["mobile", "phone"]
>         type: "mobile"
>       - field-names: ["password"]
>         type: "password"
>       # 自定义掩码策略
>       - field-names: ["custom_no"]
>         mask:
>           strategy: "border"
>           prefix-keep: 2
>           suffix-keep: 3
>           mask-char: "#"
>     content-rules:
>       - type: "mobile"
>       - type: "email"
>       - type: "ip"
> ```

**自动脱敏——推荐用法**（配置后无需手动调用）：

```go
ctx := log.NewContext(context.Background(), "trace-001")

// ✅ 推荐：敏感字段用 Infow（结构化 key-value，可走字段名匹配）
log.Ctx(ctx).Infow("用户注册",
    "mobile", "13800138000",
    "password", "abc123",
    "email", "test@example.com",
)
// 输出：用户注册 {"mobile":"138****8000","password":"******","email":"tes*************"}

// ✅ Infof 消息体——走内容正则匹配（mobile/email/ip 等有正则的类型）
log.Ctx(ctx).Infof("用户13800138000登录，邮箱test@example.com")
// 输出：用户138****8000登录，邮箱tes***@example.com

// ✅ JSON 响应体自动检测（字段值看起来像 JSON 则递归脱敏）
body := `{"user":{"mobile":"13800138000"},"password":"abc123"}`
log.Ctx(ctx).Infow("HTTP响应", "response", body)
// 输出：HTTP响应 {"response":"{\"password\":\"******\",\"user\":{\"mobile\":\"138****8000\"}}"}
```

> **Infof vs Infow 差异**：`Infof` 会将参数格式化成一条消息字符串，丢失字段 key，只能走内容正则。`Infow` 保留 key-value 结构，可同时走字段名匹配 + 内容正则 + JSON 自动检测，**脱敏更精确**。建议敏感字段统一使用 `Infow`。

**手动脱敏 API**（适用于在日志输出前对数据进行脱敏）：

```go
// 单个值脱敏
log.Desensitize("13800138000", log.Mobile)  // 138****8000

// JSON 脱敏（递归处理嵌套字段）
body := `{"user":{"mobile":"13800138000"},"password":"abc123"}`
log.DesensitizeJSON(body)   // 含长字符串截断
log.DesensitizeJSON2(body)  // 不含长字符串截断

// Query 参数脱敏
log.DesensitizeQuery("mobile=13800138000&name=张三&page=1")

// 纯文本脱敏（内容正则匹配）
log.DesensitizeText("用户13800138000的邮箱test@example.com")
```

**四种掩码策略**：

| 策略 | 说明 | 输入 | 输出 |
|------|------|------|------|
| `border` | 保留首尾 | `13800138000` | `138****8000` |
| `replace` | 全部替换 | `abc123` | `******` |
| `prefix` | 仅保留前缀 | `wxid_abc` | `wxi******` |
| `suffix` | 仅保留后缀 | `123456` | `***456` |

**预设类型及内置内容正则**：

| 类型 | 内容正则 | 说明 |
|------|----------|------|
| `mobile` | `1[3-9]\d{9}` | 手机号 |
| `idcard` | `\d{17}[\dXx]` | 身份证号 |
| `bankcard` | `\d{16,19}` | 银行卡号 |
| `email` | `[a-zA-Z0-9._%+-]+@…` | 邮箱 |
| `ip` | `\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}` | IP地址 |
| `creditcode` | `[0-9A-HJ-NPQRTUWXY]{2}\d{6}…` | 统一信用代码 |
| `qq` | `[1-9]\d{4,10}` | QQ号 |
| `password` | — | 无正则，仅字段名匹配 |
| `name` | — | 无正则，仅字段名匹配 |

> **注意**：`content-rules` 中若已填写 `pattern` 则优先使用自定义正则，否则使用预设正则。若类型无预设正则（如 `name`、`password`），纯文本脱敏会跳过该规则。

#### 📌 数据库连接 (DB Client)
```go
import (
    "github.com/wueasy/wueasy-go-tools/config"
    dbClient "github.com/wueasy/wueasy-go-tools/db-client"
)

func initDB() {
    cfg := config.DbConfig{
        DriverName: "mysql",
        Uri:        "127.0.0.1:3306/mydb?charset=utf8mb4&parseTime=True&loc=Local",
        Username:   "root",
        Password:   "123456",
        ShowSql:    true, // 开启 SQL 日志打印
    }
    
    db, err := dbClient.Init(cfg)
    if err != nil {
        panic(err)
    }
    // 使用 db (sqlx.DB) 进行原生或便捷查询操作
}
```

#### 📌 统一返回结果 (Result)
```go
import (
    "github.com/wueasy/wueasy-go-tools/result"
    "github.com/gin-gonic/gin"
)

func GetUserInfo(c *gin.Context) {
    user := map[string]interface{}{"name": "admin", "age": 18}
    // 快速返回成功响应：{ "code": 0, "msg": "success", "data": {...} }
    c.JSON(200, result.SuccessData(user))
}
```

#### 📌 Nacos 服务注册
```go
import (
    "github.com/wueasy/wueasy-go-tools/config"
    nacosClient "github.com/wueasy/wueasy-go-tools/nacos"
)

func initNacos() {
    cfg := config.NacosConfig{
        IpAddr: "127.0.0.1",
        Port:   8848,
    }
    // 初始化并注册服务
    nacosClient.RegisterService(cfg)
}
```

#### 📌 文件服务客户端 (File Client)
```go
import (
    "github.com/wueasy/wueasy-go-tools/file-client"
)

func main() {
    // 创建客户端，默认支持从 Nacos 获取服务地址，也可以直接配置 BaseUrl
    client := fileClient.NewFileClient("file-server", "DEFAULT_GROUP").
        SetBaseUrl("http://127.0.0.1:9830")

    ctx := context.Background()
    
    // 1. 上传本地文件
    uploadResp, err := client.UploadLocalFile(ctx, "document", "/path/to/test.pdf")
    if err != nil {
        panic(err)
    }

    // 2. 下载文件
    fileData, err := client.Download(ctx, "document", uploadResp.Data.FilePath)
    
    // 3. 删除文件
    _, err = client.Delete(ctx, "document", uploadResp.Data.FilePath)
}
```

---

## 📄 开源协议

本项目遵循 [Apache License 2.0](LICENSE) 协议。
