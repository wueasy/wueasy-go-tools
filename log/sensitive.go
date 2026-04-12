package log

import (
	"fmt"
	"regexp"
	"slices"
	"strings"
	"sync"

	config2 "github.com/wueasy/wueasy-go-tools/config"
)

// SensitiveType 敏感信息类型
type SensitiveType string

const (
	// 手机号
	Mobile SensitiveType = "mobile"
	// 身份证号
	IDCard SensitiveType = "idcard"
	// 银行卡号
	BankCard SensitiveType = "bankcard"
	// 邮箱
	Email SensitiveType = "email"
	// 密码
	Password SensitiveType = "password"
	// 名称
	Name SensitiveType = "name"
	// 地址
	Address SensitiveType = "address"
	// IP地址
	IP SensitiveType = "ip"
	// 社会统一信用代码
	CreditCode SensitiveType = "creditcode"
	// 护照号码
	Passport SensitiveType = "passport"
	// 军官证号码
	MilitaryID SensitiveType = "militaryid"
	// 营业执照号码
	BusinessLicense SensitiveType = "businesslicense"
	// 车牌号码
	CarNumber SensitiveType = "carnumber"
	// 微信号
	WeChatID SensitiveType = "wechatid"
	// QQ号
	QQ SensitiveType = "qq"
)

var (
	// 默认配置
	defaultConfig = config2.SensitiveConfig{
		// FieldRules: []config2.FieldRule{
		// 	{FieldNames: []string{"mobile", "phone", "tel"}, Type: "mobile"},
		// 	{FieldNames: []string{"idcard", "identity"}, Type: "idcard"},
		// 	{FieldNames: []string{"bankcard", "cardno", "card", "code"}, Type: "bankcard"},
		// 	{FieldNames: []string{"email", "mail"}, Type: "email"},
		// 	{FieldNames: []string{"password", "pwd", "sign", "pass"}, Type: "password"},
		// 	{FieldNames: []string{"name", "username", "realname"}, Type: "name"},
		// 	{FieldNames: []string{"address", "addr"}, Type: "address"},
		// 	{FieldNames: []string{"ip", "ipaddress"}, Type: "ip"},
		// 	{FieldNames: []string{"creditcode", "credit_code", "social_credit_code"}, Type: "creditcode"},
		// 	{FieldNames: []string{"passport", "passport_no"}, Type: "passport"},
		// 	{FieldNames: []string{"militaryid", "military_id"}, Type: "militaryid"},
		// 	{FieldNames: []string{"businesslicense", "business_license", "license"}, Type: "businesslicense"},
		// 	{FieldNames: []string{"carnumber", "car_number", "plate_number"}, Type: "carnumber"},
		// 	{FieldNames: []string{"wechatid", "wechat_id", "wechat"}, Type: "wechatid"},
		// 	{FieldNames: []string{"qq", "qqnumber", "qq_number"}, Type: "qq"},
		// },
		// MaxLength: 100,
	}

	// 当前配置
	currentConfig = defaultConfig

	// 正则表达式缓存
	regexCache      = make(map[string]*regexp.Regexp)
	regexCacheMux   sync.RWMutex
	longStrRegex    *regexp.Regexp
	longStrRegexMux sync.RWMutex
)

// UpdateSensitiveConfig 更新配置
func UpdateSensitiveConfig(config config2.SensitiveConfig) error {
	// 更新当前配置
	currentConfig = config

	// 清空正则表达式缓存，强制重新编译
	regexCacheMux.Lock()
	regexCache = make(map[string]*regexp.Regexp)
	regexCacheMux.Unlock()

	// 重新编译长字符串正则
	if config.MaxLength > 0 {
		longStrRegexMux.Lock()
		longStrRegex = regexp.MustCompile(`"[^"]+"\s*:\s*"([^"]{` + fmt.Sprintf("%d", config.MaxLength) + `,})"`)
		longStrRegexMux.Unlock()
	}

	return nil
}

