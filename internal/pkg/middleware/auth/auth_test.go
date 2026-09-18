package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/cigc/app/internal/conf"
)

func TestRequireAdminOrUserGET(t *testing.T) {
	issuer := NewTokenIssuer(&conf.Auth{JWTKey: "test-key"})
	adminTok, err := issuer.IssueAdmin()
	if err != nil {
		t.Fatal(err)
	}
	userTok, err := issuer.Issue(7, "0xabc")
	if err != nil {
		t.Fatal(err)
	}

	type seen struct {
		ok     bool
		isUser bool
		uid    uint64
	}
	run := func(method, token string) (int, seen) {
		var got seen
		h := RequireAdminOrUserGET("test-key", func(w http.ResponseWriter, r *http.Request) {
			got.ok = true
			got.isUser = IsUser(r.Context())
			got.uid, _ = UserIDFromContext(r.Context())
			w.WriteHeader(http.StatusOK)
		})
		req := httptest.NewRequest(method, "/api/admin/web3_goods", nil)
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		return rec.Code, got
	}

	if code, _ := run(http.MethodGet, ""); code != http.StatusUnauthorized {
		t.Fatalf("no token: %d", code)
	}
	if code, _ := run(http.MethodGet, "bad"); code != http.StatusUnauthorized {
		t.Fatalf("bad token: %d", code)
	}
	code, got := run(http.MethodGet, adminTok)
	if code != http.StatusOK || !got.ok || got.isUser || got.uid != 0 {
		t.Fatalf("admin GET: code=%d %+v", code, got)
	}
	code, got = run(http.MethodPost, adminTok)
	if code != http.StatusOK || !got.ok || got.isUser {
		t.Fatalf("admin POST: code=%d %+v", code, got)
	}
	code, got = run(http.MethodGet, userTok)
	if code != http.StatusOK || !got.ok || !got.isUser || got.uid != 7 {
		t.Fatalf("user GET: code=%d %+v", code, got)
	}
	if code, got := run(http.MethodPost, userTok); code != http.StatusUnauthorized || got.ok {
		t.Fatalf("user POST: code=%d %+v", code, got)
	}
}

func TestRequireAdminJWTStillRejectsUser(t *testing.T) {
	issuer := NewTokenIssuer(&conf.Auth{JWTKey: "test-key"})
	userTok, err := issuer.Issue(7, "0xabc")
	if err != nil {
		t.Fatal(err)
	}
	h := RequireAdminJWT("test-key", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	req := httptest.NewRequest(http.MethodPost, "/api/admin/web3_goods_create", nil)
	req.Header.Set("Authorization", "Bearer "+userTok)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("user should not pass admin write: %d", rec.Code)
	}
	bad := httptest.NewRequest(http.MethodGet, "/api/admin/all", nil)
	bad.Header.Set("Authorization", "Bearer stale-token")
	badRec := httptest.NewRecorder()
	h.ServeHTTP(badRec, bad)
	if badRec.Code != http.StatusUnauthorized {
		t.Fatalf("stale token should be 401: %d", badRec.Code)
	}
}

func TestSigningKeyMatchesIssuerWhenEmpty(t *testing.T) {
	issuer := NewTokenIssuer(&conf.Auth{})
	adminTok, err := issuer.IssueAdmin()
	if err != nil {
		t.Fatal(err)
	}
	h := RequireAdminJWT("", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	req := httptest.NewRequest(http.MethodPost, "/api/admin/web3_goods_delete", nil)
	req.Header.Set("Authorization", "Bearer "+adminTok)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("empty jwt key should verify default-signed admin token: %d", rec.Code)
	}

	opt := httptest.NewRequest(http.MethodOptions, "/api/admin/web3_goods_delete", nil)
	optRec := httptest.NewRecorder()
	h.ServeHTTP(optRec, opt)
	if optRec.Code != http.StatusNoContent {
		t.Fatalf("OPTIONS: %d", optRec.Code)
	}
}
