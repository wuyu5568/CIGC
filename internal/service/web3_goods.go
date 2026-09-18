package service

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/cigc/app/internal/biz"
	"github.com/cigc/app/internal/pkg/middleware/auth"
	"github.com/cigc/app/internal/pkg/sanitize"
	"github.com/cigc/app/internal/pkg/upload"
	"github.com/shopspring/decimal"
)

func (s *AppService) uploadDir() string {
	if s != nil && s.app != nil {
		if d := strings.TrimSpace(s.app.UploadDir); d != "" {
			return d
		}
	}
	if d := strings.TrimSpace(os.Getenv("CIGC_UPLOAD_DIR")); d != "" {
		return d
	}
	return "/data/uploads"
}

func (s *AppService) web3GoodsJSON(r *http.Request, p *biz.Package, withDetail bool) map[string]any {
	if p == nil {
		return map[string]any{}
	}
	days := p.ReleaseDays
	if !biz.ValidReleaseDays(days) {
		days = biz.ReleaseDays300
	}
	onSale := 0
	if p.Enabled {
		onSale = 1
	}
	locale := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("lang")))
	if locale != "en" {
		locale = "zh"
	}
	zh := biz.PackageContent{Title: p.Title, GoodsDesc: p.GoodsDesc, Image: p.Image, Detail: p.Detail}
	if content, ok := p.Contents["zh"]; ok {
		zh = content
	}
	selected := zh
	if locale == "en" {
		if content, ok := p.Contents["en"]; ok {
			if strings.TrimSpace(content.Title) != "" {
				selected.Title = content.Title
			}
			if strings.TrimSpace(content.GoodsDesc) != "" {
				selected.GoodsDesc = content.GoodsDesc
			}
			if strings.TrimSpace(content.Image) != "" {
				selected.Image = content.Image
			}
			if strings.TrimSpace(content.Detail) != "" {
				selected.Detail = content.Detail
			}
		}
	}
	name := strings.TrimSpace(selected.Title)
	if name == "" {
		name = selected.GoodsDesc
	}
	out := map[string]any{
		"id":         p.ID,
		"name":       name,
		"desc":       selected.GoodsDesc,
		"amount":     decStr(p.Amount),
		"daily_cap":  decStr(biz.CapForAmount(p.Amount)),
		"days":       days,
		"sort":       p.SortOrder,
		"sort_order": p.SortOrder,
		"on_sale":    onSale,
		"image":      upload.AbsoluteURL(upload.PublicBaseURL(r), selected.Image),
		"has_detail": strings.TrimSpace(selected.Detail) != "",
	}
	if withDetail {
		out["detail"] = sanitize.HTML(selected.Detail)
	}
	if !auth.IsUser(r.Context()) {
		contents := map[string]any{}
		englishComplete := false
		for _, code := range []string{"zh", "en"} {
			content, ok := p.Contents[code]
			if !ok && code == "zh" {
				content = zh
			}
			entry := map[string]any{
				"title": content.Title,
				"desc":  content.GoodsDesc,
				"image": upload.AbsoluteURL(upload.PublicBaseURL(r), content.Image),
			}
			if withDetail {
				entry["detail"] = sanitize.HTML(content.Detail)
			}
			if code == "en" {
				englishComplete = strings.TrimSpace(content.Title) != "" &&
					strings.TrimSpace(content.GoodsDesc) != "" &&
					strings.TrimSpace(content.Image) != "" &&
					strings.TrimSpace(content.Detail) != ""
			}
			contents[code] = entry
		}
		out["contents"] = contents
		out["english_complete"] = englishComplete
	}
	skus := p.SKUs
	if auth.IsUser(r.Context()) {
		filtered := make([]biz.PackageSKU, 0, len(skus))
		for _, sku := range skus {
			if sku.Enabled {
				filtered = append(filtered, sku)
			}
		}
		skus = filtered
	}
	out["skus"] = skuJSONList(r, skus, locale)
	if auth.IsUser(r.Context()) && len(skus) > 0 {
		minAmt, maxAmt := skus[0].Amount, skus[0].Amount
		for _, sku := range skus[1:] {
			if sku.Amount.LessThan(minAmt) {
				minAmt = sku.Amount
			}
			if sku.Amount.GreaterThan(maxAmt) {
				maxAmt = sku.Amount
			}
		}
		out["amount"] = decStr(minAmt)
		out["daily_cap"] = decStr(biz.CapForAmount(minAmt))
		if !minAmt.Equal(maxAmt) {
			out["amount_max"] = decStr(maxAmt)
		}
	}
	return out
}

