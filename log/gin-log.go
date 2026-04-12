package log

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"strings"
	"time"

	"github.com/wueasy/wueasy-go-tools/user"

	antpathmatcher "github.com/wueasy/wueasy-go-tools/ant-path-matcher"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type LogOptions struct {
	BreakpointConfig *BreakpointConfig
}

type LogOption func(*LogOptions)

func WithBreakpointConfig(config *BreakpointConfig, serviceName string) LogOption {
	return func(o *LogOptions) {
		o.BreakpointConfig = config
		if serviceName != "" {
			o.BreakpointConfig.ServiceName = serviceName
		}
	}
}

// responseWriter 用于捕获响应体
type responseWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w responseWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

func (w responseWriter) WriteString(s string) (int, error) {
	w.body.WriteString(s)
	return w.ResponseWriter.WriteString(s)
}

func GinLogger(opts ...LogOption) gin.HandlerFunc {
	options := &LogOptions{}
	for _, opt := range opts {
		opt(options)
	}

	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		requestId := user.GetRequestId(c)
		ip := user.GetIp(c)
		if requestId == "" {
			requestId = uuid.New().String()
		}
		if ip == "" {
			ip = c.ClientIP()
		}

		// 将 traceId 注入到 Gin 的 context 中，并更新 Request 的 context
		ctx := NewContext(c.Request.Context(), requestId)
		c.Request = c.Request.WithContext(ctx)

		// 检查是否开启断点
		var isBreakpoint bool
		if options.BreakpointConfig != nil && options.BreakpointConfig.Enabled && options.BreakpointConfig.Handler != nil {
			isBreakpoint = checkBreakpoint(c, options.BreakpointConfig, path, ip)
		}

		// 获取请求参数
		var requestParams string
		var rawBody []byte

		if c.Request.Method == "GET" {
			requestParams = DesensitizeQuery(query)
		} else {
			if strings.Contains(c.Request.Header.Get("Content-Type"), "multipart/form-data") {
				requestParams = "[文件上传]"
			} else {
				body, err := c.GetRawData()
				if err == nil && len(body) > 0 {
					rawBody = body
					requestParams = DesensitizeJSON(string(body))
					c.Request.Body = io.NopCloser(bytes.NewBuffer(body))
				}
			}
		}

		// 记录断点请求信息
		if isBreakpoint {
			sendBreakpointRequest(c, options.BreakpointConfig, start, path, ip, requestId, rawBody)
		}

		Ctx(c.Request.Context()).Infof("开始接口请求[%s]，请求参数：%s，访问ip：%s",
			path,
			requestParams,
			ip,
		)

		var rw *responseWriter
		if isBreakpoint {
			rw = &responseWriter{body: bytes.NewBufferString(""), ResponseWriter: c.Writer}
			c.Writer = rw
		}

		c.Next()

		cost := time.Since(start)
		statusCode := c.Writer.Status()

		// 记录断点响应信息
		if isBreakpoint {
			var respBody string
			if rw != nil {
				respBody = rw.body.String()
			}
			sendBreakpointResponse(c, options.BreakpointConfig, start, requestId, statusCode, respBody, nil)
		}

		Ctx(c.Request.Context()).Infof("结束接口请求[%s]，状态码：%d，执行时间：%.4f毫秒",
			path,
			statusCode,
			float64(cost.Nanoseconds())/1e6,
		)
	}
}

func checkBreakpoint(c *gin.Context, config *BreakpointConfig, path, ip string) bool {
	if len(config.Rules) == 0 {
		return true
	}

	userId := user.GetUserId(c)
	b2 := false

	for _, rule := range config.Rules {
		if antpathmatcher.Matchs(rule.Urls, path) {
			if len(rule.RuleTypes) == 0 {
				b2 = true
				break
			} else {
				for _, ruleItem := range rule.RuleTypes {
					if b2 {
						break
					}
					switch ruleItem.Type {
					case BreakpointRuleTypeIP:
						b2 = ip != "" && matchPattern(ruleItem.Data, ip)
					case BreakpointRuleTypeUSER:
						b2 = userId != "" && matchPattern(ruleItem.Data, userId)
					case BreakpointRuleTypeGATEWAY:
						b2 = c.GetHeader("wueasy-breakpoint") == "true"
					case BreakpointRuleTypeHEADER:
						val := ""
						if ruleItem.FieldName != "" {
							val = c.GetHeader(ruleItem.FieldName)
						}
						b2 = val != "" && matchPattern(ruleItem.Data, val)
					}
				}
				if b2 {
					break
				}
			}
			break
		}
	}
	return b2
}

