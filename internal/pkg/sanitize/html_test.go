package sanitize

import "strings"
import "testing"

func TestHTMLStripsScriptAndEvents(t *testing.T) {
	got := HTML(`<p onclick="alert(1)">hello</p><script>alert(1)</script><img src="javascript:alert(1)">`)
	if strings.Contains(strings.ToLower(got), "script") || strings.Contains(strings.ToLower(got), "onclick") || strings.Contains(strings.ToLower(got), "javascript") {
		t.Fatalf("got %q", got)
	}
	if !strings.Contains(got, "hello") {
		t.Fatalf("keep text %q", got)
	}
}

func TestHTMLKeepsImgAndRelativeSrc(t *testing.T) {
	got := HTML(`<p>详情</p><img src="/uploads/web3_goods/a.png" alt="cover">`)
	if !strings.Contains(got, `/uploads/web3_goods/a.png`) {
		t.Fatalf("keep img %q", got)
	}
}
