package log

import (
	"encoding/json"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"sync"

	config2 "github.com/wueasy/wueasy-go-tools/config"
)

type SensitiveType string

const (
	Mobile          SensitiveType = "mobile"
	IDCard          SensitiveType = "idcard"
	BankCard        SensitiveType = "bankcard"
	Email           SensitiveType = "email"
	Password        SensitiveType = "password"
	Name            SensitiveType = "name"
	Address         SensitiveType = "address"
	IP              SensitiveType = "ip"
	CreditCode      SensitiveType = "creditcode"
	Passport        SensitiveType = "passport"
	MilitaryID      SensitiveType = "militaryid"
	BusinessLicense SensitiveType = "businesslicense"
	CarNumber       SensitiveType = "carnumber"
	WeChatID        SensitiveType = "wechatid"
	QQ              SensitiveType = "qq"
)

type typePreset struct {
	mask           config2.MaskConfig
	contentPattern string
}

var typePresets = map[string]typePreset{
	"mobile": {
		mask:           config2.MaskConfig{Strategy: config2.MaskBorder, PrefixKeep: 3, SuffixKeep: 4, MaskChar: "*"},
		contentPattern: `1[3-9]\d{9}`,
	},
	"idcard": {
		mask:           config2.MaskConfig{Strategy: config2.MaskBorder, PrefixKeep: 6, SuffixKeep: 4, MaskChar: "*"},
		contentPattern: `\d{17}[\dXx]`,
	},
	"bankcard": {
		mask:           config2.MaskConfig{Strategy: config2.MaskBorder, PrefixKeep: 4, SuffixKeep: 4, MaskChar: "*"},
		contentPattern: `\d{16,19}`,
	},
	"email": {
		mask:           config2.MaskConfig{Strategy: config2.MaskBorder, PrefixKeep: 3, SuffixKeep: 0, MaskChar: "*"},
		contentPattern: `[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`,
	},
	"password": {
		mask:           config2.MaskConfig{Strategy: config2.MaskReplace, MaskChar: "*"},
		contentPattern: ``,
	},
	"name": {
		mask:           config2.MaskConfig{Strategy: config2.MaskBorder, PrefixKeep: 1, SuffixKeep: 1, MaskChar: "*"},
		contentPattern: ``,
	},
	"address": {
		mask:           config2.MaskConfig{Strategy: config2.MaskBorder, PrefixKeep: 6, SuffixKeep: 0, MaskChar: "*"},
		contentPattern: ``,
	},
	"ip": {
		mask:           config2.MaskConfig{Strategy: config2.MaskBorder, PrefixKeep: 2, SuffixKeep: 0, MaskChar: "*"},
		contentPattern: `\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}`,
	},
	"creditcode": {
		mask:           config2.MaskConfig{Strategy: config2.MaskBorder, PrefixKeep: 8, SuffixKeep: 2, MaskChar: "*"},
		contentPattern: `[0-9A-HJ-NPQRTUWXY]{2}\d{6}[0-9A-HJ-NPQRTUWXY]{10}`,
	},
	"passport": {
		mask:           config2.MaskConfig{Strategy: config2.MaskBorder, PrefixKeep: 4, SuffixKeep: 4, MaskChar: "*"},
		contentPattern: ``,
	},
	"militaryid": {
		mask:           config2.MaskConfig{Strategy: config2.MaskBorder, PrefixKeep: 4, SuffixKeep: 4, MaskChar: "*"},
		contentPattern: ``,
	},
	"businesslicense": {
		mask:           config2.MaskConfig{Strategy: config2.MaskBorder, PrefixKeep: 6, SuffixKeep: 4, MaskChar: "*"},
		contentPattern: ``,
	},
	"carnumber": {
		mask:           config2.MaskConfig{Strategy: config2.MaskBorder, PrefixKeep: 3, SuffixKeep: 2, MaskChar: "*"},
		contentPattern: ``,
	},
	"wechatid": {
		mask:           config2.MaskConfig{Strategy: config2.MaskPrefix, PrefixKeep: 3, MaskChar: "*"},
		contentPattern: ``,
	},
	"qq": {
		mask:           config2.MaskConfig{Strategy: config2.MaskPrefix, PrefixKeep: 4, MaskChar: "*"},
		contentPattern: `[1-9]\d{4,10}`,
	},
}

var (
	defaultConfig = config2.SensitiveConfig{}

	currentConfig = defaultConfig

	regexCache    = make(map[string]*regexp.Regexp)
	regexCacheMux sync.RWMutex

	longStrRegex    *regexp.Regexp
	longStrRegexMux sync.RWMutex

	contentRegexes    = make(map[string]*regexp.Regexp)
	contentRegexesMux sync.RWMutex
)

