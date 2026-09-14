package upload

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	MaxImageBytes = 5 << 20
	PublicPrefix  = "/uploads"
)

// DetectImageExt 仅接受 jpg/jpeg/png/webp，优先看文件头。
func DetectImageExt(filename string, data []byte) (string, bool) {
	kind := sniffImage(data)
	if kind == "" {
		return "", false
	}
	name := strings.ToLower(strings.TrimSpace(filename))
	switch kind {
	case "jpeg":
		if strings.HasSuffix(name, ".jpg") {
			return ".jpg", true
		}
		return ".jpeg", true
	case "png":
		return ".png", true
	case "webp":
		return ".webp", true
	default:
		return "", false
	}
}

func sniffImage(data []byte) string {
	if len(data) >= 3 && data[0] == 0xff && data[1] == 0xd8 && data[2] == 0xff {
		return "jpeg"
	}
	if len(data) >= 8 && data[0] == 0x89 && data[1] == 0x50 && data[2] == 0x4e && data[3] == 0x47 &&
		data[4] == 0x0d && data[5] == 0x0a && data[6] == 0x1a && data[7] == 0x0a {
		return "png"
	}
	if len(data) >= 12 && string(data[0:4]) == "RIFF" && string(data[8:12]) == "WEBP" {
		return "webp"
	}
	return ""
}

// Save 把图片写到 root/category/YYYYMMDD/random.ext，返回以 /uploads 开头的相对路径。
func Save(root, category, ext string, data []byte) (string, error) {
	root = strings.TrimSpace(root)
	if root == "" {
		return "", fmt.Errorf("upload dir empty")
	}
	category = strings.Trim(strings.TrimSpace(category), "/")
	if category == "" {
		category = "misc"
	}
	ext = strings.ToLower(strings.TrimSpace(ext))
	if ext == "" || !strings.HasPrefix(ext, ".") {
		return "", fmt.Errorf("bad ext")
	}
	if len(data) == 0 {
		return "", fmt.Errorf("empty file")
	}
	var rnd [16]byte
	if _, err := rand.Read(rnd[:]); err != nil {
		return "", err
	}
	day := time.Now().Format("20060102")
	name := hex.EncodeToString(rnd[:]) + ext
	dir := filepath.Join(root, category, day)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	full := filepath.Join(dir, name)
	if err := os.WriteFile(full, data, 0o644); err != nil {
		return "", err
	}
	rel := PublicPrefix + "/" + category + "/" + day + "/" + name
	return rel, nil
}

// PublicBaseURL 由反代头拼出站点根，例如 https://ispaygijdysxt.com。
func PublicBaseURL(r *http.Request) string {
	if r == nil {
		return ""
	}
	proto := strings.TrimSpace(r.Header.Get("X-Forwarded-Proto"))
	if proto == "" {
		if r.TLS != nil {
			proto = "https"
		} else {
			proto = "http"
		}
	}
	if i := strings.IndexByte(proto, ','); i >= 0 {
		proto = strings.TrimSpace(proto[:i])
	}
	host := strings.TrimSpace(r.Header.Get("X-Forwarded-Host"))
	if host == "" {
		host = strings.TrimSpace(r.Host)
	}
	if i := strings.IndexByte(host, ','); i >= 0 {
		host = strings.TrimSpace(host[:i])
	}
	host = strings.TrimSpace(host)
	if host == "" {
		return ""
	}
	return strings.TrimRight(proto, ":/") + "://" + host
}

// NormalizeStored 把完整 URL 收成 /uploads/...；外链原样保留。
func NormalizeStored(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if strings.HasPrefix(raw, PublicPrefix+"/") {
		return raw
	}
	u, err := url.Parse(raw)
	if err == nil && u.Path != "" && strings.HasPrefix(u.Path, PublicPrefix+"/") {
		return u.Path
	}
	return raw
}

// AbsoluteURL 列表/详情返回浏览器可打开的完整地址。
func AbsoluteURL(base, stored string) string {
	stored = strings.TrimSpace(stored)
	if stored == "" {
		return ""
	}
	if strings.HasPrefix(stored, "http://") || strings.HasPrefix(stored, "https://") {
		return stored
	}
	base = strings.TrimRight(strings.TrimSpace(base), "/")
	if strings.HasPrefix(stored, "/") {
		if base == "" {
			return stored
		}
		return base + stored
	}
	if base == "" {
		return stored
	}
	return base + "/" + stored
}
