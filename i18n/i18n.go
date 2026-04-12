package i18n

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/wueasy/wueasy-go-tools/result"

	"github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"
)

var (
	bundle *i18n.Bundle
	once   sync.Once
	// defaultLang 默认语言
	defaultLang = language.Chinese
)

// 默认的内存消息
var defaultMessages = map[string]map[string]string{
	"zh": {
		"error":                                    "很抱歉,系统繁忙,请稍后再试！",
		"invalid.path":                             "检测到非法路径访问！",
		"path.too.long":                            "路径长度超出限制！",
		"invalid.path.chars":                       "路径包含非法字符！",
		"ip.access.error":                          "当前IP禁止访问！",
		"captcha.verify.error":                     "验证码不正确!",
		"sign.param.empty":                         "必填参数不能为空!",
		"sign.param.timestamp.error":               "参数格式错误!",
		"sign.check.error":                         "签名验证失败!",
		"sign.request.expire.error":                "请求已过期!",
		"sign.app.invalid.error":                   "无效的应用ID!",
		"referer.error":                            "请求验证失败！",
		"secondary.verify.lock.error":              "验证错误次数过多，请与{{.Time}}后重试！",
		"useragent.error":                          "检测到非法请求!",
		"request.data.format.error":                "请求数据格式错误!",
		"restore.code.empty":                       "验证码不能为空!",
		"login.account.empty":                      "账号不能为空!",
		"login.password.empty":                     "密码不能为空!",
		"login.account.password.error":             "账号或密码不正确!",
		"base64.decode.error":                      "base64解码失败!",
		"decrypt.error":                            "数据解密失败!",
		"authorization.get.error":                  "获取授权信息失败！",
		"authorization.expire.parse.error":         "授权码过期时间解析失败！",
		"authorization.store.error":                "授权码存储失败！",
		"authorization.json.marshal.error":         "授权码响应数据序列化失败！",
		"authorization.encrypt.error":              "授权码响应数据加密失败！",
		"captcha.generate.error":                   "生成图片验证码失败！",
		"captcha.expire.parse.error":               "图片验证码过期时间解析失败！",
		"captcha.store.error":                      "图片验证码存储失败！",
		"captcha.json.marshal.error":               "图片验证码响应数据序列化失败！",
		"captcha.encrypt.error":                    "图片验证码响应数据加密失败！",
		"captcha.body.read.error":                  "读取验证码请求体失败！",
		"captcha.body.parse.error":                 "解析验证码请求体失败！",
		"captcha.get.error":                        "获取验证码失败！",
		"captcha.delete.error":                     "删除验证码缓存失败！",
		"captcha.restore.get.error":                "图片验证码验证失败！",
		"captcha.restore.delete.error":             "删除限流信息失败！",
		"ip.cidr.parse.error":                      "IP网段解析失败！",
		"merchant.body.read.error":                 "读取商户请求体失败！",
		"merchant.body.parse.error":                "解析商户请求体失败！",
		"merchant.url.decode.error":                "URL参数解码失败！",
		"request.encrypt.body.read.error":          "读取加密请求体失败！",
		"rate.limiter.check.error":                 "限流检查失败！",
		"rate.limiter.restore.error":               "设置限流恢复码失败！",
		"secondary.verify.lock.get.error":          "获取二次验证锁定状态失败！",
		"secondary.verify.expire.get.error":        "获取二次验证锁定过期时间失败！",
		"secondary.verify.body.read.error":         "读取二次验证请求体失败！",
		"secondary.verify.body.parse.error":        "解析二次验证请求体失败！",
		"secondary.verify.timeout.parse.error":     "解析二次验证超时时间失败！",
		"secondary.verify.request.marshal.error":   "序列化二次验证请求数据失败！",
		"secondary.verify.request.create.error":    "创建二次验证HTTP请求失败！",
		"secondary.verify.request.send.error":      "发送二次验证HTTP请求失败！",
		"secondary.verify.response.read.error":     "读取二次验证响应数据失败！",
		"secondary.verify.response.status.error":   "二次验证请求失败！",
		"secondary.verify.response.parse.error":    "解析二次验证响应失败！",
		"secondary.verify.count.increment.error":   "增加二次验证错误计数失败！",
		"secondary.verify.count.expire.error":      "设置二次验证错误计数过期时间失败！",
		"secondary.verify.count.delete.error":      "删除二次验证错误计数失败！",
		"secondary.verify.lock.expire.parse.error": "解析二次验证锁定过期时间失败！",
		"secondary.verify.lock.set.error":          "设置二次验证锁定状态失败！",
		"secondary.verify.lock.delete.error":       "删除二次验证锁定状态失败！",
		"session.jwt.logout.check.error":           "检查JWT登出状态失败！",
		"session.redis.get.error":                  "获取Redis会话失败！",
		"session.json.parse.error":                 "解析会话JSON失败！",
		"session.update.interval.parse.error":      "解析会话更新时间间隔失败！",
		"session.data.marshal.error":               "会话数据JSON序列化失败！",
		"session.http.request.create.error":        "创建HTTP请求失败！",
		"session.timeout.parse.error":              "解析超时时间失败！",
		"session.http.request.send.error":          "发送HTTP请求失败！",
		"session.response.read.error":              "读取响应数据失败！",
		"session.response.status.error":            "会话更新请求失败！",
		"session.response.parse.error":             "登录响应数据JSON反序列化失败！",
		"session.redis.update.error":               "更新Redis会话失败！",
		"session.expire.parse.error":               "解析会话过期时间失败！",
		"session.expire.get.error":                 "获取过期时间失败！",
		"session.expire.set.error":                 "设置Redis会话过期时间失败！",
		"session.jwt.logout.set.error":             "设置JWT注销状态失败！",
		"session.authorization.delete.error":       "删除授权信息失败！",
		"session.temp.authorization.delete.error":  "删除临时授权信息失败！",
		"session.login.body.read.error":            "读取登录请求体失败！",
		"session.login.body.parse.error":           "解析登录请求体JSON失败！",
		"session.login.expire.parse.error":         "解析会话过期时间失败！",
		"session.login.data.marshal.error":         "会话数据JSON序列化失败！",
		"session.login.crypto.encrypt.error":       "会话数据加密失败！",
		"session.login.jwt.sign.error":             "JWT token签名失败！",
		"session.login.temp.code.set.error":        "设置临时授权码失败！",
		"session.login.authorization.set.error":    "设置授权信息失败！",
		"session.login.response.marshal.error":     "登录响应数据JSON序列化失败！",
		"session.login.response.encrypt.error":     "登录响应数据加密失败！",
		"user.agent.rule.match.error":              "User-Agent规则匹配失败！",
		"user.agent.default.rule.match.error":      "User-Agent默认规则匹配失败！",
		"filter.strip.prefix.param.error":          "StripPrefix过滤器参数转换失败！",
		"filter.xss.body.read.error":               "XSS过滤器读取请求体失败！",
		"filter.xss.json.parse.error":              "XSS过滤器JSON解析失败！",
		"filter.xss.json.marshal.error":            "XSS过滤器JSON序列化失败！",
		"filter.login.response.read.error":         "读取登录响应数据失败！",
		"filter.login.response.parse.error":        "登录响应数据JSON反序列化失败！",
		"filter.response.encrypt.read.error":       "读取响应数据失败！",
		"filter.response.encrypt.parse.error":      "响应数据JSON反序列化失败！",
		"filter.response.encrypt.marshal.error":    "响应数据JSON序列化失败！",
		"filter.response.encrypt.error":            "响应数据加密失败！",
		"filter.response.log.read.error":           "读取响应数据失败！",
		"sql.injection.detected":                   "检测到SQL注入攻击！",
		"code.authorization.error":                 "授权码错误!",
		"sso.redirect_uri.empty":                   "重定向地址不能为空！",
		"sso.client_id.empty":                      "客户端ID不能为空！",
		"sso.redirect_uri.error":                   "重定向地址错误！",
		"sso.expire.parse.error":                   "过期时间解析失败！",
		"sso.store.error":                          "存储失败！",
		"sso.client_secret.empty":                  "客户端密钥不能为空！",
		"sso.client_id.error":                      "客户端ID错误！",
		"sso.code.empty":                           "授权码不能为空！",
		"sso.code.error":                           "授权码错误！",
		"sso.client_secret.error":                  "客户端密钥错误！",
	},
	"en": {
		"error":                                    "Sorry, the system is busy, please try again later!",
		"invalid.path":                             "Illegal path access detected!",
		"path.too.long":                            "Path length exceeds limit!",
		"invalid.path.chars":                       "Path contains illegal characters!",
		"ip.access.error":                          "Current IP access forbidden!",
		"captcha.verify.error":                     "The verification code is incorrect!",
		"sign.param.empty":                         "Required parameters cannot be empty!",
		"sign.param.timestamp.error":               "Parameter format error!",
		"sign.check.error":                         "Signature verification failed!",
		"sign.request.expire.error":                "Request expired!",
		"sign.app.invalid.error":                   "Invalid application ID!",
		"referer.error":                            "Request verification failed!",
		"secondary.verify.lock.error":              "Too many failed verification attempts, please try again after {{.Time}}!",
		"useragent.error":                          "Illegal request detected!",
		"request.data.format.error":                "Request data format error!",
		"restore.code.empty":                       "The verification code cannot be empty!",
		"login.account.empty":                      "Account cannot be empty!",
		"login.password.empty":                     "Password cannot be empty!",
		"login.account.password.error":             "Incorrect account or password!",
		"base64.decode.error":                      "Base64 decode failed!",
		"decrypt.error":                            "Data decryption failed!",
		"authorization.get.error":                  "Failed to get authorization information!",
		"authorization.expire.parse.error":         "Failed to parse authorization code expiration time!",
		"authorization.store.error":                "Failed to store authorization code!",
		"authorization.json.marshal.error":         "Failed to serialize authorization code response data!",
		"authorization.encrypt.error":              "Failed to encrypt authorization code response data!",
		"captcha.generate.error":                   "Failed to generate image verification code!",
		"captcha.expire.parse.error":               "Image captcha expiration time parsing failed!",
		"captcha.store.error":                      "Image verification code storage failed!",
		"captcha.json.marshal.error":               "Image captcha response data serialization failed!",
		"captcha.encrypt.error":                    "Failed to encrypt captcha response data!",
		"captcha.body.read.error":                  "Failed to read captcha request body!",
		"captcha.body.parse.error":                 "Failed to parse captcha request body!",
		"captcha.get.error":                        "Failed to get captcha!",
		"captcha.delete.error":                     "Failed to delete captcha cache!",
		"captcha.restore.get.error":                "Image captcha verification failed!",
		"captcha.restore.delete.error":             "Failed to delete rate limit information!",
		"ip.cidr.parse.error":                      "Failed to parse IP CIDR!",
		"merchant.body.read.error":                 "Failed to read merchant request body!",
		"merchant.body.parse.error":                "Failed to parse merchant request body!",
		"merchant.url.decode.error":                "Failed to decode URL parameters!",
		"request.encrypt.body.read.error":          "Failed to read encrypted request body!",
		"rate.limiter.check.error":                 "Rate limiter check failed!",
		"rate.limiter.restore.error":               "Failed to set rate limiter restore code!",
		"secondary.verify.lock.get.error":          "Failed to get secondary verification lock status!",
		"secondary.verify.expire.get.error":        "Failed to get secondary verification lock expiration time!",
		"secondary.verify.body.read.error":         "Failed to read secondary verification request body!",
		"secondary.verify.body.parse.error":        "Failed to parse secondary verification request body!",
		"secondary.verify.timeout.parse.error":     "Failed to parse secondary verification timeout!",
		"secondary.verify.request.marshal.error":   "Failed to serialize secondary verification request data!",
		"secondary.verify.request.create.error":    "Failed to create secondary verification HTTP request!",
		"secondary.verify.request.send.error":      "Failed to send secondary verification HTTP request!",
		"secondary.verify.response.read.error":     "Failed to read secondary verification response data!",
		"secondary.verify.response.status.error":   "Secondary verification request failed!",
		"secondary.verify.response.parse.error":    "Failed to parse secondary verification response!",
		"secondary.verify.count.increment.error":   "Failed to increment secondary verification error count!",
		"secondary.verify.count.expire.error":      "Failed to set secondary verification error count expiration!",
		"secondary.verify.count.delete.error":      "Failed to delete secondary verification error count!",
		"secondary.verify.lock.expire.parse.error": "Failed to parse secondary verification lock expiration time!",
		"secondary.verify.lock.set.error":          "Failed to set secondary verification lock status!",
		"secondary.verify.lock.delete.error":       "Failed to delete secondary verification lock status!",
		"session.jwt.logout.check.error":           "Failed to check JWT logout status!",
		"session.redis.get.error":                  "Failed to get Redis session!",
		"session.json.parse.error":                 "Failed to parse session JSON!",
		"session.update.interval.parse.error":      "Failed to parse session update interval!",
		"session.data.marshal.error":               "Failed to serialize session data to JSON!",
		"session.http.request.create.error":        "Failed to create HTTP request!",
		"session.timeout.parse.error":              "Failed to parse timeout!",
		"session.http.request.send.error":          "Failed to send HTTP request!",
		"session.response.read.error":              "Failed to read response data!",
		"session.response.status.error":            "Session update request failed!",
		"session.response.parse.error":             "Failed to deserialize login response data JSON!",
		"session.redis.update.error":               "Failed to update Redis session!",
		"session.expire.parse.error":               "Failed to parse session expiration time!",
		"session.expire.get.error":                 "Failed to get expiration time!",
		"session.expire.set.error":                 "Failed to set Redis session expiration time!",
		"session.jwt.logout.set.error":             "Failed to set JWT logout status!",
		"session.authorization.delete.error":       "Failed to delete authorization information!",
		"session.temp.authorization.delete.error":  "Failed to delete temporary authorization information!",
		"session.login.body.read.error":            "Failed to read login request body!",
		"session.login.body.parse.error":           "Failed to parse login request body JSON!",
		"session.login.expire.parse.error":         "Failed to parse session expiration time!",
		"session.login.data.marshal.error":         "Failed to serialize session data to JSON!",
		"session.login.crypto.encrypt.error":       "Failed to encrypt session data!",
		"session.login.jwt.sign.error":             "Failed to sign JWT token!",
		"session.login.temp.code.set.error":        "Failed to set temporary authorization code!",
		"session.login.authorization.set.error":    "Failed to set authorization information!",
		"session.login.response.marshal.error":     "Failed to serialize login response data to JSON!",
		"session.login.response.encrypt.error":     "Failed to encrypt login response data!",
		"user.agent.rule.match.error":              "User-Agent rule matching failed!",
		"user.agent.default.rule.match.error":      "User-Agent default rule matching failed!",
		"filter.strip.prefix.param.error":          "StripPrefix filter parameter conversion failed!",
		"filter.xss.body.read.error":               "XSS filter failed to read request body!",
		"filter.xss.json.parse.error":              "XSS filter JSON parsing failed!",
		"filter.xss.json.marshal.error":            "XSS filter JSON marshaling failed!",
		"filter.login.response.read.error":         "Failed to read login response data!",
		"filter.login.response.parse.error":        "Failed to deserialize login response data JSON!",
		"filter.response.encrypt.read.error":       "Failed to read response data!",
		"filter.response.encrypt.parse.error":      "Failed to deserialize response data JSON!",
		"filter.response.encrypt.marshal.error":    "Failed to serialize response data JSON!",
		"filter.response.encrypt.error":            "Failed to encrypt response data!",
		"filter.response.log.read.error":           "Failed to read response data!",
		"sql.injection.detected":                   "SQL injection attack detected!",
		"code.authorization.error":                 "Authorization code error!",
		"sso.redirect_uri.empty":                   "Redirect URI cannot be empty!",
		"sso.client_id.empty":                      "Client ID cannot be empty!",
		"sso.redirect_uri.error":                   "Redirect URI error!",
		"sso.expire.parse.error":                   "Expire time parse error!",
		"sso.store.error":                          "Store error!",
		"sso.client_secret.empty":                  "Client secret cannot be empty!",
		"sso.client_id.error":                      "Client ID error!",
		"sso.code.empty":                           "Authorization code cannot be empty!",
		"sso.code.error":                           "Authorization code error!",
		"sso.client_secret.error":                  "Client secret error!",
	},
}

