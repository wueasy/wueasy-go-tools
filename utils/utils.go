package utils

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	log2 "github.com/wueasy/wueasy-go-tools/log"

	"github.com/dlclark/regexp2"
	"github.com/google/uuid"
	"gopkg.in/yaml.v3"
)

// EncryptionType 加密类型枚举
type EncryptionType string

const (
	DES3 EncryptionType = "DES3"
	SM4  EncryptionType = "SM4"
)

// EncryptionConfig 加密配置
type EncryptionConfig struct {
	Type    EncryptionType
	Pattern string
	Key     []byte
	Decrypt func([]byte, []byte) ([]byte, error)
}

// DecryptEncryptedContent 通用的加密内容解密处理方法
// @param content 要处理的内容
// @param configs 加密配置列表
// @return 解密后的内容
func DecryptEncryptedContent(ctx context.Context, content string, configs []EncryptionConfig) string {
	contentStr := content

	for _, config := range configs {
		regex := regexp.MustCompile(config.Pattern)
		contentStr = regex.ReplaceAllStringFunc(contentStr, func(match string) string {
			// 提取括号内的加密内容
			matches := regex.FindStringSubmatch(match)
			if len(matches) < 2 {
				if log2.IsErrorEnabled() {
					log2.Ctx(ctx).Errorf("%s加密内容格式错误, match=%s", config.Type, match)
				}
				return match
			}

			encryptedContent := matches[1]

			// Base64解码
			decodeBytes, err := base64.StdEncoding.DecodeString(encryptedContent)
			if err != nil {
				if log2.IsErrorEnabled() {
					log2.Ctx(ctx).Errorf("%s加密内容base64解码失败, data=%s, error=%v", config.Type, encryptedContent, err)
				}
				return encryptedContent
			}

			// 解密
			decryptedData, decryptErr := config.Decrypt(decodeBytes, config.Key)
			if decryptErr != nil {
				if log2.IsErrorEnabled() {
					log2.Ctx(ctx).Errorf("%s解密失败, data=%s, error=%v", config.Type, encryptedContent, decryptErr)
				}
				return encryptedContent
			}

			return string(decryptedData)
		})
	}

	return contentStr
}

// CreateEncryptionConfigs 创建加密配置列表
// @param des3Key DES3密钥
// @param sm4Key SM4密钥
// @return 加密配置列表
func CreateEncryptionConfigs(des3Key string, sm4Key string) []EncryptionConfig {
	configs := make([]EncryptionConfig, 0)

	// 添加DES3配置
	if des3Key != "" {
		configs = append(configs, EncryptionConfig{
			Type:    DES3,
			Pattern: `ENCDES3\(([^)]+)\)`,
			Key:     []byte(des3Key),
			Decrypt: Decrypt3DES,
		})
	}

	// 添加SM4配置
	if sm4Key != "" {
		configs = append(configs, EncryptionConfig{
			Type:    SM4,
			Pattern: `ENCSM4\(([^)]+)\)`,
			Key:     []byte(sm4Key),
			Decrypt: DecryptSM4,
		})
	}

	return configs
}

func GetLocalIPv4Address() (ipv4Address string) {
	// 获取所有网络接口
	interfaces, err := net.InterfaceAddrs()
	if err != nil {
		return "127.0.0.1"
	}

	// 取第一个非lo的网卡IP
	for _, addr := range interfaces {
		// 这个网络地址是IP地址: ipv4, ipv6
		ipNet, isIpNet := addr.(*net.IPNet)
		if isIpNet && !ipNet.IP.IsLoopback() {
			// 跳过IPV6
			if ipNet.IP.To4() != nil && ipNet.IP.IsPrivate() {
				ipv4 := ipNet.IP.String() // 192.168.1.1
				return ipv4
			}
		}
	}
	return "127.0.0.1"
}

/*
版本号转数字
*/
func VersionToNumber(version string) (int64, error) {

	// 分割版本号
	parts := strings.Split(version, ".")

	length := len(parts)
	if length > 4 {
		return 0, errors.New("版本号格式错误")
	}

	// 转换并加权求和
	var result string

	for i := 0; i < 4; i++ {

		var part string
		if length > i {
			part = parts[i]
		} else {
			part = "0"
		}

		if len(part) > 4 {
			return 0, errors.New("版本号格式错误")
		}

		num, err := strconv.ParseInt(part, 10, 64)
		if err != nil {
			return 0, errors.New("版本号格式错误")
		}

		formattedPart := fmt.Sprintf("%04d", num)

		result += formattedPart
	}

	return strconv.ParseInt(result, 10, 64)
}

