package biz

// SignatureVerifier 校验钱包 personal_sign。
type SignatureVerifier interface {
	Verify(address, message, signature string) error
}

// TokenIssuer 签发用户/管理 JWT。
type TokenIssuer interface {
	Issue(userID uint64, address string) (string, error)
	IssueAdmin() (string, error)
}

// LoginResult 登录成功后的令牌与用户。
type LoginResult struct {
	Token string
	User  *User
}