// getOrCompileRegex 获取或编译正则表达式（带缓存）
func getOrCompileRegex(pattern string) *regexp.Regexp {
	// 先尝试读锁
	regexCacheMux.RLock()
	if regex, ok := regexCache[pattern]; ok {
		regexCacheMux.RUnlock()
		return regex
	}
	regexCacheMux.RUnlock()

	// 需要编译，使用写锁
	regexCacheMux.Lock()
	defer regexCacheMux.Unlock()

	// 双重检查，避免重复编译
	if regex, ok := regexCache[pattern]; ok {
		return regex
	}

	regex := regexp.MustCompile(pattern)
	regexCache[pattern] = regex
	return regex
}

// Desensitize 对字符串进行脱敏处理
func Desensitize(str string, sensitiveType SensitiveType) string {
	if str == "" {
		return str
	}

	switch sensitiveType {
	case Mobile:
		return desensitizeMobile(str)
	case IDCard:
		return desensitizeIDCard(str)
	case BankCard:
		return desensitizeBankCard(str)
	case Email:
		return desensitizeEmail(str)
	case Password:
		return "******"
	case Name:
		return desensitizeName(str)
	case Address:
		return desensitizeAddress(str)
	case IP:
		return desensitizeIP(str)
	case CreditCode:
		return desensitizeCreditCode(str)
	case Passport:
		return desensitizePassport(str)
	case MilitaryID:
		return desensitizeMilitaryID(str)
	case BusinessLicense:
		return desensitizeBusinessLicense(str)
	case CarNumber:
		return desensitizeCarNumber(str)
	case WeChatID:
		return desensitizeWeChatID(str)
	case QQ:
		return desensitizeQQ(str)
	default:
		return str
	}
}

// desensitizeName 姓名脱敏
func desensitizeName(name string) string {
	// 去除首尾空格
	name = strings.TrimSpace(name)
	if name == "" {
		return name
	}

	// 转换为rune切片处理
	nameRunes := []rune(name)
	length := len(nameRunes)
	if length <= 1 {
		return name
	}
	if length == 2 {
		var builder strings.Builder
		builder.Grow(2)
		builder.WriteRune(nameRunes[0])
		builder.WriteRune('*')
		return builder.String()
	}

	var builder strings.Builder
	builder.Grow(length)
	builder.WriteRune(nameRunes[0])
	builder.WriteString(strings.Repeat("*", length-2))
	builder.WriteRune(nameRunes[length-1])
	return builder.String()
}

// desensitizeAddress 地址脱敏
func desensitizeAddress(address string) string {
	// 去除首尾空格
	address = strings.TrimSpace(address)
	if address == "" {
		return address
	}

	addressRunes := []rune(address)
	// 处理特殊情况
	if len(addressRunes) <= 6 {
		return address
	}

	// 分割地址
	parts := strings.Split(address, " ")
	if len(parts) == 1 {
		// 没有空格分隔的地址
		return string(addressRunes[:6]) + strings.Repeat("*", len(addressRunes)-6)
	}

	// 有空格分隔的地址
	var result []string
	for i, part := range parts {
		partRunes := []rune(part)
		if i == 0 {
			// 保留第一个部分
			result = append(result, part)
		} else if i == len(parts)-1 {
			// 保留最后一个部分
			result = append(result, part)
		} else {
			// 中间部分脱敏
			if len(partRunes) > 0 {
				result = append(result, strings.Repeat("*", len(partRunes)))
			} else {
				result = append(result, part)
			}
		}
	}

	return strings.Join(result, " ")
}