func GetPageSize(pageSize interface{}) (int64, error) {
	var value int64
	var err error
	switch v := pageSize.(type) {
	case string:
		value, err = strconv.ParseInt(v, 10, 64)
	case int:
		value = int64(v)
	case int64:
		value = v
	case float64:
		value = int64(v)
	case float32:
		value = int64(v)
	default:
		err = errors.New("类型不匹配")
	}
	return value, err
}

func GetFileExt(path string) string {
	ext := filepath.Ext(path)
	switch ext {
	case ".gz", ".bz2", ".xz":
		// 如果是压缩扩展名，检查前一个扩展名
		base := strings.TrimSuffix(path, ext)
		prevExt := filepath.Ext(base)
		if prevExt != "" {
			return prevExt + ext
		}
	}
	return ext
}

func GetUuid() string {
	return strings.ReplaceAll(uuid.New().String(), "-", "")
}

func GetTransformSql(driverName string, sql string) string {
	if driverName == "mysql" || driverName == "" {
		return sql
	}
	return ReplaceQuestionToDollar(sql)
}

// 将sql语句中的?转换成$i
func ReplaceQuestionToDollar(sql string) string {
	var temp = 1
	start := 0
	var i = 0
L:
	for i = start; i < len(sql); i++ {
		if string(sql[i]) == "?" {
			sql = string(sql[:i]) + "$" + strconv.Itoa(temp) + string(sql[i+1:])
			temp++
			start = i + 2
			goto L
		}

		if i == len(sql)-1 {
			return sql
		}
	}
	return sql
}

// expandEnv 替换环境变量，支持默认值
func ExpandEnv(s string) string {
	// 使用正则表达式匹配 ${VAR:default} 格式
	re := regexp.MustCompile(`\${([^:}]+)(?::([^}]+))?}`)
	return re.ReplaceAllStringFunc(s, func(match string) string {
		// 提取变量名和默认值
		parts := re.FindStringSubmatch(match)
		if len(parts) < 2 {
			return match
		}

		varName := parts[1]
		defaultValue := ""
		if len(parts) > 2 {
			defaultValue = parts[2]
		}

		// 获取环境变量值
		value := os.Getenv(varName)
		if value == "" {
			return defaultValue
		}
		return value
	})
}

func ReadConfig(ctx context.Context, filePath string, config any, encDes3Key string, encSm4Key string) {
	// 检查filePath是否是目录
	cleanPath := filepath.Clean(filePath)
	fileInfo, err := os.Stat(cleanPath)
	if err != nil {
		fmt.Println("Error checking file path:", err)
		os.Exit(1)
		return
	}

	// 如果是目录，则添加config.yaml
	if fileInfo.IsDir() {
		cleanPath = filepath.Join(cleanPath, "config.yaml")
	}

	// 打开 YAML 文件
	file, err := os.Open(cleanPath)
	if err != nil {
		fmt.Println("Error opening file:", err)
		os.Exit(1)
		return
	}
	defer file.Close()

	// 读取文件内容
	content, err := io.ReadAll(file)
	if err != nil {
		fmt.Println("Error reading file:", err)
		os.Exit(1)
		return
	}

	// 使用通用方法处理加密内容解密
	configs := CreateEncryptionConfigs(encDes3Key, encSm4Key)
	contentStr := DecryptEncryptedContent(ctx, string(content), configs)

	content = []byte(contentStr)

	// 替换环境变量
	expandedContent := ExpandEnv(string(content))
	content = []byte(expandedContent)

	// 创建解析器
	decoder := yaml.NewDecoder(bytes.NewReader(content))

	// 解析 YAML 数据
	err = decoder.Decode(config)

	if err != nil {
		fmt.Println("Error decoding YAML:", err)
		os.Exit(1)
		return
	}
}

