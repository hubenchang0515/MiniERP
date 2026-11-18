package service

import (
	"context"
	"encoding/json"
	"errors"
	"gorm.io/gorm"
	"io"
	"minierp/internal/model"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"
)

type AIModelListInput struct {
	ResourceID uint   `json:"resourceId"`
	BaseURL    string `json:"baseURL"`
	APIKey     string `json:"apiKey"`
}

// FetchAIModels only returns model IDs, never provider response bodies or credentials.
func (s *Service) FetchAIModels(ctx context.Context, in AIModelListInput) ([]string, error) {
	if in.BaseURL == "" && in.ResourceID != 0 {
		var resource model.AIResource
		if err := s.ReadDB().First(&resource, in.ResourceID).Error; err != nil {
			return nil, Normalize(err)
		}
		in.BaseURL = resource.BaseURL
	}
	target, err := parseAIBaseURL(in.BaseURL)
	if err != nil {
		return nil, err
	}
	secret := in.APIKey
	usePool := secret == ""
	if secret == "" {
		if in.ResourceID == 0 {
			return nil, Fail(400, "validation", "请先填写外部 API KEY")
		}
		var resource model.AIResource
		if err := s.ReadDB().First(&resource, in.ResourceID).Error; err != nil {
			return nil, Normalize(err)
		}
		// A stored credential may only be reused with its saved upstream address.
		if strings.TrimRight(strings.TrimSpace(resource.BaseURL), "/") != target.String() {
			return nil, Fail(400, "validation", "API 基础地址已修改，请重新填写外部 API KEY 后获取模型")
		}
	}
	if len(secret) > 8192 || strings.ContainsAny(secret, "\r\n") {
		return nil, Fail(400, "validation", "外部 API KEY 格式不正确")
	}
	release, err := s.AcquireAIProxy()
	if err != nil {
		return nil, err
	}
	defer release()
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	target.Path = strings.TrimRight(target.Path, "/") + "/models"
	target.RawPath = ""
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target.String(), nil)
	if err != nil {
		return nil, Fail(400, "validation", "API 基础地址无效")
	}
	req.Header.Set("Authorization", "Bearer "+secret)
	req.Header.Set("Accept", "application/json")
	client := &http.Client{Transport: aiHTTPTransport, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	if usePool {
		client.Transport = &AIProviderTransport{Service: s, ResourceID: in.ResourceID}
	}
	response, err := client.Do(req)
	if err != nil {
		return nil, Fail(502, "upstream_error", "获取上游模型失败或超时，请检查地址及网络")
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, Fail(502, "upstream_error", "上游模型列表请求失败（HTTP "+strconv.Itoa(response.StatusCode)+"），请检查地址、密钥及上游是否支持模型列表")
	}
	const limit = 2 << 20
	body, err := io.ReadAll(io.LimitReader(response.Body, limit+1))
	if err != nil || len(body) > limit {
		return nil, Fail(502, "upstream_error", "上游模型列表无法读取或超过 2 MiB")
	}
	var payload struct {
		Data *[]struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &payload); err != nil || payload.Data == nil {
		return nil, Fail(502, "upstream_error", "上游未返回兼容的模型列表，可手动填写外部模型 ID")
	}
	ids := make([]string, 0, len(*payload.Data))
	seen := make(map[string]bool)
	for _, m := range *payload.Data {
		if validAIModelID(m.ID) && !seen[m.ID] {
			seen[m.ID] = true
			ids = append(ids, m.ID)
		}
	}
	sort.Strings(ids)
	return ids, nil
}

type AIModelInput struct {
	InternalID string                `json:"internalId"`
	ResourceID uint                  `json:"resourceId"`
	ExternalID string                `json:"externalId"`
	Enabled    bool                  `json:"enabled"`
	Pricing    *model.AIModelPricing `json:"pricing"`
}

func (s *Service) SaveAIModel(id uint, in AIModelInput) (model.AIModel, error) {
	var m model.AIModel
	if !validAIModelID(in.InternalID) || !validAIModelID(in.ExternalID) || in.ResourceID == 0 {
		return m, Fail(400, "validation", "请选择供应商，模型 ID 必须为 1–256 字节且不包含空白或控制字符")
	}
	if in.Pricing != nil {
		p := in.Pricing
		if p.InputPerMillion < 0 || p.InputPerMillion > MaxAIPoints ||
			p.OutputPerMillion < 0 || p.OutputPerMillion > MaxAIPoints ||
			p.ReservePoints < 0 || p.ReservePoints > MaxAIPoints ||
			(p.ReservePoints == 0 && (p.InputPerMillion > 0 || p.OutputPerMillion > 0)) {
			return m, Fail(400, "invalid_pricing", "模型单价和预留须为有效的整数积分，设置价格时须预留至少 1 积分")
		}
	}
	err := s.DB.Transaction(func(tx *gorm.DB) error {
		var resource model.AIResource
		if err := tx.First(&resource, in.ResourceID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return Fail(400, "validation", "供应商不存在")
			}
			return err
		}
		if id != 0 {
			if err := tx.First(&m, id).Error; err != nil {
				return err
			}
		}
		m.InternalID, m.ResourceID, m.ExternalID, m.Enabled = in.InternalID, in.ResourceID, in.ExternalID, in.Enabled
		if in.Pricing != nil {
			m.Pricing = *in.Pricing
		}
		return tx.Save(&m).Error
	})
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		err = Fail(409, "duplicate", "内部模型 ID 已存在")
	}
	return m, Normalize(err)
}

func (s *Service) ResolveAIModel(internalID string) (model.AIModel, error) {
	var m model.AIModel
	err := s.ReadDB().Joins("Resource").Where("ai_models.internal_id = ? AND ai_models.enabled = ?", internalID, true).First(&m).Error
	if errors.Is(err, gorm.ErrRecordNotFound) || (err == nil && !m.Resource.Enabled) {
		return m, Fail(400, "invalid_model", "模型不存在或已停用，请通过 /ai/v1/models 查询可用模型")
	}
	if err != nil {
		return m, err
	}
	return m, nil
}
