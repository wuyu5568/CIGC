package service

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/cigc/app/internal/biz"
)

func readDailyCapTiersReq(r *http.Request) ([]biz.CapTierJSON, error) {
	if r == nil {
		return nil, biz.ErrConfigInvalid
	}
	ct := r.Header.Get("Content-Type")
	if strings.Contains(ct, "json") {
		var body struct {
			Tiers json.RawMessage `json:"tiers"`
		}
		if err := decodeJSON(r, &body); err != nil {
			return nil, biz.ErrConfigInvalid
		}
		return parseDailyCapTiersRaw(body.Tiers)
	}
	if err := parseAdminForm(r); err != nil {
		return nil, biz.ErrConfigInvalid
	}
	return parseDailyCapFormTiers(r.Form)
}

func parseDailyCapTiersRaw(raw json.RawMessage) ([]biz.CapTierJSON, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return nil, biz.ErrConfigInvalid
	}
	var rows []biz.CapTierJSON
	if err := json.Unmarshal(raw, &rows); err != nil {
		return nil, biz.ErrConfigInvalid
	}
	return rows, nil
}

func parseDailyCapFormTiers(form url.Values) ([]biz.CapTierJSON, error) {
	if form == nil {
		return nil, biz.ErrConfigInvalid
	}
	raw := strings.TrimSpace(form.Get("tiers"))
	if strings.HasPrefix(raw, "[") {
		return parseDailyCapTiersRaw(json.RawMessage(raw))
	}
	maxIdx := -1
	for key := range form {
		idx, ok := dailyCapFormIndex(key)
		if !ok {
			continue
		}
		if idx > maxIdx {
			maxIdx = idx
		}
	}
	if maxIdx < 0 {
		return nil, biz.ErrConfigInvalid
	}
	out := make([]biz.CapTierJSON, 0, maxIdx+1)
	for i := 0; i <= maxIdx; i++ {
		prefix := "tiers[" + strconv.Itoa(i) + "]"
		out = append(out, biz.CapTierJSON{
			MaxAmount: form.Get(prefix + "[max_amount]"),
			DailyCap:  form.Get(prefix + "[daily_cap]"),
		})
	}
	return out, nil
}

func dailyCapFormIndex(key string) (int, bool) {
	if !strings.HasPrefix(key, "tiers[") {
		return 0, false
	}
	rest := strings.TrimPrefix(key, "tiers[")
	end := strings.IndexByte(rest, ']')
	if end <= 0 {
		return 0, false
	}
	idx, err := strconv.Atoi(rest[:end])
	if err != nil || idx < 0 {
		return 0, false
	}
	return idx, true
}