func GetRootPath(envname string) string {
	var rootPath string
	path := os.Getenv(envname)
	if path != "" {
		rootPath = path
	} else {
		cwd, err := os.Getwd()
		if err != nil {
			// 如果获取当前工作目录时发生错误，则打印错误并退出程序
			fmt.Println("Error getting current working directory:", err)
			return ""
		}
		rootPath = cwd
	}
	return rootPath
}

// defaultString 返回默认字符串值
// 如果str为空，则返回defaultValue
// @param str 字符串值
// @param defaultValue 默认值
// @return 如果str为空返回defaultValue,否则返回str
func GetDefaultString(str string, defaultValue string) string {
	if str == "" {
		return defaultValue
	}
	return str
}

// Print 输出内容
// @author: fallsea
// @param content 内容
func ResponseWrite(content interface{}, resp *http.Response) {
	var newPayLoad []byte
	switch v := content.(type) {
	case []byte:
		newPayLoad = v
	default:
		newPayLoad, _ = json.Marshal(content)
	}
	//3.4将数据再次填充到resp中(ioutil.NopCloser()该函数直接将byte数据转换为Body中的read)
	resp.Body = io.NopCloser(bytes.NewBuffer(newPayLoad))
	//3.5重置响应数据长度
	resp.ContentLength = int64(len(newPayLoad))
	resp.Header.Set("Content-Length", fmt.Sprint(len(newPayLoad)))
	resp.Header.Set("Content-Type", "application/json")
}

func ResponseWrite2(content interface{}, w http.ResponseWriter) {
	var newPayLoad []byte
	switch v := content.(type) {
	case []byte:
		newPayLoad = v
	case string:
		newPayLoad = []byte(v)
	default:
		newPayLoad, _ = json.Marshal(content)
	}
	w.Header().Set("Content-Length", fmt.Sprint(len(newPayLoad)))
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(newPayLoad)
}

func ResponseWrite3(content interface{}, w http.ResponseWriter, statusCode int) {
	var newPayLoad []byte
	switch v := content.(type) {
	case []byte:
		newPayLoad = v
	case string:
		newPayLoad = []byte(v)
	default:
		newPayLoad, _ = json.Marshal(content)
	}
	w.Header().Set("Content-Length", fmt.Sprint(len(newPayLoad)))
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	w.Write(newPayLoad)
}

// GetIpAddr 获取IP地址
// @author: fallsea
// @param r *http.Request 请求对象
// @return string IP地址
func GetIpAddr(r *http.Request) string {
	// 尝试从请求头获取IP
	ip := r.Header.Get("X-Forwarded-For")

	if ip != "" && strings.ToLower(ip) != "unknown" {
		// 多次反向代理后会有多个IP值，第一个IP才是真实IP
		if idx := strings.Index(ip, ","); idx != -1 {
			ip = ip[:idx]
		}
		return ip
	}

	// 尝试从Proxy-Client-IP获取
	ip = r.Header.Get("Proxy-Client-IP")
	if ip != "" && strings.ToLower(ip) != "unknown" {
		return ip
	}

	// 尝试从WL-Proxy-Client-IP获取
	ip = r.Header.Get("WL-Proxy-Client-IP")
	if ip != "" && strings.ToLower(ip) != "unknown" {
		return ip
	}

	// 如果都没有则使用RemoteAddr
	ip = r.RemoteAddr
	if idx := strings.LastIndex(ip, ":"); idx != -1 {
		ip = ip[:idx]
	}

	return ip
}

