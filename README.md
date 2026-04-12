# wueasy-go-tools

`wueasy-go-tools` 是一个为 Go 语言开发者打造的高效、易用的工具包集合。它封装了后端开发中常用的各种组件和功能模块，旨在减少重复代码编写，提高开发效率并统一代码规范。

## 📦 核心功能模块

*   **配置管理 (`config`)**: 统一的结构化配置定义与管理。
*   **日志系统 (`log`)**: 
    *   基于 `zap` 和 `lumberjack` 的高性能日志组件。
    *   支持日志切割、敏感信息脱敏。
    *   支持链路追踪 (TraceId) 并在 Gin 中间件中无缝接入。
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

---

## 📄 开源协议

本项目遵循 [Apache License 2.0](LICENSE) 协议。