// 错误码映射
var errorCodes = map[string]int{
	"error":                                    -1,
	"invalid.path":                             -101,
	"path.too.long":                            -102,
	"invalid.path.chars":                       -103,
	"ip.access.error":                          -104,
	"captcha.verify.error":                     -105,
	"sign.param.empty":                         -106,
	"sign.param.timestamp.error":               -107,
	"sign.check.error":                         -108,
	"sign.request.expire.error":                -109,
	"sign.app.invalid.error":                   -110,
	"referer.error":                            -111,
	"secondary.verify.lock.error":              -112,
	"useragent.error":                          -113,
	"request.data.format.error":                -114,
	"restore.code.empty":                       -115,
	"login.account.empty":                      -116,
	"login.password.empty":                     -117,
	"login.account.password.error":             -118,
	"base64.decode.error":                      -119,
	"decrypt.error":                            -120,
	"authorization.get.error":                  -121,
	"authorization.expire.parse.error":         -122,
	"authorization.store.error":                -123,
	"authorization.json.marshal.error":         -124,
	"authorization.encrypt.error":              -125,
	"captcha.generate.error":                   -126,
	"captcha.expire.parse.error":               -127,
	"captcha.store.error":                      -128,
	"captcha.json.marshal.error":               -129,
	"captcha.encrypt.error":                    -130,
	"captcha.body.read.error":                  -131,
	"captcha.body.parse.error":                 -132,
	"captcha.get.error":                        -133,
	"captcha.delete.error":                     -134,
	"captcha.restore.get.error":                -135,
	"captcha.restore.delete.error":             -136,
	"ip.cidr.parse.error":                      -137,
	"merchant.body.read.error":                 -138,
	"merchant.body.parse.error":                -139,
	"merchant.url.decode.error":                -140,
	"request.encrypt.body.read.error":          -141,
	"rate.limiter.check.error":                 -142,
	"rate.limiter.restore.error":               -143,
	"secondary.verify.lock.get.error":          -144,
	"secondary.verify.expire.get.error":        -145,
	"secondary.verify.body.read.error":         -146,
	"secondary.verify.body.parse.error":        -147,
	"secondary.verify.timeout.parse.error":     -148,
	"secondary.verify.request.marshal.error":   -149,
	"secondary.verify.request.create.error":    -150,
	"secondary.verify.request.send.error":      -151,
	"secondary.verify.response.read.error":     -152,
	"secondary.verify.response.status.error":   -153,
	"secondary.verify.response.parse.error":    -154,
	"secondary.verify.count.increment.error":   -155,
	"secondary.verify.count.expire.error":      -156,
	"secondary.verify.count.delete.error":      -157,
	"secondary.verify.lock.expire.parse.error": -158,
	"secondary.verify.lock.set.error":          -159,
	"secondary.verify.lock.delete.error":       -160,
	"session.jwt.logout.check.error":           -161,
	"session.redis.get.error":                  -162,
	"session.json.parse.error":                 -163,
	"session.update.interval.parse.error":      -164,
	"session.data.marshal.error":               -165,
	"session.http.request.create.error":        -166,
	"session.timeout.parse.error":              -167,
	"session.http.request.send.error":          -168,
	"session.response.read.error":              -169,
	"session.response.status.error":            -170,
	"session.response.parse.error":             -171,
	"session.redis.update.error":               -172,
	"session.expire.parse.error":               -173,
	"session.expire.get.error":                 -174,
	"session.expire.set.error":                 -175,
	"session.jwt.logout.set.error":             -176,
	"session.authorization.delete.error":       -177,
	"session.temp.authorization.delete.error":  -178,
	"session.login.body.read.error":            -179,
	"session.login.body.parse.error":           -180,
	"session.login.expire.parse.error":         -181,
	"session.login.data.marshal.error":         -182,
	"session.login.crypto.encrypt.error":       -183,
	"session.login.jwt.sign.error":             -184,
	"session.login.temp.code.set.error":        -185,
	"session.login.authorization.set.error":    -186,
	"session.login.response.marshal.error":     -187,
	"session.login.response.encrypt.error":     -188,
	"user.agent.rule.match.error":              -189,
	"user.agent.default.rule.match.error":      -190,
	"filter.strip.prefix.param.error":          -191,
	"filter.xss.body.read.error":               -192,
	"filter.xss.json.parse.error":              -193,
	"filter.xss.json.marshal.error":            -194,
	"filter.login.response.read.error":         -195,
	"filter.login.response.parse.error":        -196,
	"filter.response.encrypt.read.error":       -197,
	"filter.response.encrypt.parse.error":      -198,
	"filter.response.encrypt.marshal.error":    -199,
	"filter.response.encrypt.error":            -200,
	"filter.response.log.read.error":           -201,
	"sql.injection.detected":                   -202,
	"code.authorization.error":                 -203,
	"sso.redirect_uri.empty":                   -204,
	"sso.client_id.empty":                      -205,
	"sso.redirect_uri.error":                   -206,
	"sso.expire.parse.error":                   -207,
	"sso.store.error":                          -208,
	"sso.client_secret.empty":                  -209,
	"sso.client_id.error":                      -210,
	"sso.code.empty":                           -211,
	"sso.code.error":                           -212,
	"sso.client_secret.error":                  -213,
}