// GetSignMap 获取签名map
// @author: fallsea
// @param bodyString 请求体字符串
// @param mediaType 媒体类型
// @param signType 签名类型
// @return map[string]string 签名map
func GetSignMap(bodyString string, mediaType string, signType string) map[string]string {
	// 创建有序map
	signMap := make(map[string]string)

	if bodyString != "" {
		var jsonNode map[string]interface{}
		err := json.Unmarshal([]byte(bodyString), &jsonNode)
		if err == nil {
			if signType == "all" {
				// 递归处理所有JSON节点
				getSignAllMap("", jsonNode, signMap)
			} else {
				// 只处理第一层非数组和对象的字段
				for key, value := range jsonNode {
					switch v := value.(type) {
					case string:
						signMap[key] = v
					case float64:
						signMap[key] = strconv.FormatFloat(v, 'f', -1, 64)
					case float32:
						signMap[key] = strconv.FormatFloat(float64(v), 'f', -1, 32)
					case int:
						signMap[key] = strconv.Itoa(v)
					case int8:
						signMap[key] = strconv.FormatInt(int64(v), 10)
					case int16:
						signMap[key] = strconv.FormatInt(int64(v), 10)
					case int32:
						signMap[key] = strconv.FormatInt(int64(v), 10)
					case int64:
						signMap[key] = strconv.FormatInt(v, 10)
					case uint:
						signMap[key] = strconv.FormatUint(uint64(v), 10)
					case uint8:
						signMap[key] = strconv.FormatUint(uint64(v), 10)
					case uint16:
						signMap[key] = strconv.FormatUint(uint64(v), 10)
					case uint32:
						signMap[key] = strconv.FormatUint(uint64(v), 10)
					case uint64:
						signMap[key] = strconv.FormatUint(v, 10)
					case bool:
						signMap[key] = strconv.FormatBool(v)
					case nil:
						signMap[key] = ""
					default:
						signMap[key] = fmt.Sprintf("%v", v)
					}
				}
			}
		}
	}

	return signMap
}

// getSignAllMap 递归处理所有JSON节点生成签名map
// @param path 当前节点路径
// @param node 当前节点
// @param signMap 签名map
func getSignAllMap(path string, node interface{}, signMap map[string]string) {
	switch v := node.(type) {
	case map[string]interface{}:
		// 处理对象
		for key, value := range v {
			newPath := key
			if path != "" {
				newPath = path + "." + key
			}
			if value == nil {
				signMap[newPath] = ""
			} else {
				getSignAllMap(newPath, value, signMap)
			}
		}
	case []interface{}:
		// 处理数组
		for i, item := range v {
			newPath := fmt.Sprintf("%s[%d]", path, i)
			if item == nil {
				signMap[newPath] = ""
			} else {
				getSignAllMap(newPath, item, signMap)
			}
		}
	default:
		// 处理基本类型
		var strValue string
		switch val := v.(type) {
		case string:
			strValue = val
		case float64:
			strValue = strconv.FormatFloat(val, 'f', -1, 64)
		case float32:
			strValue = strconv.FormatFloat(float64(val), 'f', -1, 32)
		case int:
			strValue = strconv.Itoa(val)
		case int8:
			strValue = strconv.FormatInt(int64(val), 10)
		case int16:
			strValue = strconv.FormatInt(int64(val), 10)
		case int32:
			strValue = strconv.FormatInt(int64(val), 10)
		case int64:
			strValue = strconv.FormatInt(val, 10)
		case uint:
			strValue = strconv.FormatUint(uint64(val), 10)
		case uint8:
			strValue = strconv.FormatUint(uint64(val), 10)
		case uint16:
			strValue = strconv.FormatUint(uint64(val), 10)
		case uint32:
			strValue = strconv.FormatUint(uint64(val), 10)
		case uint64:
			strValue = strconv.FormatUint(val, 10)
		case bool:
			strValue = strconv.FormatBool(val)
		case nil:
			strValue = ""
		default:
			if bytes, err := json.Marshal(val); err == nil {
				strValue = string(bytes)
			}
		}
		signMap[path] = strValue
	}
}