// DesensitizeQuery 对URL query参数进行脱敏处理
func DesensitizeQuery(query string) string {
	if query == "" {
		return query
	}

	// 确保配置已加载
	if len(currentConfig.FieldRules) == 0 {
		return query
	}

	// 解析query参数
	params := make(map[string]string)
	for pair := range strings.SplitSeq(query, "&") {
		kv := strings.Split(pair, "=")
		if len(kv) == 2 {
			key := kv[0]
			value := kv[1]

			// 对长参数值进行截取
			if currentConfig.MaxLength > 0 {
				valueRunes := []rune(value)
				if len(valueRunes) > currentConfig.MaxLength {
					var builder strings.Builder
					builder.Grow(currentConfig.MaxLength)
					builder.WriteString(string(valueRunes[:currentConfig.MaxLength-3]))
					builder.WriteString("...")
					value = builder.String()
				}
			}

			// 根据字段规则进行脱敏
			value = desensitizeByFieldRules(key, value)
			params[key] = value
		}
	}

	// 重新组装query字符串
	var builder strings.Builder
	builder.Grow(len(query))
	first := true
	for k, v := range params {
		if !first {
			builder.WriteRune('&')
		}
		builder.WriteString(k)
		builder.WriteRune('=')
		builder.WriteString(v)
		first = false
	}
	return builder.String()
}

// DesensitizeJSON 对JSON字符串中的敏感信息进行脱敏
func DesensitizeJSON(jsonStr string) string {
	if jsonStr == "" {
		return jsonStr
	}

	// 确保配置已加载
	if len(currentConfig.FieldRules) == 0 {
		return jsonStr
	}

	// 根据字段规则进行脱敏
	for _, rule := range currentConfig.FieldRules {
		// 构建字段名正则表达式
		pattern := `"(` + strings.Join(rule.FieldNames, "|") + `)"\s*:\s*("[^"]*"|[^,}\s]+)`
		regex := getOrCompileRegex(pattern)

		// 替换匹配的字段
		jsonStr = regex.ReplaceAllStringFunc(jsonStr, func(match string) string {
			parts := strings.SplitN(match, ":", 2)
			if len(parts) != 2 {
				return match
			}

			// 提取字段值,处理带引号和不带引号的情况
			value := strings.TrimSpace(parts[1])
			if strings.HasPrefix(value, `"`) {
				value = strings.Trim(value, `"`)
			}

			desensitized := Desensitize(value, SensitiveType(rule.Type))

			var builder strings.Builder
			builder.Grow(len(parts[0]) + len(desensitized) + 4)
			builder.WriteString(parts[0])
			builder.WriteString(`:"`)
			builder.WriteString(desensitized)
			builder.WriteRune('"')
			return builder.String()
		})
	}

	// 对长字符串进行截取
	if currentConfig.MaxLength > 0 {
		longStrRegexMux.RLock()
		regex := longStrRegex
		longStrRegexMux.RUnlock()

		if regex != nil {
			jsonStr = regex.ReplaceAllStringFunc(jsonStr, func(match string) string {
				parts := strings.Split(match, `":"`)
				if len(parts) != 2 {
					return match
				}
				value := strings.TrimSuffix(parts[1], `"`)
				runes := []rune(value)
				if len(runes) <= currentConfig.MaxLength {
					return match
				}

				var builder strings.Builder
				builder.Grow(len(parts[0]) + currentConfig.MaxLength + 6)
				builder.WriteString(parts[0])
				builder.WriteString(`":"`)
				builder.WriteString(string(runes[:currentConfig.MaxLength-3]))
				builder.WriteString(`..."`)
				return builder.String()
			})
		}
	}

	return jsonStr
}