// TranslateFailResult 翻译失败结果，返回带有翻译消息的失败响应
// messageID: 消息ID
// lang: 语言标识
func TranslateFailResult(messageID string, lang string) result.ResultVo[any] {
	code := errorCodes[messageID]
	if code == 0 {
		code = -1
	}
	return result.Fail(code, TranslateWithoutData(messageID, lang))
}

// Config i18n配置
type Config struct {
	// 语言文件目录
	LocaleDir string
	// 默认语言
	DefaultLang language.Tag
}

// Init 初始化i18n
func Init(config Config) error {
	var err error
	once.Do(func() {
		if config.DefaultLang != language.Und {
			defaultLang = config.DefaultLang
		}

		// 初始化bundle
		bundle = i18n.NewBundle(defaultLang)
		bundle.RegisterUnmarshalFunc("json", json.Unmarshal)

		// 加载默认内存消息
		loadDefaultMessages()

		// 如果配置了语言文件目录，则加载文件
		if config.LocaleDir != "" {
			err = loadMessageFiles(config.LocaleDir)
		}
	})
	return err
}

// loadDefaultMessages 加载默认的内存消息
func loadDefaultMessages() {
	for lang, messages := range defaultMessages {
		// 创建消息
		for msgID, msgText := range messages {
			bundle.AddMessages(language.Make(lang), &i18n.Message{
				ID:    msgID,
				Other: msgText,
			})
		}
	}
}