// GetBody 从请求中获取body字符串
// 返回:
// - bodyString: body字符串
// - err: 错误信息
func GetBody(r *http.Request) (bodyString string, err error) {
	contentType := r.Header.Get("Content-Type")

	// 如果是JSON格式请求
	if strings.Contains(contentType, "application/json") {
		body := r.Body
		// 读取请求体数据
		if body != nil {
			var bodyBytes []byte
			bodyBytes, err = io.ReadAll(body)
			if err != nil {
				return "", fmt.Errorf("读取请求体失败: %v", err)
			}
			// 重新设置请求体,因为ReadAll会清空body
			r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
			// 将body转换为string类型
			bodyString = string(bodyBytes)
		}
	} else if strings.Contains(contentType, "application/x-www-form-urlencoded") {
		// 如果是表单格式请求
		body := r.Body
		if body != nil {
			bodyBytes, err := io.ReadAll(body)
			if err != nil {
				return "", fmt.Errorf("读取请求体失败: %v", err)
			}
			// 重新设置请求体,因为ReadAll会清空body
			r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

			// 解析表单数据并转换为JSON格式
			values, parseErr := url.ParseQuery(string(bodyBytes))
			if parseErr != nil {
				return "", fmt.Errorf("解析表单数据失败: %v", parseErr)
			}

			// 将表单数据转换为map
			formMap := make(map[string]interface{})
			for key, valueList := range values {
				if len(valueList) == 1 {
					formMap[key] = valueList[0]
				} else {
					formMap[key] = strings.Join(valueList, ",")
				}
			}

			// 转换为JSON字符串
			jsonBytes, jsonErr := json.Marshal(formMap)
			if jsonErr != nil {
				return "", fmt.Errorf("转换为JSON失败: %v", jsonErr)
			}
			bodyString = string(jsonBytes)
		}
	}
	return bodyString, nil
}

// MatchPattern 执行正则表达式匹配
// 参数:
// - pattern: 正则表达式模式字符串
// - value: 需要匹配的字符串
// 返回:
// - bool: 是否匹配成功
func MatchPattern(pattern string, value string) bool {
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

// Base64Encode 对字节数组进行Base64编码
// 参数:
// - data: 需要编码的字节数组
// 返回:
// - string: Base64编码后的字符串
func Base64Encode(data []byte) string {
	return base64.StdEncoding.EncodeToString(data)
}

// Base64EncodeString 对字符串进行Base64编码
// 参数:
// - str: 需要编码的字符串
// 返回:
// - string: Base64编码后的字符串
func Base64EncodeString(str string) string {
	return base64.StdEncoding.EncodeToString([]byte(str))
}

// Base64Decode 对Base64编码的字符串进行解码
// 参数:
// - encoded: Base64编码的字符串
// 返回:
// - []byte: 解码后的字节数组
// - error: 解码错误信息
func Base64Decode(encoded string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(encoded)
}

// Base64DecodeString 对Base64编码的字符串进行解码并返回字符串
// 参数:
// - encoded: Base64编码的字符串
// 返回:
// - string: 解码后的字符串
// - error: 解码错误信息
func Base64DecodeString(encoded string) (string, error) {
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", err
	}
	return string(decoded), nil
}

// Base64URLEncode 对字节数组进行Base64 URL安全编码
// 参数:
// - data: 需要编码的字节数组
// 返回:
// - string: Base64 URL安全编码后的字符串
func Base64URLEncode(data []byte) string {
	return base64.URLEncoding.EncodeToString(data)
}

// Base64URLEncodeString 对字符串进行Base64 URL安全编码
// 参数:
// - str: 需要编码的字符串
// 返回:
// - string: Base64 URL安全编码后的字符串
func Base64URLEncodeString(str string) string {
	return base64.URLEncoding.EncodeToString([]byte(str))
}

// Base64URLDecode 对Base64 URL安全编码的字符串进行解码
// 参数:
// - encoded: Base64 URL安全编码的字符串
// 返回:
// - []byte: 解码后的字节数组
// - error: 解码错误信息
func Base64URLDecode(encoded string) ([]byte, error) {
	return base64.URLEncoding.DecodeString(encoded)
}

// Base64URLDecodeString 对Base64 URL安全编码的字符串进行解码并返回字符串
// 参数:
// - encoded: Base64 URL安全编码的字符串
// 返回:
// - string: 解码后的字符串
// - error: 解码错误信息
func Base64URLDecodeString(encoded string) (string, error) {
	decoded, err := base64.URLEncoding.DecodeString(encoded)
	if err != nil {
		return "", err
	}
	return string(decoded), nil
}

const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

// 生成指定长度的随机字符串
func GenerateRandomString(length int) string {
	rand.Seed(time.Now().UnixNano()) // 初始化随机种子
	result := make([]byte, length)
	for i := range result {
		result[i] = charset[rand.Intn(len(charset))]
	}
	return string(result)
}

