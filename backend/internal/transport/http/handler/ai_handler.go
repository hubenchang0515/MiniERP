package handler

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"io"
	"log"
	"minioa/internal/model"
	"minioa/internal/service"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

func (h *Handler) AIResources(c *gin.Context) {
	page, size, ok := pagination(c)
	if !ok {
		return
	}
	items := []model.AIResource{}
	var total int64
	q := h.Service.ReadDB().WithContext(c.Request.Context()).Model(&model.AIResource{})
	if text := c.Query("q"); text != "" {
		q = q.Where("name LIKE ? OR base_url LIKE ?", "%"+text+"%", "%"+text+"%")
	}
	if err := q.Count(&total).Error; err != nil {
		RespondError(c, err)
		return
	}
	if err := q.Preload("Keys", func(db *gorm.DB) *gorm.DB { return db.Order("id") }).Order("id DESC").Offset((page - 1) * size).Limit(size).Find(&items).Error; err != nil {
		RespondError(c, err)
		return
	}
	c.JSON(200, gin.H{"items": items, "total": total, "page": page, "pageSize": size})
}
func (h *Handler) SaveAIResource(c *gin.Context) {
	var resourceID uint
	if c.Param("id") != "" {
		var ok bool
		resourceID, ok = id(c)
		if !ok {
			return
		}
	}
	var in service.AIResourceInput
	if !bind(c, &in) {
		return
	}
	r, err := h.Service.SaveAIResource(resourceID, in)
	if err != nil {
		RespondError(c, err)
		return
	}
	c.JSON(200, r)
}

func (h *Handler) FetchAIModels(c *gin.Context) {
	var in service.AIModelListInput
	if !bind(c, &in) {
		return
	}
	ids, err := h.Service.FetchAIModels(c.Request.Context(), in)
	if err != nil {
		RespondError(c, err)
		return
	}
	c.JSON(200, gin.H{"modelIds": ids})
}
func (h *Handler) AIResourceOptions(c *gin.Context) {
	page, size, ok := pagination(c)
	if !ok {
		return
	}
	q := h.Service.ReadDB().WithContext(c.Request.Context()).Model(&model.AIResource{})
	if text := c.Query("q"); text != "" {
		q = q.Where("name LIKE ?", "%"+text+"%")
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		RespondError(c, err)
		return
	}
	resources := []model.AIResource{}
	if err := q.Select("id", "name", "enabled").Order("id DESC").Offset((page - 1) * size).Limit(size).Find(&resources).Error; err != nil {
		RespondError(c, err)
		return
	}
	type option struct {
		ID      uint   `json:"id"`
		Name    string `json:"name"`
		Enabled bool   `json:"enabled"`
	}
	items := make([]option, 0, len(resources))
	for _, r := range resources {
		items = append(items, option{ID: r.ID, Name: r.Name, Enabled: r.Enabled})
	}
	c.JSON(200, gin.H{"items": items, "total": total, "page": page, "pageSize": size})
}
func (h *Handler) AIKeys(c *gin.Context) {
	page, size, ok := pagination(c)
	if !ok {
		return
	}
	u := c.MustGet("user").(model.User)
	db := h.Service.ReadDB().WithContext(c.Request.Context())
	q := db.Model(&model.AIKey{}).Joins("JOIN users ON users.id = ai_keys.user_id")
	all := c.Query("all") == "true"
	if all {
		if !systemAdmin(u) {
			RespondError(c, service.Fail(403, "forbidden", "仅系统管理员可查看全部密钥"))
			return
		}
	} else {
		q = q.Where("ai_keys.user_id = ?", u.ID)
	}
	if text := c.Query("q"); text != "" {
		q = q.Where("(ai_keys.name LIKE ? OR users.username LIKE ?)", "%"+text+"%", "%"+text+"%")
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		RespondError(c, err)
		return
	}
	items := []struct {
		model.AIKey
		Username string `json:"username"`
	}{}
	if err := q.Select("ai_keys.*, users.username AS username").Order("ai_keys.id DESC").Offset((page - 1) * size).Limit(size).Scan(&items).Error; err != nil {
		RespondError(c, err)
		return
	}
	result := gin.H{"items": items, "total": total, "page": page, "pageSize": size}
	if !all {
		count, err := service.ActiveAIKeyCount(db, u.ID)
		if err != nil {
			RespondError(c, err)
			return
		}
		result["activeCount"], result["limit"] = count, service.MaxActiveAIKeys
	}
	c.JSON(200, result)
}
func systemAdmin(u model.User) bool {
	for _, r := range u.Roles {
		if r.System {
			return true
		}
	}
	return false
}
func (h *Handler) CreateAIKey(c *gin.Context) {
	var in service.AIKeyInput
	if !bind(c, &in) {
		return
	}
	u := c.MustGet("user").(model.User)
	k, secret, err := h.Service.CreateAIKey(u.ID, in)
	if err != nil {
		RespondError(c, err)
		return
	}
	c.JSON(201, gin.H{"key": k, "secret": secret})
}
func (h *Handler) ChangeAIKey(c *gin.Context) {
	keyID, ok := id(c)
	if !ok {
		return
	}
	u := c.MustGet("user").(model.User)
	owner := u.ID
	if systemAdmin(u) && c.Request.Method == http.MethodDelete {
		owner = 0
	}
	secret, err := h.Service.ChangeAIKey(keyID, owner, c.Request.Method == http.MethodPost)
	if err != nil {
		RespondError(c, err)
		return
	}
	if secret != "" {
		c.JSON(200, gin.H{"secret": secret})
	} else {
		c.Status(204)
	}
}

