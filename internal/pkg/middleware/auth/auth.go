package auth

import (
	"context"
	"net/http"
	"strings"

	"github.com/cigc/app/internal/biz"
	"github.com/go-kratos/kratos/v3/middleware"
	"github.com/go-kratos/kratos/v3/transport"
	khttp "github.com/go-kratos/kratos/v3/transport/http"
	"github.com/golang-jwt/jwt/v5"
)

type ctxKey struct{}
type roleKey struct{}

const (
	roleUser  = "user"
	roleAdmin = "admin"
)

// WithUserID 把用户 ID 写入请求上下文。
func WithUserID(ctx context.Context, userID uint64) context.Context {
	return context.WithValue(ctx, ctxKey{}, userID)
}

// UserIDFromContext 读取 JWT 解析出的用户 ID。
func UserIDFromContext(ctx context.Context) (uint64, bool) {
	v, ok := ctx.Value(ctxKey{}).(uint64)
	return v, ok
}

func withRole(ctx context.Context, role string) context.Context {
	return context.WithValue(ctx, roleKey{}, role)
}

// IsUser 表示当前请求是用户端 JWT（非管理端）。
func IsUser(ctx context.Context) bool {
	v, _ := ctx.Value(roleKey{}).(string)
	return v == roleUser
}

func parseBearerUID(tokenStr string, key []byte) (uint64, bool) {
	token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
		return key, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
	if err != nil || !token.Valid {
		return 0, false
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return 0, false
	}
	if role, _ := claims["role"].(string); role == "admin" {
		return 0, false
	}
	raw, ok := claims["uid"].(float64)
	if !ok {
		return 0, false
	}
	return uint64(raw), true
}

func parseBearerAdmin(tokenStr string, key []byte) bool {
	token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
		return key, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
	if err != nil || !token.Valid {
		return false
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return false
	}
	role, _ := claims["role"].(string)
	return role == "admin"
}

// RequireAdminJWT 校验管理端 Bearer JWT。
func RequireAdminJWT(jwtKey string, next http.HandlerFunc) http.HandlerFunc {
	key := []byte(jwtKey)
	return func(w http.ResponseWriter, r *http.Request) {
		authz := r.Header.Get("Authorization")
		if authz == "" || !strings.HasPrefix(authz, "Bearer ") {
			http.Error(w, `{"message":"unauthorized"}`, http.StatusUnauthorized)
			return
		}
		tokenStr := strings.TrimPrefix(authz, "Bearer ")
		if !parseBearerAdmin(tokenStr, key) {
			http.Error(w, `{"message":"forbidden"}`, http.StatusForbidden)
			return
		}
		next(w, r)
	}
}

// RequireJWT 校验用户端 Bearer JWT，并把 uid 放入 context。
func RequireJWT(jwtKey string, next http.HandlerFunc) http.HandlerFunc {
	key := []byte(jwtKey)
	return func(w http.ResponseWriter, r *http.Request) {
		authz := r.Header.Get("Authorization")
		if authz == "" || !strings.HasPrefix(authz, "Bearer ") {
			http.Error(w, `{"message":"unauthorized"}`, http.StatusUnauthorized)
			return
		}
		tokenStr := strings.TrimPrefix(authz, "Bearer ")
		uid, ok := parseBearerUID(tokenStr, key)
		if !ok {
			http.Error(w, `{"message":"unauthorized"}`, http.StatusUnauthorized)
			return
		}
		next(w, r.WithContext(withRole(WithUserID(r.Context(), uid), roleUser)))
	}
}

func unauthorized(w http.ResponseWriter) {
	http.Error(w, `{"message":"unauthorized"}`, http.StatusUnauthorized)
}

// RequireAdminOrUserGET 管理端 JWT 放行任意方法；用户 JWT 仅放行 GET。缺失或无效 token 返回 401。
func RequireAdminOrUserGET(jwtKey string, next http.HandlerFunc) http.HandlerFunc {
	key := []byte(jwtKey)
	return func(w http.ResponseWriter, r *http.Request) {
		authz := r.Header.Get("Authorization")
		if authz == "" || !strings.HasPrefix(authz, "Bearer ") {
			unauthorized(w)
			return
		}
		tokenStr := strings.TrimPrefix(authz, "Bearer ")
		if parseBearerAdmin(tokenStr, key) {
			next(w, r.WithContext(withRole(r.Context(), roleAdmin)))
			return
		}
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			unauthorized(w)
			return
		}
		uid, ok := parseBearerUID(tokenStr, key)
		if !ok {
			unauthorized(w)
			return
		}
		next(w, r.WithContext(withRole(WithUserID(r.Context(), uid), roleUser)))
	}
}

// JWT 是 Kratos 中间件形态的用户鉴权（给后续生成路由预留）。
func JWT(jwtKey string) middleware.Middleware {
	key := []byte(jwtKey)
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			if tr, ok := transport.FromServerContext(ctx); ok {
				if ht, ok := tr.(*khttp.Transport); ok {
					authz := ht.Request().Header.Get("Authorization")
					if authz == "" || !strings.HasPrefix(authz, "Bearer ") {
						return nil, biz.ErrUnauthorized
					}
					tokenStr := strings.TrimPrefix(authz, "Bearer ")
					uid, ok := parseBearerUID(tokenStr, key)
					if !ok {
						return nil, biz.ErrUnauthorized
					}
					ctx = WithUserID(ctx, uid)
				}
			}
			return handler(ctx, req)
		}
	}
}
