package result

type SessionVo struct {
	// 用户id
	UserId string `json:"userId"`

	// 用户昵称
	Nickname string `json:"nickname"`

	// 是否超级管理员
	IsSystem bool `json:"isSystem"`

	// 头像地址
	AvatarUrl string `json:"avatarUrl"`

	// 临时授权码
	TempAuthorization string `json:"tempAuthorization"`

	// 权限url集合
	AuthorizeUrlList []string `json:"authorizeUrlList"`

	// 权限代码集合
	AuthorizeCodeList []string `json:"authorizeCodeList"`

	// 拓展map
	ExtendedMap map[string]string `json:"extendedMap"`

	// 数据权限Map<业务代码,Set<数据标识>>
	DataAuthorizeMap map[string][]string `json:"dataAuthorizeMap"`

	// 登录成功携带的业务参数，此参数不会保存到session中
	SuccessfulMap map[string]string `json:"successfulMap"`

	// 最后更新时间
	LastUpdateTime int64 `json:"lastUpdateTime"`

	// 更新字段集合，只有更新session时使用
	UpdateFieldList []string `json:"updateFieldList"`

	// 是否需要双因子认证
	TwoFactorAuth bool `json:"twoFactorAuth"`

	DataScopeMap map[string]string `json:"dataScopeMap"` // 数据范围

}

// SessionData 用户会话数据
type SessionData struct {
	UserId             string              `json:"userId"`             // 用户id
	Nickname           string              `json:"nickname"`           // 用户昵称
	IsSystem           bool                `json:"isSystem"`           // 是否超级管理员
	CustomParameterMap map[string]string   `json:"customParameterMap"` // 自定义参数
	DataAuthorizeMap   map[string][]string `json:"dataAuthorizeMap"`   // 数据权限
	DataScopeMap       map[string]string   `json:"dataScopeMap"`       // 数据范围
}