// matchPattern 执行正则表达式匹配
// 参数:
// - pattern: 正则表达式模式字符串
// - value: 需要匹配的字符串
// 返回:
// - bool: 是否匹配成功
func matchPattern(pattern string, value string) bool {
	// 如果模式或值为空,返回false
	if pattern == "" || value == "" {
		return false
	}

	// 编译正则表达式
	reg, err := regexp.Compile(pattern)
	if err != nil {
		return false
	}

	// 执行匹配
	return reg.MatchString(value)
}

func sendBreakpointRequest(c *gin.Context, config *BreakpointConfig, start time.Time, path, ip, requestId string, rawBody []byte) {
	// 获取所有请求头
	headers := make(map[string]string)
	for k, v := range c.Request.Header {
		headers[k] = strings.Join(v, ",")
	}
	headersJson, _ := json.Marshal(headers)

	// URL参数
	urlParams := make(map[string]string)
	for k, v := range c.Request.URL.Query() {
		urlParams[k] = strings.Join(v, ",")
	}
	urlParamsJson, _ := json.Marshal(urlParams)

	// body参数 (假设如果不是文件上传且是json则转为map)
	var bodyParams string
	if len(rawBody) > 0 {
		var bodyMap map[string]interface{}
		if err := json.Unmarshal(rawBody, &bodyMap); err == nil {
			bodyJson, _ := json.Marshal(bodyMap)
			bodyParams = string(bodyJson)
		} else {
			// 不是json就直接转字符串或者保持为空
			bodyParams = string(rawBody)
		}
	}

	// Session信息
	var sessionStr string
	sessionData, _ := user.GetSessionData(c)
	if sessionData != nil {
		sessionJson, _ := json.Marshal(sessionData)
		sessionStr = string(sessionJson)
	}

	dto := BreakpointAddDto{
		ApiUrl:      path,
		Body:        bodyParams,
		UrlParams:   string(urlParamsJson),
		Headers:     string(headersJson),
		LogType:     BreakpointLogTypeRequest,
		RequestType: c.Request.Method,
		RequestId:   requestId,
		RequestIp:   ip,
		RequestTime: start.Format("2006-01-02 15:04:05.000000"),
		ServiceName: config.ServiceName,
		UserSession: sessionStr,
	}

	config.Handler(dto)
}

func sendBreakpointResponse(c *gin.Context, config *BreakpointConfig, start time.Time, requestId string, statusCode int, responseBody string, err error) {
	// 获取所有响应头
	headers := make(map[string]string)
	for k, v := range c.Writer.Header() {
		headers[k] = strings.Join(v, ",")
	}
	headersJson, _ := json.Marshal(headers)

	dto := BreakpointAddDto{
		RequestId:       requestId,
		ResponseTime:    time.Now().Format("2006-01-02 15:04:05.000000"),
		LogType:         BreakpointLogTypeResponse,
		HttpStatus:      fmt.Sprintf("%d", statusCode),
		ServiceName:     config.ServiceName,
		ResponseHeaders: string(headersJson),
	}

	if err != nil {
		dto.Response = err.Error() // 或者堆栈信息
	} else {
		dto.Response = responseBody
	}

	config.Handler(dto)
}

// GinRecovery 使用 zap 记录 panic 日志
func GinRecovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				Ctx(c.Request.Context()).Error("[Recovery from panic]",
					zap.Any("error", err),
					zap.String("request", c.Request.Method+" "+c.Request.URL.Path),
					zap.String("query", c.Request.URL.RawQuery),
					zap.Stack("stack"),
				)

				c.AbortWithStatusJSON(500, gin.H{
					"code":    500,
					"message": "Internal Server Error",
				})
			}
		}()
		c.Next()
	}
}
