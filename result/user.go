package result

// UserVo 用户信息
type UserVo struct {
	// 用户id
	UserId string `json:"userId"`

	// 用户昵称
	Nickname string `json:"nickname"`

	// 头像地址
	AvatarUrl string `json:"avatarUrl"`

	// 权限代码集合
	AuthorizeCodeList []string `json:"authorizeCodeList"`

	// 拓展map
	ExtendedMap map[string]string `json:"extendedMap"`
}
