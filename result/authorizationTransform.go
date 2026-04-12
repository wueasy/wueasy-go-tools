package result

// CaptchaVo 验证码返回对象
type AuthorizationTransformVo struct {
	// 代码
	Code string `json:"code"`
	// 过期时间
	Expire string `json:"expire"`
}
