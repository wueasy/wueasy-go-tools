package config

type ServerConfig struct {
	Port string     `yaml:"port"` // 服务端口
	Name string     `yaml:"name"` // 服务名称
	Http HttpConfig `yaml:"http"` // HTTP服务配置
}

// HttpConfig HTTP服务配置
type HttpConfig struct {
	MaxHeaderBytes    string `yaml:"max-header-bytes"`    // 最大请求头大小,如:"2MB","1024KB"
	MaxBodyBytes      string `yaml:"max-body-bytes"`      // 最大请求体大小,如:"10MB","100MB"
	ReadHeaderTimeout string `yaml:"read-header-timeout"` // 读取请求头超时时间,如:"10s","1m"
	WriteTimeout      string `yaml:"write-timeout"`       // 写入响应超时时间,如:"60s","2m"
	ReadTimeout       string `yaml:"read-timeout"`        // 读取请求超时时间,如:"60s","2m"
	IdleTimeout       string `yaml:"idle-timeout"`        // 空闲连接超时时间,如:"120s","5m"
}

type NacosConfig struct {
	ServerAddr string          `yaml:"server-addr"` // nacos配置地址
	Username   string          `yaml:"username"`    // Nacos用户名
	Password   string          `yaml:"password"`    // Nacos密码
	Config     ConfigConfig    `yaml:"config"`      // 配置中心配置
	Discovery  DiscoveryConfig `yaml:"discovery"`   // 注册中心配置
}

// ConfigConfig 配置中心配置
type ConfigConfig struct {
	Namespace string `yaml:"namespace"` // 命名空间
	Group     string `yaml:"group"`     // 分组
	Enabled   bool   `yaml:"enabled"`   // 是否启用
}

// DiscoveryConfig 注册中心配置
type DiscoveryConfig struct {
	Namespace       string            `yaml:"namespace"`         // 命名空间
	Group           string            `yaml:"group"`             // 分组
	Ip              string            `yaml:"ip"`                // 服务IP
	Port            string            `yaml:"port"`              // 服务端口
	Enabled         bool              `yaml:"enabled"`           // 是否启用
	Weight          float64           `yaml:"weight"`            // 权重
	Metadata        map[string]string `yaml:"metadata"`          // 元数据
	LoadBalanceType string            `yaml:"load-balance-type"` // 负载均衡策略: round_robin(轮询), random(随机), weighted_round_robin(加权轮询)
}

type DbConfig struct {
	DriverName   string `yaml:"driver-name"`
	Uri          string `yaml:"uri"`
	Username     string `yaml:"username"`
	Password     string `yaml:"password"`
	MaxOpenConns int    `yaml:"max-open-conns"`
	MaxIdleConns int    `yaml:"max-idle-conns"`
	ShowSql      bool   `yaml:"show-sql"` // 是否输出SQL日志（debug模式）
}

type SystemServiceConfig struct {
	Version     string
	EnvRootPath string
	Name        string
	DisplayName string
	Description string
}

// LogConfig 日志配置
type LogConfig struct {
	Level      string            `yaml:"level"`       // 日志级别
	MaxSize    int               `yaml:"max-size"`    // 日志文件最大大小（MB）
	MaxBackups int               `yaml:"max-backups"` // 最大保留的旧日志文件数量
	MaxAge     int               `yaml:"max-age"`     // 最大保留天数
	Sensitive  SensitiveConfig   `yaml:"sensitive"`   // 脱敏配置
	Request    LogRequestConfig  `yaml:"request"`     // 请求参数配置
	Response   LogResponseConfig `yaml:"response"`    // 响应参数配置
	Async      bool              `yaml:"async"`       // 是否异步写入日志
}

type LogRequestConfig struct {
	Enabled bool `yaml:"enabled"` // 是否启用参数配置
	// Urls 允许的url
	Urls []string `yaml:"urls"`
	// ForbidUrls 禁止的url
	ForbidUrls []string `yaml:"forbid-urls"`
}

type LogResponseConfig struct {
	Enabled bool `yaml:"enabled"` // 是否启用参数配置
	// Urls 允许的url
	Urls []string `yaml:"urls"`
	// ForbidUrls 禁止的url
	ForbidUrls []string `yaml:"forbid-urls"`
}

// SensitiveConfig 脱敏配置
type SensitiveConfig struct {
	// 字段脱敏规则
	FieldRules []FieldRule `yaml:"field-rules"`
	// 长字符串截取长度
	MaxLength int `yaml:"max-length"`
}

// FieldRule 字段脱敏规则
type FieldRule struct {
	// 字段名列表（支持正则表达式）
	FieldNames []string `yaml:"field-names"`
	// 脱敏类型
	Type string `yaml:"type"`
}

// RedisConfig 定义了Redis连接的配置结构
type RedisConfig struct {
	Addrs    string `yaml:"addrs"`    // Redis服务器地址列表
	DB       int    `yaml:"db"`       // Redis数据库索引
	Username string `yaml:"username"` // Redis用户名
	Password string `yaml:"password"` // Redis密码

	// 连接池配置
	PoolSize     int    `yaml:"pool-size"`      // 连接池大小
	MinIdleConns int    `yaml:"min-idle-conns"` // 最小空闲连接数
	MaxIdleConns int    `yaml:"max-idle-conns"` // 最大空闲连接数
	PoolTimeout  string `yaml:"pool-timeout"`   // 连接池超时时间

	// 超时配置
	DialTimeout  string `yaml:"dial-timeout"`  // 连接超时时间
	ReadTimeout  string `yaml:"read-timeout"`  // 读取超时时间
	WriteTimeout string `yaml:"write-timeout"` // 写入超时时间

	// 重试配置
	MaxRetries      int    `yaml:"max-retries"`       // 最大重试次数
	MinRetryBackoff string `yaml:"min-retry-backoff"` // 最小重试间隔
	MaxRetryBackoff string `yaml:"max-retry-backoff"` // 最大重试间隔

	// 集群配置
	MaxRedirects   int  `yaml:"max-redirects"`    // 最大重定向次数
	ReadOnly       bool `yaml:"read-only"`        // 是否只读
	RouteByLatency bool `yaml:"route-by-latency"` // 是否按延迟路由
	RouteRandomly  bool `yaml:"route-randomly"`   // 是否随机路由

	// 哨兵配置
	MasterName string `yaml:"master-name"` // 主节点名称，用于哨兵模式

}
