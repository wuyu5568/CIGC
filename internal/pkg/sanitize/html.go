package sanitize

import (
	"bytes"
	"strings"

	"golang.org/x/net/html"
)

var dropEntire = map[string]bool{
	"script":   true,
	"style":    true,
	"iframe":   true,
	"object":   true,
	"embed":    true,
	"form":     true,
	"input":    true,
	"button":   true,
	"textarea": true,
	"select":   true,
	"link":     true,
	"meta":     true,
	"base":     true,
}

var allowed = map[string]bool{
	"p": true, "br": true, "div": true, "span": true, "strong": true, "b": true,
	"em": true, "i": true, "u": true, "s": true, "strike": true, "ul": true, "ol": true,
	"li": true, "h1": true, "h2": true, "h3": true, "h4": true, "h5": true, "h6": true,
	"table": true, "thead": true, "tbody": true, "tr": true, "td": true, "th": true,
	"img": true, "a": true, "blockquote": true, "hr": true, "font": true, "sup": true,
	"sub": true, "figure": true, "figcaption": true, "section": true,
}

// HTML 清洗商品详情富文本：去掉脚本与事件属性，保留常见排版标签。
func HTML(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	doc, err := html.Parse(strings.NewReader(s))
	if err != nil {
		return ""
	}
	body := findElement(doc, "body")
	if body == nil {
		return ""
	}
	sanitizeNode(body)
	var buf bytes.Buffer
	for c := body.FirstChild; c != nil; c = c.NextSibling {
		_ = html.Render(&buf, c)
	}
	return strings.TrimSpace(buf.String())
}

func findElement(n *html.Node, name string) *html.Node {
	if n == nil {
		return nil
	}
	if n.Type == html.ElementNode && strings.EqualFold(n.Data, name) {
		return n
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if got := findElement(c, name); got != nil {
			return got
		}
	}
	return nil
}

func sanitizeNode(n *html.Node) {
	c := n.FirstChild
	for c != nil {
		next := c.NextSibling
		switch c.Type {
		case html.ElementNode:
			tag := strings.ToLower(c.Data)
			if dropEntire[tag] {
				n.RemoveChild(c)
				c = next
				continue
			}
			sanitizeNode(c)
			if !allowed[tag] {
				for gc := c.FirstChild; gc != nil; {
					gnext := gc.NextSibling
					c.RemoveChild(gc)
					n.InsertBefore(gc, c)
					gc = gnext
				}
				n.RemoveChild(c)
				c = next
				continue
			}
			c.Attr = filterAttrs(tag, c.Attr)
		case html.CommentNode:
			n.RemoveChild(c)
		}
		c = next
	}
}

func filterAttrs(tag string, attrs []html.Attribute) []html.Attribute {
	out := make([]html.Attribute, 0, len(attrs))
	for _, a := range attrs {
		key := strings.ToLower(strings.TrimSpace(a.Key))
		if key == "" || strings.HasPrefix(key, "on") {
			continue
		}
		val := strings.TrimSpace(a.Val)
		switch key {
		case "src":
			if tag != "img" || !safeURL(val) {
				continue
			}
		case "href":
			if tag != "a" || !safeURL(val) {
				continue
			}
		case "alt", "title", "width", "height", "class", "colspan", "rowspan", "align":
		case "target":
			if tag != "a" {
				continue
			}
			low := strings.ToLower(val)
			if low != "_blank" && low != "_self" {
				continue
			}
		case "rel":
			if tag != "a" {
				continue
			}
		case "style":
			if !safeStyle(val) {
				continue
			}
		default:
			continue
		}
		out = append(out, html.Attribute{Key: key, Val: val})
	}
	if tag == "a" {
		out = ensureNoopener(out)
	}
	return out
}

func ensureNoopener(attrs []html.Attribute) []html.Attribute {
	blank := false
	relIdx := -1
	for i, a := range attrs {
		if a.Key == "target" && strings.EqualFold(a.Val, "_blank") {
			blank = true
		}
		if a.Key == "rel" {
			relIdx = i
		}
	}
	if !blank {
		return attrs
	}
	if relIdx >= 0 {
		rel := strings.ToLower(attrs[relIdx].Val)
		if !strings.Contains(rel, "noopener") {
			attrs[relIdx].Val = strings.TrimSpace(attrs[relIdx].Val + " noopener")
		}
		if !strings.Contains(strings.ToLower(attrs[relIdx].Val), "noreferrer") {
			attrs[relIdx].Val = strings.TrimSpace(attrs[relIdx].Val + " noreferrer")
		}
		return attrs
	}
	return append(attrs, html.Attribute{Key: "rel", Val: "noopener noreferrer"})
}

func safeURL(u string) bool {
	u = strings.TrimSpace(u)
	if u == "" {
		return false
	}
	low := strings.ToLower(u)
	if strings.HasPrefix(low, "javascript:") || strings.HasPrefix(low, "data:") || strings.HasPrefix(low, "vbscript:") {
		return false
	}
	return strings.HasPrefix(low, "https://") || strings.HasPrefix(low, "http://") ||
		strings.HasPrefix(low, "/") || strings.HasPrefix(low, "//")
}

func safeStyle(s string) bool {
	low := strings.ToLower(s)
	if strings.Contains(low, "javascript") || strings.Contains(low, "expression") || strings.Contains(low, "url(") {
		return false
	}
	return true
}