// DesensitizeJSON2 对JSON字符串中的敏感信息进行脱敏（不截取长字符串）
func DesensitizeJSON2(jsonStr string) string {
	if jsonStr == "" {
		return jsonStr
	}

	// 确保配置已加载
	if len(currentConfig.FieldRules) == 0 {
		return jsonStr
	}

	// 根据字段规则进行脱敏
	for _, rule := range currentConfig.FieldRules {
		// 构建字段名正则表达式
		pattern := `"(` + strings.Join(rule.FieldNames, "|") + `)"\s*:\s*("[^"]*"|[^,}\s]+)`
		regex := getOrCompileRegex(pattern)

		// 替换匹配的字段
		jsonStr = regex.ReplaceAllStringFunc(jsonStr, func(match string) string {
			parts := strings.SplitN(match, ":", 2)
			if len(parts) != 2 {
				return match
			}

			// 提取字段值,处理带引号和不带引号的情况
			value := strings.TrimSpace(parts[1])
			if strings.HasPrefix(value, `"`) {
				value = strings.Trim(value, `"`)
			}

			desensitized := Desensitize(value, SensitiveType(rule.Type))

			var builder strings.Builder
			builder.Grow(len(parts[0]) + len(desensitized) + 4)
			builder.WriteString(parts[0])
			builder.WriteString(`:"`)
			builder.WriteString(desensitized)
			builder.WriteRune('"')
			return builder.String()
		})
	}

	return jsonStr
}

// desensitizeByFieldRules 根据字段规则进行脱敏
func desensitizeByFieldRules(fieldName string, value string) string {
	fieldName = strings.ToLower(fieldName)
	for _, rule := range currentConfig.FieldRules {
		// 使用 slices.Contains 替代循环
		if slices.Contains(rule.FieldNames, fieldName) {
			return Desensitize(value, SensitiveType(rule.Type))
		}
	}
	return value
}

// desensitizeMobile 手机号脱敏
func desensitizeMobile(mobile string) string {
	// 转换为rune切片处理
	mobileRunes := []rune(mobile)
	if len(mobileRunes) != 11 {
		return mobile
	}

	var builder strings.Builder
	builder.Grow(11)
	builder.WriteString(string(mobileRunes[:3]))
	builder.WriteString("****")
	builder.WriteString(string(mobileRunes[7:]))
	return builder.String()
}

// desensitizeIDCard 身份证号脱敏
func desensitizeIDCard(idCard string) string {
	// 转换为rune切片处理
	idCardRunes := []rune(idCard)
	if len(idCardRunes) != 18 {
		return idCard
	}

	var builder strings.Builder
	builder.Grow(22)
	builder.WriteString(string(idCardRunes[:6]))
	builder.WriteString("********")
	builder.WriteString(string(idCardRunes[14:]))
	return builder.String()
}

// desensitizeBankCard 银行卡号脱敏
func desensitizeBankCard(bankCard string) string {
	// 转换为rune切片处理
	bankCardRunes := []rune(bankCard)
	length := len(bankCardRunes)
	if length < 8 {
		return bankCard
	}

	var builder strings.Builder
	builder.Grow(length)
	builder.WriteString(string(bankCardRunes[:4]))
	builder.WriteString("****")
	builder.WriteString(string(bankCardRunes[length-4:]))
	return builder.String()
}

// desensitizeEmail 邮箱脱敏
func desensitizeEmail(email string) string {
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return email
	}

	// 转换为rune切片处理
	usernameRunes := []rune(parts[0])
	length := len(usernameRunes)

	var builder strings.Builder
	builder.Grow(len(email) + 3)

	if length <= 3 {
		builder.WriteRune(usernameRunes[0])
		builder.WriteString("***@")
	} else {
		builder.WriteString(string(usernameRunes[:3]))
		builder.WriteString("***@")
	}
	builder.WriteString(parts[1])
	return builder.String()
}

// desensitizeIP IP地址脱敏
func desensitizeIP(ip string) string {
	// 去除首尾空格
	ip = strings.TrimSpace(ip)
	if ip == "" {
		return ip
	}

	// 转换为rune切片处理
	ipRunes := []rune(ip)
	if len(ipRunes) == 0 {
		return ip
	}

	// 分割IP地址
	parts := strings.Split(string(ipRunes), ".")
	if len(parts) != 4 {
		return string(ipRunes)
	}

	// 保留前两段，后两段用*替代
	return parts[0] + "." + parts[1] + ".***.***"
}

