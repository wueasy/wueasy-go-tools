package utils

import (
	"fmt"
	"strconv"
	"strings"
)

// parseBytes 解析字节大小字符串，支持 KB、MB、GB、TB 单位
// 例如: "1024", "1KB", "10MB", "2GB", "1TB"
func ParseFileBytes(sizeStr string) (int64, error) {
	if sizeStr == "" {
		return 0, fmt.Errorf("大小字符串为空")
	}

	// 移除空格并转为小写
	sizeStr = strings.TrimSpace(strings.ToLower(sizeStr))

	// 定义单位映射
	units := map[string]int64{
		"b":  1,
		"kb": 1024,
		"mb": 1024 * 1024,
		"gb": 1024 * 1024 * 1024,
		"tb": 1024 * 1024 * 1024 * 1024,
		"k":  1024,
		"m":  1024 * 1024,
		"g":  1024 * 1024 * 1024,
		"t":  1024 * 1024 * 1024 * 1024,
	}

	// 查找数字部分和单位部分
	var numStr string
	var unitStr string

	for i, char := range sizeStr {
		if char >= '0' && char <= '9' || char == '.' {
			numStr += string(char)
		} else {
			unitStr = sizeStr[i:]
			break
		}
	}

	if numStr == "" {
		return 0, fmt.Errorf("无效的大小格式: %s", sizeStr)
	}

	// 解析数字部分
	num, err := strconv.ParseFloat(numStr, 64)
	if err != nil {
		return 0, fmt.Errorf("无效的数字格式: %s", numStr)
	}

	// 如果没有单位，默认为字节
	if unitStr == "" {
		unitStr = "b"
	}

	// 查找单位
	multiplier, exists := units[unitStr]
	if !exists {
		return 0, fmt.Errorf("不支持的单位: %s", unitStr)
	}

	return int64(num * float64(multiplier)), nil
}

// formatSize 格式化字节大小为人性化字符串
func FormatFileSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.2f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