// loadMessageFiles 加载指定目录下的所有语言文件
func loadMessageFiles(dir string) error {
	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if filepath.Ext(path) != ".json" {
			return nil
		}
		_, err = bundle.LoadMessageFile(path)
		return err
	})
}

// TranslateWithoutData 不带模板数据的翻译消息
func TranslateWithoutData(messageID string, lang string) string {
	return Translate(messageID, lang, nil)
}

// Translate 翻译消息
func Translate(messageID string, lang string, templateData map[string]interface{}) string {
	if bundle == nil {
		return messageID
	}

	// 如果未指定语言，使用默认语言
	if lang == "" {
		lang = defaultLang.String()
	}

	localizer := i18n.NewLocalizer(bundle, lang)
	msg, err := localizer.Localize(&i18n.LocalizeConfig{
		MessageID:    messageID,
		TemplateData: templateData,
	})

	if err != nil {
		return messageID
	}
	return msg
}

// T 快捷翻译方法
func T(messageID string, args ...interface{}) string {
	if len(args) == 0 {
		return Translate(messageID, "", nil)
	}

	// 如果只有一个参数且是map类型，则作为模板数据
	if len(args) == 1 {
		if templateData, ok := args[0].(map[string]interface{}); ok {
			return Translate(messageID, "", templateData)
		}
	}

	// 处理格式化参数
	return fmt.Sprintf(Translate(messageID, "", nil), args...)
}

