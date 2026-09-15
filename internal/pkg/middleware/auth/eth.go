package auth

import (
	"strconv"
	"strings"
	"time"

	"github.com/cigc/app/internal/biz"
	"github.com/cigc/app/internal/conf"
	"github.com/cigc/app/internal/pkg/ethsign"
	"github.com/golang-jwt/jwt/v5"
)

// EthVerifier 校验钱包 personal_sign。
type EthVerifier struct{}

func NewSignatureVerifier() biz.SignatureVerifier { return &EthVerifier{} }

func (v *EthVerifier) Verify(address, message, signature string) error {
	address = strings.TrimSpace(address)
	signature = strings.TrimSpace(signature)
	if address == "" || signature == "" || message == "" {
		return biz.ErrInvalidSignature
	}
	sig, err := ethsign.DecodeSignature(signature)
	if err != nil {
		return biz.ErrInvalidSignature
	}
	want, ok := ethsign.NormalizeAddress(address)
	if !ok {
		return biz.ErrInvalidSignature
	}
	candidates := []string{message}
	lower := strings.ToLower(message)
	if lower != message {
		candidates = append(candidates, lower)
	}
	if n, ok := ethsign.NormalizeAddress(message); ok {
		candidates = append(candidates, n, strings.ToLower(message), message)
		if c := ethsign.ChecksumAddress(n); c != "" {
			candidates = append(candidates, c)
		}
	}
	if c := ethsign.ChecksumAddress(want); c != "" {
		candidates = append(candidates, c)
	}
	seen := map[string]struct{}{}
	for _, msg := range candidates {
		if _, dup := seen[msg]; dup {
			continue
		}
		seen[msg] = struct{}{}
		got, err := ethsign.RecoverAddress([]byte(msg), sig)
		if err != nil {
			continue
		}
		if strings.EqualFold(got, want) {
			return nil
		}
	}
	return biz.ErrInvalidSignature
}

// JWTIssuer 签发 HS256 JWT。
type JWTIssuer struct {
	key []byte
	ttl time.Duration
}

func NewTokenIssuer(authConf *conf.Auth) biz.TokenIssuer {
	key := signingKey("")
	ttl := 24 * time.Hour
	if authConf != nil {
		key = signingKey(authConf.JWTKey)
	}
	return &JWTIssuer{key: key, ttl: ttl}
}

func (i *JWTIssuer) Issue(userID uint64, address string) (string, error) {
	now := time.Now()
	claims := jwt.MapClaims{
		"uid":  userID,
		"addr": address,
		"exp":  now.Add(i.ttl).Unix(),
		"iat":  now.Unix(),
		"sub":  strconv.FormatUint(userID, 10),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(i.key)
}

func (i *JWTIssuer) IssueAdmin() (string, error) {
	now := time.Now()
	claims := jwt.MapClaims{
		"role": "admin",
		"exp":  now.Add(i.ttl).Unix(),
		"iat":  now.Unix(),
		"sub":  "admin",
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(i.key)
}

var _ biz.SignatureVerifier = (*EthVerifier)(nil)
var _ biz.TokenIssuer = (*JWTIssuer)(nil)