// desensitizeCreditCode 社会统一信用代码脱敏
func desensitizeCreditCode(code string) string {
	// 去除首尾空格
	code = strings.TrimSpace(code)
	if code == "" {
		return code
	}

	// 转换为rune切片处理
	codeRunes := []rune(code)
	if len(codeRunes) != 18 {
		return code
	}

	// 保留前8位和后4位，中间用*替代
	return string(codeRunes[:8]) + "********" + string(codeRunes[16:])
}

// desensitizePassport 护照号码脱敏
func desensitizePassport(passport string) string {
	// 去除首尾空格
	passport = strings.TrimSpace(passport)
	if passport == "" {
		return passport
	}

	// 转换为rune切片处理
	passportRunes := []rune(passport)
	length := len(passportRunes)
	if length < 8 {
		return passport
	}

	// 保留前4位和后4位，中间用*替代
	var builder strings.Builder
	builder.Grow(length)
	builder.WriteString(string(passportRunes[:4]))
	builder.WriteString(strings.Repeat("*", length-8))
	builder.WriteString(string(passportRunes[length-4:]))
	return builder.String()
}

// desensitizeMilitaryID 军官证号码脱敏
func desensitizeMilitaryID(id string) string {
	// 去除首尾空格
	id = strings.TrimSpace(id)
	if id == "" {
		return id
	}

	// 转换为rune切片处理
	idRunes := []rune(id)
	length := len(idRunes)
	if length < 8 {
		return id
	}

	// 保留前4位和后4位，中间用*替代
	var builder strings.Builder
	builder.Grow(length)
	builder.WriteString(string(idRunes[:4]))
	builder.WriteString(strings.Repeat("*", length-8))
	builder.WriteString(string(idRunes[length-4:]))
	return builder.String()
}

// desensitizeBusinessLicense 营业执照号码脱敏
func desensitizeBusinessLicense(license string) string {
	// 去除首尾空格
	license = strings.TrimSpace(license)
	if license == "" {
		return license
	}

	// 转换为rune切片处理
	licenseRunes := []rune(license)
	if len(licenseRunes) != 15 {
		return license
	}

	// 保留前6位和后4位，中间用*替代
	var builder strings.Builder
	builder.Grow(15)
	builder.WriteString(string(licenseRunes[:6]))
	builder.WriteString("*****")
	builder.WriteString(string(licenseRunes[11:]))
	return builder.String()
}

// desensitizeCarNumber 车牌号码脱敏
func desensitizeCarNumber(number string) string {
	// 去除首尾空格
	number = strings.TrimSpace(number)
	if number == "" {
		return number
	}

	// 转换为rune切片处理
	numberRunes := []rune(number)
	length := len(numberRunes)
	if length < 7 {
		return number
	}

	// 保留前3位和后2位，中间用*替代
	var builder strings.Builder
	builder.Grow(length)
	builder.WriteString(string(numberRunes[:3]))
	builder.WriteString(strings.Repeat("*", length-5))
	builder.WriteString(string(numberRunes[length-2:]))
	return builder.String()
}

// desensitizeWeChatID 微信号脱敏
func desensitizeWeChatID(id string) string {
	// 去除首尾空格
	id = strings.TrimSpace(id)
	idRunes := []rune(id)
	length := len(idRunes)
	if length <= 3 {
		return id
	}

	// 保留前3位，后面用*替代
	var builder strings.Builder
	builder.Grow(length)
	builder.WriteString(string(idRunes[:3]))
	builder.WriteString(strings.Repeat("*", length-3))
	return builder.String()
}

// desensitizeQQ QQ号脱敏
func desensitizeQQ(qq string) string {
	// 去除首尾空格
	qq = strings.TrimSpace(qq)
	qqRunes := []rune(qq)
	length := len(qqRunes)
	if length <= 4 {
		return qq
	}

	// 保留前4位，后面用*替代
	var builder strings.Builder
	builder.Grow(length)
	builder.WriteString(string(qqRunes[:4]))
	builder.WriteString(strings.Repeat("*", length-4))
	return builder.String()
}