// ObfuscateKey 对密钥进行混淆处理
// 通过hash方式动态混淆密钥中的几个字符，相同输入返回相同输出
// 参数:
// - key: 需要混淆的密钥字符串
// 返回:
// - string: 混淆后的密钥字符串
func ObfuscateKey(key string) string {
	if len(key) == 0 {
		return key
	}

	// 多层hash混淆
	h1 := sha256.Sum256([]byte(key))
	h2 := sha256.Sum256(h1[:])
	h3 := sha256.Sum256(append(h1[:], h2[:]...))

	// 动态字符集构建
	dynCharset := make([]byte, 0, len(charset)*2)
	for i, c := range []byte(charset) {
		dynCharset = append(dynCharset, c^byte(h3[i%len(h3)]))
	}
	dynCharset = append(dynCharset, charset...)

	// 多重变换
	kb := []byte(key)
	kl := len(kb)

	if kl < 3 {
		return key
	}

	// 动态混淆参数
	moc := (kl >> 1) + (kl >> 2) // 位运算替代除法
	if moc < 1 {
		moc = 1
	}
	if moc > 7 {
		moc = 7
	}

	// 多轮混淆
	for round := 0; round < 3; round++ {
		hashSeed := append(h1[:], byte(round))
		roundHash := sha256.Sum256(hashSeed)

		for i := 0; i < moc && i < len(roundHash)-1; i++ {
			// 位运算混淆位置计算
			pos := int(roundHash[i]^roundHash[i+1]) % kl

			// 动态字符选择
			ci := int(roundHash[(i+2)%len(roundHash)]^roundHash[(i+3)%len(roundHash)]) % len(dynCharset)

			// XOR混淆
			kb[pos] = kb[pos] ^ dynCharset[ci] ^ byte(round+1)
		}
	}

	// 最终字符集映射
	for i := range kb {
		if kb[i] < 32 || kb[i] > 126 {
			kb[i] = charset[int(kb[i])%len(charset)]
		}
	}

	return string(kb)
}

// XOR加密函数
func XorEncrypt(input, key string) string {
	if len(input) == 0 || len(key) == 0 {
		return input
	}

	inputBytes := []byte(input)
	keyBytes := []byte(key)
	keyLen := len(keyBytes)
	output := make([]byte, len(inputBytes))

	for i, b := range inputBytes {
		output[i] = b ^ keyBytes[i%keyLen]
	}

	return hex.EncodeToString(output)
}

// XOR解密函数
func XorDecrypt(hexInput, key string) string {
	// 将十六进制字符串转换为字节数组
	input, err := hex.DecodeString(hexInput)
	if err != nil {
		return ""
	}

	output := make([]byte, len(input))
	for i := range input {
		output[i] = input[i] ^ key[i%len(key)]
	}
	return string(output)
}

var regexCache = sync.Map{}

/**
 * RegexMatch 检查字符串是否匹配正则表达式
 * @param pattern 正则表达式模式
 * @param input 待匹配的字符串
 * @return 是否匹配
 */
func RegexMatch(pattern string, input string) (bool, error) {

	if pattern == "" {
		return false, nil
	}

	// 转义正则表达式特殊字符
	pattern2 := pattern

	// 确保路径匹配
	if !strings.HasPrefix(pattern2, "^") {
		pattern2 = "^" + pattern2
	}
	if !strings.HasSuffix(pattern2, "$") {
		pattern2 = pattern2 + "$"
	}

	// 检查缓存
	if regex, ok := regexCache.Load(pattern2); ok {
		return regex.(*regexp2.Regexp).MatchString(input)
	}

	// 编译正则表达式
	regex, err := regexp2.Compile(pattern2, regexp2.None)
	if err != nil {
		return false, err
	}

	// 缓存编译后的正则表达式
	regexCache.Store(pattern2, regex)

	return regex.MatchString(input)
}

// LogSql 输出SQL调试日志（仅在showSql=true且日志级别为debug时输出）
func LogSql(ctx context.Context, showSql bool, sql string, args ...interface{}) {
	if !showSql {
		return
	}
	if !log2.IsDebugEnabled() {
		return
	}
	if len(args) == 0 {
		log2.Ctx(ctx).Debugf("[SQL] %s", sql)
	} else {
		log2.Ctx(ctx).Debugf("[SQL] %s | args=%v", sql, args)
	}
}