func skuJSONList(r *http.Request, skus []biz.PackageSKU, locale string) []map[string]any {
	out := make([]map[string]any, 0, len(skus))
	for _, sku := range skus {
		name := strings.TrimSpace(sku.Name)
		if locale == "en" && strings.TrimSpace(sku.NameEn) != "" {
			name = strings.TrimSpace(sku.NameEn)
		}
		on := 0
		if sku.Enabled {
			on = 1
		}
		out = append(out, map[string]any{
			"id":         sku.ID,
			"name":       name,
			"name_zh":    sku.Name,
			"name_en":    sku.NameEn,
			"amount":     decStr(sku.Amount),
			"image":      upload.AbsoluteURL(upload.PublicBaseURL(r), sku.Image),
			"enabled":    on,
			"sort_order": sku.SortOrder,
		})
	}
	return out
}

func writeWeb3GoodsBiz(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, biz.ErrPackageNotFound):
		writeJSON(w, http.StatusOK, map[string]string{"status": "商品不存在"})
	case errors.Is(err, biz.ErrPackageAmountTaken):
		writeJSON(w, http.StatusOK, map[string]string{"status": "金额已存在"})
	case errors.Is(err, biz.ErrPackageTitle):
		writeJSON(w, http.StatusOK, map[string]string{"status": "请填写名称"})
	case errors.Is(err, biz.ErrPackageDesc):
		writeJSON(w, http.StatusOK, map[string]string{"status": "请填写描述"})
	case errors.Is(err, biz.ErrPackageDetail):
		writeJSON(w, http.StatusOK, map[string]string{"status": "详情过长"})
	case errors.Is(err, biz.ErrPackageInUse):
		writeJSON(w, http.StatusOK, map[string]string{"status": "已有订单不能删除，请先下架"})
	case errors.Is(err, biz.ErrInvalidAmount):
		writeJSON(w, http.StatusOK, map[string]string{"status": "金额错误"})
	case errors.Is(err, biz.ErrInvalidReleaseDays):
		writeJSON(w, http.StatusOK, map[string]string{"status": "请选择释放天数"})
	case errors.Is(err, biz.ErrSKUInvalid):
		writeJSON(w, http.StatusOK, map[string]string{"status": "SKU 参数错误"})
	default:
		writeJSON(w, http.StatusOK, map[string]string{"status": "fail"})
	}
}

func queryWeb3Days(r *http.Request) (int, error) {
	raw := strings.TrimSpace(r.URL.Query().Get("days"))
	if raw == "" {
		return 0, nil
	}
	return parseReleaseDays(raw)
}

func queryWeb3GoodsID(r *http.Request) uint64 {
	id, _ := strconv.ParseUint(strings.TrimSpace(r.URL.Query().Get("id")), 10, 64)
	return id
}

func (s *AppService) AdminWeb3GoodsList(w http.ResponseWriter, r *http.Request) {
	days, err := queryWeb3Days(r)
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]string{"status": "请选择释放天数"})
		return
	}
	rows, total, err := s.orders.ListWeb3Goods(r.Context(), days, parsePage(r), parsePageSize(r), auth.IsUser(r.Context()))
	if err != nil {
		writeWeb3GoodsBiz(w, err)
		return
	}
	list := make([]map[string]any, 0, len(rows))
	for _, p := range rows {
		list = append(list, s.web3GoodsJSON(r, p, false))
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"status": "ok",
		"count":  strconv.Itoa(total),
		"list":   list,
	})
}