func (h *Handler) AIProxy(c *gin.Context) {
	started := time.Now()
	requestID := rand.Text()
	c.Header("X-Request-ID", requestID)
	path := c.Param("path")
	allowed := (c.Request.Method == "GET" && path == "/models") || (c.Request.Method == "POST" && (path == "/chat/completions" || path == "/embeddings" || path == "/responses"))
	if !allowed {
		c.JSON(404, gin.H{"error": gin.H{"message": "不支持的 AI 接口", "type": "invalid_request_error"}})
		return
	}
	parts := strings.Fields(c.GetHeader("Authorization"))
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		RespondError(c, service.Fail(401, "invalid_key", "请提供密钥"))
		return
	}
	key, err := h.Service.ResolveAIKey(parts[1])
	if err != nil {
		RespondError(c, err)
		return
	}
	if c.Request.Method == http.MethodGet {
		c.Header("Cache-Control", "no-store")
		type entry struct {
			ID      string `json:"id"`
			Object  string `json:"object"`
			Created int64  `json:"created"`
			OwnedBy string `json:"owned_by"`
		}
		var models []model.AIModel
		if err := availableAIModels(h.Service).Select("ai_models.*").Order("ai_models.internal_id").Find(&models).Error; err != nil {
			RespondError(c, err)
			return
		}
		items := make([]entry, 0, len(models))
		for _, m := range models {
			items = append(items, entry{ID: m.InternalID, Object: "model", Created: m.CreatedAt.Unix(), OwnedBy: "minioa"})
		}
		c.JSON(200, gin.H{"object": "list", "data": items})
		return
	}
	release, err := h.Service.AcquireAIProxy()
	if err != nil {
		log.Printf("ai proxy rejected request_id=%s reason=admission_failed limit=%d", requestID, h.Service.Config.ProxyConcurrency())
		RespondError(c, err)
		return
	}
	defer release()
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 8<<20)
	body, err := io.ReadAll(c.Request.Body)
	_ = c.Request.Body.Close()
	if err != nil {
		status := http.StatusBadRequest
		var limit *http.MaxBytesError
		if errors.As(err, &limit) {
			status = http.StatusRequestEntityTooLarge
		}
		RespondError(c, service.Fail(status, "invalid_request", "请求体无法读取或超过 8 MiB"))
		return
	}
	var payload map[string]json.RawMessage
	if err := json.Unmarshal(body, &payload); err != nil || payload == nil {
		RespondError(c, service.Fail(400, "invalid_request", "请求体必须是 JSON 对象"))
		return
	}
	var internalID string
	if err := json.Unmarshal(payload["model"], &internalID); err != nil || internalID == "" {
		RespondError(c, service.Fail(400, "invalid_model", "请使用系统内部模型 ID"))
		return
	}
	mapping, err := h.Service.ResolveAIModel(internalID)
	if err != nil {
		RespondError(c, err)
		return
	}
	target, err := url.Parse(mapping.Resource.BaseURL)
	if err != nil {
		RespondError(c, err)
		return
	}
	quota, err := h.Service.ReserveAIQuota(c.Request.Context(), requestID, key, mapping, time.Time{})
	if err != nil {
		var blocked *service.Error
		if errors.As(err, &blocked) && blocked.Status == http.StatusForbidden {
			c.JSON(blocked.Status, gin.H{"error": gin.H{
				"message": blocked.Message, "type": blocked.Code, "code": blocked.Code,
			}})
			return
		}
		RespondError(c, err)
		return
	}
	var observer *aiUsageObserver
	var quotaRequestID string
	if quota != nil {
		quotaRequestID = quota.RequestID
		defer func() {
			var usage *service.AITokenUsage
			if observer != nil {
				usage = observer.finish()
			}
			ctx, cancel := context.WithTimeout(context.WithoutCancel(c.Request.Context()), 5*time.Second)
			defer cancel()
			if err := h.Service.SettleAIQuota(ctx, quotaRequestID, usage); err != nil {
				log.Printf("ai quota settlement pending request_id=%s: %v", quotaRequestID, err)
			}
		}()
		if path == "/chat/completions" && !mapping.Pricing.SkipStreamUsage {
			if err := requestChatStreamUsage(payload); err != nil {
				RespondError(c, err)
				return
			}
		}
	}
	payload["model"], _ = json.Marshal(mapping.ExternalID)
	body, err = json.Marshal(payload)
	if err != nil {
		RespondError(c, err)
		return
	}
	c.Request.Body = io.NopCloser(bytes.NewReader(body))
	c.Request.ContentLength = int64(len(body))
	c.Request.TransferEncoding = nil
	if err = h.Service.RecordAIRequest(key.ID); err != nil {
		RespondError(c, err)
		return
	}
	clientCtx := c.Request.Context()
	idleTimeout := h.Service.Config.ProxyIdleTimeout()
	ctx, activity := watchProxyActivity(clientCtx, idleTimeout)
	defer activity.close()
	var streamErr error
	var writeTimedOut atomic.Bool
	defer logProxyAbort(clientCtx, ctx, &streamErr, &writeTimedOut, path, requestID, started)
	c.Request = c.Request.WithContext(ctx)
	_ = http.NewResponseController(c.Writer).SetWriteDeadline(time.Time{})
	timing := &service.AIProxyTiming{}
	proxy := httputil.ReverseProxy{Transport: &service.AIProviderTransport{Service: h.Service, ResourceID: mapping.ResourceID, Body: body, Inference: true, Timing: timing, QuotaRequestID: quotaRequestID}, FlushInterval: -1, ErrorLog: log.New(io.Discard, "", 0), Rewrite: func(r *httputil.ProxyRequest) {
		r.SetURL(target)
		r.Out.URL.Path = strings.TrimRight(target.Path, "/") + path
		r.Out.URL.RawPath = ""
		r.Out.Host = target.Host
		r.Out.Header = http.Header{"Content-Type": []string{"application/json"}, "Content-Length": []string{strconv.Itoa(len(body))}, "Accept": []string{r.In.Header.Get("Accept")}}
	}, ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
		reason := proxyStopReason(clientCtx, ctx, err)
		if reason == "upstream_read_failed" {
			reason = "upstream_connection_failed"
		}
		var e *service.Error
		if errors.As(err, &e) {
			reason = e.Code
		}
		log.Printf("ai proxy stopped request_id=%s stage=headers reason=%s path=%s attempts=%d elapsed=%s", requestID, reason, path, timing.Attempts, time.Since(started).Round(time.Millisecond))
		if clientCtx.Err() != nil {
			return
		}
		w.Header().Set("Content-Type", "application/json")
		status, message, kind := 502, "上游 AI 服务连接失败", "upstream_error"
		if errors.Is(context.Cause(ctx), errAIIdleTimeout) {
			status, message, kind = 504, "上游 AI 服务长时间未返回数据", "gateway_idle_timeout"
		}
		if errors.As(err, &e) {
			status, message, kind = e.Status, e.Message, e.Code
			if e.RetryAfter > 0 {
				w.Header().Set("Retry-After", retryAfterSeconds(e.RetryAfter))
			}
		}
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(gin.H{"error": gin.H{"message": message, "type": kind}})
	}, ModifyResponse: func(r *http.Response) error {
		activity.touch()
		if r.StatusCode >= 300 {
			retryAfter := r.Header.Get("Retry-After")
			r.Body.Close()
			r.Body = io.NopCloser(strings.NewReader(`{"error":{"message":"上游 AI 服务请求失败","type":"upstream_error"}}`))
			r.ContentLength = -1
			r.Header = http.Header{"Content-Type": []string{"application/json"}}
			if retryAfter != "" {
				r.Header.Set("Retry-After", retryAfter)
			}
			if r.StatusCode < 400 {
				r.StatusCode = 502
			}
		}
		r.Header.Del("Set-Cookie")
		r.Header.Del("Authorization")
		// ReverseProxy adds provider headers to the existing response headers.
		// Keep the locally assigned ID and discard the provider's ID.
		r.Header.Del("X-Request-ID")
		stream := strings.HasPrefix(strings.ToLower(r.Header.Get("Content-Type")), "text/event-stream")
		if stream {
			r.Header.Set("Cache-Control", "no-cache")
			r.Header.Set("X-Accel-Buffering", "no")
		}
		observed := &observedProxyBody{ReadCloser: r.Body, readErr: &streamErr, touch: activity.touch}
		if quota != nil && r.StatusCode >= 200 && r.StatusCode < 300 {
			observer = observeAIUsage(path, stream)
			observed.observe = observer.observe
		}
		r.Body = observed
		local := time.Since(started) - timing.UpstreamHeaders
		r.Header.Set("Server-Timing", fmt.Sprintf("gateway;dur=%.2f, upstream_headers;dur=%.2f, attempts;desc=\"%d\"", float64(local)/float64(time.Millisecond), float64(timing.UpstreamHeaders)/float64(time.Millisecond), timing.Attempts))
		return nil
	}}
	proxy.ServeHTTP(&proxyStreamWriter{ResponseWriter: c.Writer, idle: idleTimeout, timedOut: &writeTimedOut}, c.Request)
}

func requestChatStreamUsage(payload map[string]json.RawMessage) error {
	var stream bool
	if json.Unmarshal(payload["stream"], &stream) != nil || !stream {
		return nil
	}
	options := make(map[string]json.RawMessage)
	if existing, ok := payload["stream_options"]; ok {
		if err := json.Unmarshal(existing, &options); err != nil {
			return service.Fail(400, "invalid_request", "stream_options 必须是 JSON 对象")
		}
		if options == nil {
			options = make(map[string]json.RawMessage)
		}
	}
	options["include_usage"] = json.RawMessage("true")
	payload["stream_options"], _ = json.Marshal(options)
	return nil
}