// TL 指定语言的翻译方法
func TL(lang, messageID string, args ...interface{}) string {
	if len(args) == 0 {
		return Translate(messageID, lang, nil)
	}

	// 如果只有一个参数且是map类型，则作为模板数据
	if len(args) == 1 {
		if templateData, ok := args[0].(map[string]interface{}); ok {
			return Translate(messageID, lang, templateData)
		}
	}

	// 处理格式化参数
	return fmt.Sprintf(Translate(messageID, lang, nil), args...)
}

// RegisterMessage 注册单条消息到指定语言
// lang: 语言标识，如 "zh"、"en"
// msgID: 消息ID
// msgText: 消息文本
func RegisterMessage(lang string, msgID string, msgText string) {
	if bundle == nil {
		return
	}
	bundle.AddMessages(language.Make(lang), &i18n.Message{
		ID:    msgID,
		Other: msgText,
	})
}

// RegisterMessages 批量注册消息
// messages: map[语言]map[消息ID]消息文本
func RegisterMessages(messages map[string]map[string]string) {
	if bundle == nil {
		return
	}
	for lang, msgs := range messages {
		for msgID, msgText := range msgs {
			bundle.AddMessages(language.Make(lang), &i18n.Message{
				ID:    msgID,
				Other: msgText,
			})
		}
	}
}