func (s *AppService) AdminWeb3GoodsDetail(w http.ResponseWriter, r *http.Request) {
	id := queryWeb3GoodsID(r)
	if id == 0 {
		writeJSON(w, http.StatusOK, map[string]string{"status": "商品不存在"})
		return
	}
	p, err := s.orders.GetWeb3Goods(r.Context(), id, auth.IsUser(r.Context()))
	if err != nil {
		writeWeb3GoodsBiz(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "item": s.web3GoodsJSON(r, p, true)})
}

func writeWeb3GoodsReqErr(w http.ResponseWriter, err error) {
	if errors.Is(err, biz.ErrSKUInvalid) || errors.Is(err, biz.ErrInvalidReleaseDays) || errors.Is(err, biz.ErrInvalidAmount) {
		writeWeb3GoodsBiz(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "金额错误"})
}

func (s *AppService) AdminWeb3GoodsCreate(w http.ResponseWriter, r *http.Request) {
	in, err := readWeb3GoodsReq(r)
	if err != nil {
		writeWeb3GoodsReqErr(w, err)
		return
	}
	p, err := s.orders.CreateWeb3Goods(r.Context(), &in.Web3GoodsInput)
	if err != nil {
		writeWeb3GoodsBiz(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "item": s.web3GoodsJSON(r, p, true)})
}

func (s *AppService) AdminWeb3GoodsUpdate(w http.ResponseWriter, r *http.Request) {
	in, err := readWeb3GoodsReq(r)
	if err != nil {
		writeWeb3GoodsReqErr(w, err)
		return
	}
	if in.ID == 0 {
		writeJSON(w, http.StatusOK, map[string]string{"status": "商品不存在"})
		return
	}
	p, err := s.orders.UpdateWeb3Goods(r.Context(), &in.Web3GoodsInput)
	if err != nil {
		writeWeb3GoodsBiz(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "item": s.web3GoodsJSON(r, p, true)})
}

func (s *AppService) AdminWeb3GoodsStatus(w http.ResponseWriter, r *http.Request) {
	in, err := readWeb3GoodsReq(r)
	if err != nil {
		writeWeb3GoodsReqErr(w, err)
		return
	}
	if in.ID == 0 {
		writeJSON(w, http.StatusOK, map[string]string{"status": "商品不存在"})
		return
	}
	if !in.HasOnSale {
		writeJSON(w, http.StatusOK, map[string]string{"status": "fail"})
		return
	}
	p, err := s.orders.SetWeb3GoodsOnSale(r.Context(), in.ID, in.OnSale)
	if err != nil {
		writeWeb3GoodsBiz(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "item": s.web3GoodsJSON(r, p, false)})
}