func getMaskConfig(ruleType string, customMask *config2.MaskConfig) config2.MaskConfig {
	if customMask != nil && customMask.Strategy != "" {
		if customMask.MaskChar == "" {
			customMask.MaskChar = "*"
		}
		return *customMask
	}
	if preset, ok := typePresets[ruleType]; ok {
		return preset.mask
	}
	return config2.MaskConfig{Strategy: config2.MaskReplace, MaskChar: "*"}
}

func applyMask(value string, cfg config2.MaskConfig) string {
	if value == "" {
		return value
	}

	maskChar := cfg.MaskChar
	if maskChar == "" {
		maskChar = "*"
	}

	runes := []rune(value)
	length := len(runes)
	if length == 0 {
		return value
	}

	switch cfg.Strategy {
	case config2.MaskReplace:
		return strings.Repeat(maskChar, length)

	case config2.MaskBorder:
		prefixKeep := cfg.PrefixKeep
		if prefixKeep > length {
			prefixKeep = length
		}
		suffixKeep := cfg.SuffixKeep
		if prefixKeep+suffixKeep > length {
			suffixKeep = length - prefixKeep
		}
		midLen := length - prefixKeep - suffixKeep
		if midLen < 0 {
			midLen = 0
		}
		return string(runes[:prefixKeep]) + strings.Repeat(maskChar, midLen) + string(runes[length-suffixKeep:])

	case config2.MaskPrefix:
		keep := cfg.PrefixKeep
		if keep > length {
			keep = length
		}
		return string(runes[:keep]) + strings.Repeat(maskChar, length-keep)

	case config2.MaskSuffix:
		keep := cfg.SuffixKeep
		if keep > length {
			keep = length
		}
		return strings.Repeat(maskChar, length-keep) + string(runes[length-keep:])

	default:
		return value
	}
}

func getOrCompileRegex(pattern string) *regexp.Regexp {
	regexCacheMux.RLock()
	if regex, ok := regexCache[pattern]; ok {
		regexCacheMux.RUnlock()
		return regex
	}
	regexCacheMux.RUnlock()

	regexCacheMux.Lock()
	defer regexCacheMux.Unlock()

	if regex, ok := regexCache[pattern]; ok {
		return regex
	}

	regex := regexp.MustCompile(pattern)
	regexCache[pattern] = regex
	return regex
}

func getOrCompileContentRegex(pattern string) *regexp.Regexp {
	contentRegexesMux.RLock()
	if regex, ok := contentRegexes[pattern]; ok {
		contentRegexesMux.RUnlock()
		return regex
	}
	contentRegexesMux.RUnlock()

	contentRegexesMux.Lock()
	defer contentRegexesMux.Unlock()

	if regex, ok := contentRegexes[pattern]; ok {
		return regex
	}

	regex := regexp.MustCompile(pattern)
	contentRegexes[pattern] = regex
	return regex
}

func UpdateSensitiveConfig(config config2.SensitiveConfig) error {
	currentConfig = config

	regexCacheMux.Lock()
	regexCache = make(map[string]*regexp.Regexp)
	regexCacheMux.Unlock()

	contentRegexesMux.Lock()
	contentRegexes = make(map[string]*regexp.Regexp)
	contentRegexesMux.Unlock()

	if config.MaxLength > 0 {
		longStrRegexMux.Lock()
		longStrRegex = regexp.MustCompile(`"[^"]+"\s*:\s*"([^"]{` + fmt.Sprintf("%d", config.MaxLength) + `,})"`)
		longStrRegexMux.Unlock()
	}

	return nil
}

func Desensitize(str string, sensitiveType SensitiveType) string {
	if str == "" {
		return str
	}
	if sensitiveType == Email {
		return desensitizeEmail(str)
	}
	cfg := getMaskConfig(string(sensitiveType), nil)
	return applyMask(str, cfg)
}

func desensitizeEmail(email string) string {
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return email
	}
	usernameRunes := []rune(parts[0])
	if len(usernameRunes) == 0 {
		return email
	}
	if len(usernameRunes) <= 3 {
		return string(usernameRunes[0]) + "***@" + parts[1]
	}
	return string(usernameRunes[:3]) + "***@" + parts[1]
}

func DesensitizeJSON(jsonStr string) string {
	return desensitizeJSON(jsonStr, true)
}

