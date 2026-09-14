package service

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/cigc/app/internal/biz"
	"github.com/cigc/app/internal/pkg/middleware/auth"
	"github.com/cigc/app/internal/pkg/upload"
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

func (s *AppService) web3GoodsJSON(r *http.Request, p *biz.Package) map[string]any {
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
	return map[string]any{
		"id":        p.ID,
		"desc":      p.GoodsDesc,
		"amount":    decStr(p.Amount),
		"daily_cap": decStr(p.DailyCap),
		"days":      days,
		"sort":      p.SortOrder,
		"on_sale":   onSale,
		"image":     upload.AbsoluteURL(upload.PublicBaseURL(r), p.Image),
	}
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
	case errors.Is(err, biz.ErrPackageInUse):
		writeJSON(w, http.StatusOK, map[string]string{"status": "已有订单不能删除，请先下架"})
	case errors.Is(err, biz.ErrInvalidAmount):
		writeJSON(w, http.StatusOK, map[string]string{"status": "金额错误"})
	case errors.Is(err, biz.ErrInvalidReleaseDays):
		writeJSON(w, http.StatusOK, map[string]string{"status": "请选择释放天数"})
	default:
		writeJSON(w, http.StatusOK, map[string]string{"status": "fail"})
	}
}

func queryWeb3Days(r *http.Request) (int, error) {
	raw := strings.TrimSpace(r.URL.Query().Get("days"))
	if raw == "" {
		return 0, biz.ErrInvalidReleaseDays
	}
	return parseReleaseDays(raw)
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
		list = append(list, s.web3GoodsJSON(r, p))
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"status": "ok",
		"count":  strconv.Itoa(total),
		"list":   list,
	})
}

func (s *AppService) AdminWeb3GoodsCreate(w http.ResponseWriter, r *http.Request) {
	in, err := readWeb3GoodsReq(r)
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]string{"status": "金额错误"})
		return
	}
	if !in.hasDays {
		writeJSON(w, http.StatusOK, map[string]string{"status": "请选择释放天数"})
		return
	}
	p, err := s.orders.CreateWeb3Goods(r.Context(), &in.Web3GoodsInput)
	if err != nil {
		writeWeb3GoodsBiz(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "item": s.web3GoodsJSON(r, p)})
}

func (s *AppService) AdminWeb3GoodsUpdate(w http.ResponseWriter, r *http.Request) {
	in, err := readWeb3GoodsReq(r)
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]string{"status": "金额错误"})
		return
	}
	if in.ID == 0 {
		writeJSON(w, http.StatusOK, map[string]string{"status": "商品不存在"})
		return
	}
	if !in.hasDays {
		writeJSON(w, http.StatusOK, map[string]string{"status": "请选择释放天数"})
		return
	}
	p, err := s.orders.UpdateWeb3Goods(r.Context(), &in.Web3GoodsInput)
	if err != nil {
		writeWeb3GoodsBiz(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "item": s.web3GoodsJSON(r, p)})
}

func (s *AppService) AdminWeb3GoodsStatus(w http.ResponseWriter, r *http.Request) {
	in, err := readWeb3GoodsReq(r)
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]string{"status": "金额错误"})
		return
	}
	if in.ID == 0 {
		writeJSON(w, http.StatusOK, map[string]string{"status": "商品不存在"})
		return
	}
	if !in.hasDays {
		writeJSON(w, http.StatusOK, map[string]string{"status": "请选择释放天数"})
		return
	}
	if !in.HasOnSale {
		writeJSON(w, http.StatusOK, map[string]string{"status": "fail"})
		return
	}
	p, err := s.orders.SetWeb3GoodsOnSale(r.Context(), in.ID, in.Days, in.OnSale)
	if err != nil {
		writeWeb3GoodsBiz(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "item": s.web3GoodsJSON(r, p)})
}

func (s *AppService) AdminWeb3GoodsDelete(w http.ResponseWriter, r *http.Request) {
	in, err := readWeb3GoodsReq(r)
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]string{"status": "金额错误"})
		return
	}
	if in.ID == 0 {
		writeJSON(w, http.StatusOK, map[string]string{"status": "商品不存在"})
		return
	}
	if !in.hasDays {
		writeJSON(w, http.StatusOK, map[string]string{"status": "请选择释放天数"})
		return
	}
	if err := s.orders.DeleteWeb3Goods(r.Context(), in.ID, in.Days); err != nil {
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
	if _, err := parseReleaseDays(daysRaw); err != nil {
		writeJSON(w, http.StatusOK, map[string]string{"status": "请选择释放天数"})
		return
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
			ID       json.RawMessage `json:"id"`
			Days     json.RawMessage `json:"days"`
			Name     string          `json:"name"`
			Title    string          `json:"title"`
			Desc     string          `json:"desc"`
			Goods    string          `json:"goods"`
			Amount   json.RawMessage `json:"amount"`
			DailyCap json.RawMessage `json:"daily_cap"`
			Sort     json.RawMessage `json:"sort"`
			OnSale   json.RawMessage `json:"on_sale"`
			Image    json.RawMessage `json:"image"`
		}
		if err := decodeJSON(r, &body); err != nil {
			return nil, err
		}
		in.ID, _ = strconv.ParseUint(strings.Trim(string(body.ID), `"`), 10, 64)
		in.Name = firstNonEmpty(body.Name, body.Title)
		in.Desc = firstNonEmpty(body.Desc, body.Goods)
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
		if img := strings.Trim(string(body.Image), `"`); img != "" && img != "null" {
			in.HasImage = true
			in.Image = upload.NormalizeStored(img)
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
	if img := strings.TrimSpace(r.Form.Get("image")); img != "" {
		in.HasImage = true
		in.Image = upload.NormalizeStored(img)
	}
	return in, nil
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
