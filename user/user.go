package user

import (
	"encoding/base64"
	"encoding/json"

	"github.com/wueasy/wueasy-go-tools/result"

	"github.com/gin-gonic/gin"
)

// GetSessionData 从请求头获取用户会话数据
func GetSessionData(c *gin.Context) (*result.SessionData, error) {
	// 从请求头获取会话数据
	sessionDataStr := c.GetHeader("wueasy-session-data")
	if sessionDataStr == "" {
		return nil, nil
	}

	// Base64解码
	decoded, err := base64.StdEncoding.DecodeString(sessionDataStr)
	if err != nil {
		return nil, err
	}

	// JSON解析
	var sessionData result.SessionData
	err = json.Unmarshal(decoded, &sessionData)
	if err != nil {
		return nil, err
	}

	return &sessionData, nil
}

// GetUserId 获取用户ID
func GetUserId(c *gin.Context) string {
	sessionData, err := GetSessionData(c)
	if err != nil || sessionData == nil {
		return ""
	}
	return sessionData.UserId
}

// GetNickname 获取用户昵称
func GetNickname(c *gin.Context) string {
	sessionData, err := GetSessionData(c)
	if err != nil || sessionData == nil {
		return ""
	}
	return sessionData.Nickname
}

// IsSystemUser 判断是否为超级管理员
func IsSystemUser(c *gin.Context) bool {
	sessionData, err := GetSessionData(c)
	if err != nil || sessionData == nil {
		return false
	}
	return sessionData.IsSystem
}

// GetCustomParameter 获取自定义参数
func GetCustomParameter(c *gin.Context, key string) string {
	sessionData, err := GetSessionData(c)
	if err != nil || sessionData == nil {
		return ""
	}
	return sessionData.CustomParameterMap[key]
}

// GetCustomParameterMap 获取所有自定义参数
func GetCustomParameterMap(c *gin.Context) map[string]string {
	sessionData, err := GetSessionData(c)
	if err != nil || sessionData == nil {
		return make(map[string]string)
	}
	return sessionData.CustomParameterMap
}

// GetRequestId 获取请求ID
func GetRequestId(c *gin.Context) string {
	return c.GetHeader("wueasy-request-id")
}

// GetIp 获取访问IP
func GetIp(c *gin.Context) string {
	return c.GetHeader("wueasy-request-ip")
}
