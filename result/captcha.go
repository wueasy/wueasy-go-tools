package result

// CaptchaVo 验证码返回对象
type CaptchaVo struct {
	// 验证码
	Captcha string `json:"captcha"`
	// 代码或验证码值
	Code string `json:"code"`
}