func DesensitizeJSON2(jsonStr string) string {
	return desensitizeJSON(jsonStr, false)
}

func desensitizeJSON(jsonStr string, truncate bool) string {
	if jsonStr == "" {
		return jsonStr
	}
	if len(currentConfig.FieldRules) == 0 && len(currentConfig.ContentRules) == 0 && currentConfig.MaxLength <= 0 {
		return jsonStr
	}

	var data any
	if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
		return jsonStr
	}

	walkAndMask(data)

	if truncate && currentConfig.MaxLength > 0 {
		walkAndTruncate(data, currentConfig.MaxLength)
	}

	result, err := json.Marshal(data)
	if err != nil {
		return jsonStr
	}
	return string(result)
}

func walkAndMask(node any) {
	switch v := node.(type) {
	case map[string]any:
		for key, val := range v {
			switch child := val.(type) {
			case string:
				if mask, ok := findFieldMask(key); ok {
					v[key] = applyMask(child, mask)
				}
			case map[string]any:
				walkAndMask(child)
			case []any:
				walkAndMask(child)
			}
		}
	case []any:
		for _, item := range v {
			walkAndMask(item)
		}
	}
}

func walkAndTruncate(node any, maxLen int) {
	switch v := node.(type) {
	case map[string]any:
		for key, val := range v {
			switch child := val.(type) {
			case string:
				runes := []rune(child)
				if len(runes) > maxLen {
					v[key] = string(runes[:maxLen-3]) + "..."
				}
			case map[string]any:
				walkAndTruncate(child, maxLen)
			case []any:
				walkAndTruncate(child, maxLen)
			}
		}
	case []any:
		for _, item := range v {
			walkAndTruncate(item, maxLen)
		}
	}
}

func findFieldMask(fieldName string) (config2.MaskConfig, bool) {
	lowerName := strings.ToLower(fieldName)
	for _, rule := range currentConfig.FieldRules {
		if slices.Contains(rule.FieldNames, lowerName) {
			return getMaskConfig(rule.Type, rule.Mask), true
		}
	}
	return config2.MaskConfig{}, false
}

func DesensitizeText(text string) string {
	if text == "" {
		return text
	}

	if len(currentConfig.ContentRules) == 0 {
		return text
	}

	for _, rule := range currentConfig.ContentRules {
		pattern := rule.Pattern
		if pattern == "" {
			if preset, ok := typePresets[rule.Type]; ok && preset.contentPattern != "" {
				pattern = preset.contentPattern
			} else {
				continue
			}
		}

		regex := getOrCompileContentRegex(pattern)

		text = regex.ReplaceAllStringFunc(text, func(match string) string {
			if rule.Mask != nil && rule.Mask.Strategy != "" {
				return applyMask(match, *rule.Mask)
			}
			return Desensitize(match, SensitiveType(rule.Type))
		})
	}

	return text
}

func DesensitizeQuery(query string) string {
	if query == "" {
		return query
	}

	if len(currentConfig.FieldRules) == 0 {
		return query
	}

	params := make(map[string]string)
	for pair := range strings.SplitSeq(query, "&") {
		kv := strings.Split(pair, "=")
		if len(kv) == 2 {
			key := kv[0]
			value := kv[1]

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

			value = desensitizeByFieldRules(key, value)
			params[key] = value
		}
	}

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

func desensitizeByFieldRules(fieldName string, value string) string {
	fieldName = strings.ToLower(fieldName)
	for _, rule := range currentConfig.FieldRules {
		if slices.Contains(rule.FieldNames, fieldName) {
			mask := getMaskConfig(rule.Type, rule.Mask)
			return applyMask(value, mask)
		}
	}
	return value
}

func IsSensitiveEnabled() bool {
	return len(currentConfig.FieldRules) > 0 || len(currentConfig.ContentRules) > 0
}

func looksLikeJSON(s string) bool {
	s = strings.TrimSpace(s)
	if len(s) < 2 {
		return false
	}
	return (s[0] == '{' && s[len(s)-1] == '}') || (s[0] == '[' && s[len(s)-1] == ']')
}

func desensitizeFieldValue(fieldName, value string) string {
	if value == "" {
		return value
	}

	if mask, ok := findFieldMask(fieldName); ok {
		return applyMask(value, mask)
	}

	if looksLikeJSON(value) && len(currentConfig.FieldRules) > 0 {
		if result := desensitizeJSON(value, true); result != value {
			return result
		}
	}

	if len(currentConfig.ContentRules) > 0 {
		return DesensitizeText(value)
	}

	return value
}
