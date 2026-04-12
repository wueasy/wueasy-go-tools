package result

// LoginVo 登录返回对象
type LoginVo struct {
	// 授权码
	Authorization string `json:"authorization"`

	// 临时授权码
	TempAuthorization string `json:"tempAuthorization"`

	// 拓展map
	SuccessfulMap map[string]string `json:"successfulMap"`

	// 是否需要双因子认证
	TwoFactorAuth bool `json:"twoFactorAuth"`
}