func (s *AppService) AdminWeb3GoodsDelete(w http.ResponseWriter, r *http.Request) {
	in, err := readWeb3GoodsReq(r)
	if err != nil {
		writeWeb3GoodsReqErr(w, err)
		return
	}
	if in.ID == 0 {
		writeJSON(w, http.StatusOK, map[string]string{"status": "商品不存在"})
		return
	}
	if err := s.orders.DeleteWeb3Goods(r.Context(), in.ID); err != nil {
		writeWeb3GoodsBiz(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
}

func (s *AppService) AdminWeb3GoodsSort(w http.ResponseWriter, r *http.Request) {
	ids, err := parseWeb3GoodsSortIDs(r)
	if err != nil || len(ids) == 0 {
		writeJSON(w, http.StatusOK, map[string]string{"status": "排序参数错误"})
		return
	}
	if err := s.orders.SortWeb3Goods(r.Context(), ids); err != nil {
		if errors.Is(err, biz.ErrInvalidAmount) {
			writeJSON(w, http.StatusOK, map[string]string{"status": "排序参数错误"})
			return
		}
		writeWeb3GoodsBiz(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
}

func (s *AppService) AdminWeb3GoodsImageUpload(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, upload.MaxImageBytes+512*1024)
	if err := r.ParseMultipartForm(upload.MaxImageBytes + 512*1024); err != nil {
		if isMaxBytesErr(err) {
			writeJSON(w, http.StatusOK, map[string]string{"status": "图片不能超过5MB"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "请上传图片"})
		return
	}
	daysRaw := firstNonEmpty(r.Form.Get("days"), r.Form.Get("release_days"))
	if daysRaw != "" {
		if _, err := parseReleaseDays(daysRaw); err != nil {
			writeJSON(w, http.StatusOK, map[string]string{"status": "请选择释放天数"})
			return
		}
	}
	f, hdr, err := r.FormFile("file")
	if err != nil || f == nil {
		writeJSON(w, http.StatusOK, map[string]string{"status": "请上传图片"})
		return
	}
	defer f.Close()
	if hdr.Size > upload.MaxImageBytes {
		writeJSON(w, http.StatusOK, map[string]string{"status": "图片不能超过5MB"})
		return
	}
	data, err := io.ReadAll(io.LimitReader(f, upload.MaxImageBytes+1))
	if err != nil {
		if isMaxBytesErr(err) {
			writeJSON(w, http.StatusOK, map[string]string{"status": "图片不能超过5MB"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "图片上传失败"})
		return
	}
	if len(data) == 0 {
		writeJSON(w, http.StatusOK, map[string]string{"status": "请上传图片"})
		return
	}
	if len(data) > upload.MaxImageBytes {
		writeJSON(w, http.StatusOK, map[string]string{"status": "图片不能超过5MB"})
		return
	}
	ext, ok := upload.DetectImageExt(hdr.Filename, data)
	if !ok {
		writeJSON(w, http.StatusOK, map[string]string{"status": "图片格式不支持"})
		return
	}
	rel, err := upload.Save(s.uploadDir(), "web3_goods", ext, data)
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]string{"status": "图片上传失败"})
		return
	}
	url := upload.AbsoluteURL(upload.PublicBaseURL(r), rel)
	if url == "" {
		writeJSON(w, http.StatusOK, map[string]string{"status": "图片上传失败"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "url": url})
}

func isMaxBytesErr(err error) bool {
	if err == nil {
		return false
	}
	var maxErr *http.MaxBytesError
	if errors.As(err, &maxErr) {
		return true
	}
	return strings.Contains(strings.ToLower(err.Error()), "http: request body too large")
}

type web3GoodsReq struct {
	biz.Web3GoodsInput
	hasDays bool
}

func readWeb3GoodsReq(r *http.Request) (*web3GoodsReq, error) {
	in := &web3GoodsReq{}
	ct := r.Header.Get("Content-Type")
	if strings.Contains(ct, "json") {
		var body struct {
			ID       json.RawMessage               `json:"id"`
			Days     json.RawMessage               `json:"days"`
			Name     string                        `json:"name"`
			Title    string                        `json:"title"`
			Desc     string                        `json:"desc"`
			Goods    string                        `json:"goods"`
			Amount   json.RawMessage               `json:"amount"`
			DailyCap json.RawMessage               `json:"daily_cap"`
			Sort     json.RawMessage               `json:"sort"`
			OnSale   json.RawMessage               `json:"on_sale"`
			Image    json.RawMessage               `json:"image"`
			Detail   json.RawMessage               `json:"detail"`
			Contents map[string]web3GoodsContentIn `json:"contents"`
			SKUs     json.RawMessage               `json:"skus"`
		}
		if err := decodeJSON(r, &body); err != nil {
			return nil, err
		}
		in.ID, _ = strconv.ParseUint(strings.Trim(string(body.ID), `"`), 10, 64)
		in.Name = firstNonEmpty(body.Name, body.Title)
		in.Desc = firstNonEmpty(body.Desc, body.Goods)
		applyWeb3GoodsContents(in, body.Contents)
		if days := strings.Trim(string(body.Days), `"`); days != "" && days != "null" {
			in.hasDays = true
			if n, err := parseReleaseDays(days); err == nil {
				in.Days = n
			}
		}
		if amt := strings.Trim(string(body.Amount), `"`); amt != "" && amt != "null" {
			d, err := parseAmount(amt)
			if err != nil {
				return nil, err
			}
			in.Amount = d
		}
		if cap := strings.Trim(string(body.DailyCap), `"`); cap != "" && cap != "null" {
			d, err := parseAmount(cap)
			if err != nil {
				return nil, err
			}
			in.DailyCap = d
			in.HasDailyCap = true
		}
		if sortRaw := strings.Trim(string(body.Sort), `"`); sortRaw != "" && sortRaw != "null" {
			n, err := strconv.Atoi(sortRaw)
			if err != nil {
				return nil, biz.ErrInvalidAmount
			}
			in.Sort = n
			in.HasSort = true
		}
		if on := strings.Trim(string(body.OnSale), `"`); on != "" && on != "null" {
			in.HasOnSale = true
			in.OnSale = parseOnSaleFlag(on)
		}
		if s, ok := parseJSONString(body.Image); ok {
			in.HasImage = true
			in.Image = upload.NormalizeStored(s)
		}
		if s, ok := parseJSONString(body.Detail); ok {
			in.HasDetail = true
			in.Detail = s
		}
		if err := parseWeb3GoodsSKUs(in, body.SKUs); err != nil {
			return nil, err
		}
		return in, nil
	}
	if err := parseAdminForm(r); err != nil {
		return nil, err
	}
	in.ID, _ = strconv.ParseUint(r.Form.Get("id"), 10, 64)
	in.Name = firstNonEmpty(r.Form.Get("name"), r.Form.Get("title"))
	in.Desc = firstNonEmpty(r.Form.Get("desc"), r.Form.Get("goods"))
	if days := firstNonEmpty(r.Form.Get("days"), r.Form.Get("release_days")); days != "" {
		in.hasDays = true
		if n, err := parseReleaseDays(days); err == nil {
			in.Days = n
		}
	}
	if amt := r.Form.Get("amount"); amt != "" {
		d, err := parseAmount(amt)
		if err != nil {
			return nil, err
		}
		in.Amount = d
	}
	if _, ok := r.Form["daily_cap"]; ok {
		cap := strings.TrimSpace(r.Form.Get("daily_cap"))
		if cap == "" {
			in.HasDailyCap = true
		} else {
			d, err := parseAmount(cap)
			if err != nil {
				return nil, err
			}
			in.DailyCap = d
			in.HasDailyCap = true
		}
	}
	if _, ok := r.Form["sort"]; ok {
		sortRaw := strings.TrimSpace(r.Form.Get("sort"))
		if sortRaw == "" {
			in.HasSort = true
			in.Sort = 0
		} else {
			n, err := strconv.Atoi(sortRaw)
			if err != nil {
				return nil, biz.ErrInvalidAmount
			}
			in.Sort = n
			in.HasSort = true
		}
	}
	if _, ok := r.Form["on_sale"]; ok {
		in.HasOnSale = true
		in.OnSale = parseOnSaleFlag(r.Form.Get("on_sale"))
	}
	applyWeb3GoodsFormContents(in, r.Form)
	if err := parseWeb3GoodsFormSKUs(in, r.Form); err != nil {
		return nil, err
	}
	if _, ok := r.Form["image"]; ok {
		in.HasImage = true
		in.Image = upload.NormalizeStored(r.Form.Get("image"))
	}
	if _, ok := r.Form["detail"]; ok {
		in.HasDetail = true
		in.Detail = r.Form.Get("detail")
	}
	return in, nil
}

type web3GoodsContentIn struct {
	Title  string `json:"title"`
	Desc   string `json:"desc"`
	Image  string `json:"image"`
	Detail string `json:"detail"`
}

func applyWeb3GoodsContents(in *web3GoodsReq, contents map[string]web3GoodsContentIn) {
	if in == nil || len(contents) == 0 {
		return
	}
	in.Contents = make(map[string]biz.PackageContent, len(contents))
	for locale, content := range contents {
		in.Contents[strings.ToLower(strings.TrimSpace(locale))] = biz.PackageContent{
			Title: content.Title, GoodsDesc: content.Desc,
			Image: upload.NormalizeStored(content.Image), Detail: content.Detail,
		}
	}
	if zh, ok := in.Contents["zh"]; ok {
		in.Name, in.Desc = zh.Title, zh.GoodsDesc
		in.Image, in.Detail = zh.Image, zh.Detail
		in.HasImage, in.HasDetail = true, true
	}
}

func applyWeb3GoodsFormContents(in *web3GoodsReq, form url.Values) {
	if in == nil || form == nil {
		return
	}
	raw := strings.TrimSpace(form.Get("contents"))
	if strings.HasPrefix(raw, "{") {
		var contents map[string]web3GoodsContentIn
		if err := json.Unmarshal([]byte(raw), &contents); err == nil && len(contents) > 0 {
			applyWeb3GoodsContents(in, contents)
			return
		}
	}
	contents := map[string]web3GoodsContentIn{}
	for _, locale := range []string{"zh", "en"} {
		prefix := "contents[" + locale + "]"
		if !formHasPrefix(form, prefix) {
			continue
		}
		contents[locale] = web3GoodsContentIn{
			Title:  form.Get(prefix + "[title]"),
			Desc:   form.Get(prefix + "[desc]"),
			Image:  form.Get(prefix + "[image]"),
			Detail: form.Get(prefix + "[detail]"),
		}
	}
	applyWeb3GoodsContents(in, contents)
}

func parseWeb3GoodsFormSKUs(in *web3GoodsReq, form url.Values) error {
	if in == nil || form == nil {
		return nil
	}
	raw := strings.TrimSpace(form.Get("skus"))
	if raw != "" && raw != "null" && (strings.HasPrefix(raw, "[") || strings.HasPrefix(raw, "{")) {
		return parseWeb3GoodsSKUs(in, json.RawMessage(raw))
	}
	maxIdx := -1
	for key := range form {
		idx, ok := skuFormIndex(key)
		if !ok {
			continue
		}
		if idx > maxIdx {
			maxIdx = idx
		}
	}
	if maxIdx < 0 {
		if _, ok := form["skus"]; ok {
			in.HasSKUs = true
			in.SKUs = []biz.PackageSKU{}
		}
		return nil
	}
	rows := make([]map[string]string, 0, maxIdx+1)
	for i := 0; i <= maxIdx; i++ {
		prefix := "skus[" + strconv.Itoa(i) + "]"
		rows = append(rows, map[string]string{
			"id":      form.Get(prefix + "[id]"),
			"name":    form.Get(prefix + "[name]"),
			"name_zh": form.Get(prefix + "[name_zh]"),
			"name_en": form.Get(prefix + "[name_en]"),
			"amount":  form.Get(prefix + "[amount]"),
			"image":   form.Get(prefix + "[image]"),
			"enabled": form.Get(prefix + "[enabled]"),
		})
	}
	encoded, err := json.Marshal(rows)
	if err != nil {
		return biz.ErrSKUInvalid
	}
	return parseWeb3GoodsSKUs(in, encoded)
}

func skuFormIndex(key string) (int, bool) {
	if !strings.HasPrefix(key, "skus[") {
		return 0, false
	}
	rest := strings.TrimPrefix(key, "skus[")
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

func formHasPrefix(form url.Values, prefix string) bool {
	if form == nil {
		return false
	}
	if _, ok := form[prefix]; ok {
		return true
	}
	for key := range form {
		if strings.HasPrefix(key, prefix+"[") || key == prefix {
			return true
		}
	}
	return false
}

func parseWeb3GoodsSKUs(in *web3GoodsReq, raw json.RawMessage) error {
	if in == nil || len(raw) == 0 || string(raw) == "null" {
		return nil
	}
	var rows []struct {
		ID      json.RawMessage `json:"id"`
		Name    string          `json:"name"`
		NameZh  string          `json:"name_zh"`
		NameEn  string          `json:"name_en"`
		Amount  json.RawMessage `json:"amount"`
		Image   json.RawMessage `json:"image"`
		Enabled json.RawMessage `json:"enabled"`
	}
	if err := json.Unmarshal(raw, &rows); err != nil {
		return biz.ErrSKUInvalid
	}
	in.HasSKUs = true
	in.SKUs = make([]biz.PackageSKU, 0, len(rows))
	for _, row := range rows {
		id, _ := strconv.ParseUint(strings.Trim(string(row.ID), `"`), 10, 64)
		amt := decimal.Zero
		if s := strings.Trim(string(row.Amount), `"`); s != "" && s != "null" {
			d, err := parseAmount(s)
			if err != nil {
				return err
			}
			amt = d
		}
		enabled := true
		if on := strings.Trim(string(row.Enabled), `"`); on != "" && on != "null" {
			enabled = parseOnSaleFlag(on)
		}
		image := ""
		if s, ok := parseJSONString(row.Image); ok {
			image = upload.NormalizeStored(s)
		}
		in.SKUs = append(in.SKUs, biz.PackageSKU{
			ID:      id,
			Name:    firstNonEmpty(row.NameZh, row.Name),
			NameEn:  row.NameEn,
			Amount:  amt,
			Image:   image,
			Enabled: enabled,
		})
	}
	return nil
}

func parseJSONString(raw json.RawMessage) (string, bool) {
	if len(raw) == 0 || string(raw) == "null" {
		return "", false
	}
	var s string
	if err := json.Unmarshal(raw, &s); err != nil {
		return "", false
	}
	return s, true
}

func parseAdminForm(r *http.Request) error {
	ct := r.Header.Get("Content-Type")
	if strings.Contains(ct, "multipart/form-data") {
		return r.ParseMultipartForm(8 << 20)
	}
	return r.ParseForm()
}

func parseOnSaleFlag(s string) bool {
	s = strings.TrimSpace(strings.ToLower(s))
	if s == "0" || s == "false" || s == "off" || s == "no" {
		return false
	}
	return isTruthyFlag(s)
}

func parseWeb3GoodsSortIDs(r *http.Request) ([]uint64, error) {
	ct := r.Header.Get("Content-Type")
	if strings.Contains(ct, "json") {
		var body struct {
			IDs json.RawMessage `json:"ids"`
		}
		if err := decodeJSON(r, &body); err != nil {
			return nil, err
		}
		return parseIDList(string(body.IDs))
	}
	if err := parseAdminForm(r); err != nil {
		return nil, err
	}
	raw := firstNonEmpty(r.Form.Get("ids"), strings.Join(r.Form["ids"], ","))
	return parseIDList(raw)
}

func parseIDList(raw string) ([]uint64, error) {
	raw = strings.TrimSpace(raw)
	raw = strings.Trim(raw, `[]"`)
	if raw == "" || raw == "null" {
		return nil, biz.ErrInvalidAmount
	}
	parts := strings.FieldsFunc(raw, func(r rune) bool {
		return r == ',' || r == ' ' || r == '\n' || r == '\t'
	})
	out := make([]uint64, 0, len(parts))
	for _, p := range parts {
		p = strings.Trim(p, `"'`)
		if p == "" {
			continue
		}
		id, err := strconv.ParseUint(p, 10, 64)
		if err != nil || id == 0 {
			return nil, biz.ErrInvalidAmount
		}
		out = append(out, id)
	}
	if len(out) == 0 {
		return nil, biz.ErrInvalidAmount
	}
	return out, nil
}
