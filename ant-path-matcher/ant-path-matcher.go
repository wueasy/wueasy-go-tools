package antpathmatcher

import (
	"path"
	"regexp"
	"strings"
	"sync"
)

var patternCache = sync.Map{}

// Matchs 判断给定的路径是否与多个模式中的任意一个相匹配
func Matchs(patterns []string, pathStr string) bool {

	for _, pattern := range patterns {
		if Match(pattern, pathStr) {
			return true
		}
	}
	return false
}

// Match 判断给定的路径是否与模式相匹配
func Match(pattern string, pathStr string) bool {
	// 快速路径 - 直接比较原始字符串
	if pattern == pathStr {
		return true
	}

	// 清理路径,但仅在必要时执行
	if strings.Contains(pattern, "//") {
		pattern = path.Clean(pattern)
	}
	if strings.Contains(pathStr, "//") {
		pathStr = path.Clean(pathStr)
	}

	// 再次检查清理后的路径是否相等
	if pattern == pathStr {
		return true
	}

	// 如果模式中不包含通配符,则可以直接返回false
	if !strings.ContainsAny(pattern, "*?") {
		return false
	}

	// 检查缓存
	if regex, ok := patternCache.Load(pattern); ok {
		return regex.(*regexp.Regexp).MatchString(pathStr)
	}

	// 将 Ant 风格的模式转换为正则表达式
	regexPattern := convertPatternToRegex(pattern)
	regex, err := regexp.Compile(regexPattern)
	if err != nil {
		return false
	}

	// 缓存编译后的正则表达式
	patternCache.Store(pattern, regex)

	return regex.MatchString(pathStr)
}

// convertPatternToRegex 将 Ant 风格的模式转换为正则表达式
func convertPatternToRegex(pattern string) string {
	// 转义正则表达式特殊字符
	pattern = regexp.QuoteMeta(pattern)

	// 替换 Ant 风格的通配符
	pattern = strings.ReplaceAll(pattern, "\\*\\*", ".*") // ** 匹配任意路径
	pattern = strings.ReplaceAll(pattern, "\\*", "[^/]*") // * 匹配除/外的任意字符
	pattern = strings.ReplaceAll(pattern, "\\?", ".")     // ? 匹配单个字符

	// 确保路径匹配
	if !strings.HasPrefix(pattern, "^") {
		pattern = "^" + pattern
	}
	if !strings.HasSuffix(pattern, "$") {
		pattern = pattern + "$"
	}

	return pattern
}

// ClearCache 清除模式缓存
func ClearCache() {
	patternCache.Range(func(key, _ interface{}) bool {
		patternCache.Delete(key)
		return true
	})
}
