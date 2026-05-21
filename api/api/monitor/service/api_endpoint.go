package service

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"dodevops-api/api/monitor/dao"
	"dodevops-api/api/monitor/model"
	"dodevops-api/common/result"

	"github.com/gin-gonic/gin"
)

type APIEndpointServiceInterface interface {
	GetList(c *gin.Context, req *model.APIEndpointListReq)
	GetByID(c *gin.Context, id uint)
	Add(c *gin.Context, req *model.APIEndpointAddReq)
	Update(c *gin.Context, req *model.APIEndpointUpdateReq)
	Delete(c *gin.Context, id uint)
	BatchDelete(c *gin.Context, ids []uint)
	CheckEndpoint(c *gin.Context, id uint)
	CheckAllEndpoints(c *gin.Context)
}

type APIEndpointServiceImpl struct {
	dao          *dao.APIEndpointDao
	incidentDao  *dao.IncidentDao
	alertService AlertServiceInterface
}

func NewAPIEndpointService() APIEndpointServiceInterface {
	return &APIEndpointServiceImpl{
		dao:          dao.NewAPIEndpointDao(),
		incidentDao:  dao.NewIncidentDao(),
		alertService: NewAlertService(),
	}
}

func (s *APIEndpointServiceImpl) GetList(c *gin.Context, req *model.APIEndpointListReq) {
	list, total, err := s.dao.GetList(req)
	if err != nil {
		result.Failed(c, 500, "查询列表失败: "+err.Error())
		return
	}
	result.Success(c, map[string]interface{}{"list": list, "total": total})
}

func (s *APIEndpointServiceImpl) GetByID(c *gin.Context, id uint) {
	m, err := s.dao.GetByID(id)
	if err != nil {
		result.Failed(c, 404, "未找到该记录")
		return
	}
	result.Success(c, m)
}

func (s *APIEndpointServiceImpl) Add(c *gin.Context, req *model.APIEndpointAddReq) {
	if req.Method == "" {
		req.Method = "GET"
	}
	if req.CheckInterval <= 0 {
		req.CheckInterval = 300
	}
	if req.Timeout <= 0 {
		req.Timeout = 10
	}
	if req.ExpectedCode <= 0 {
		req.ExpectedCode = 200
	}
	now := time.Now().Format("2006-01-02 15:04:05")
	m := &model.MonitorAPIEndpoint{
		Name:          req.Name,
		URL:           req.URL,
		Method:        strings.ToUpper(req.Method),
		Headers:       req.Headers,
		Body:          req.Body,
		CheckInterval: req.CheckInterval,
		Timeout:       req.Timeout,
		ExpectedCode:  req.ExpectedCode,
		ExpectedBody:  req.ExpectedBody,
		Status:        1,
		CheckTime:     now,
	}
	if err := s.dao.Create(m); err != nil {
		result.Failed(c, 500, "添加失败: "+err.Error())
		return
	}
	s.checkEndpointInternal(m)
	s.dao.Update(m)
	result.Success(c, m)
}

func (s *APIEndpointServiceImpl) Update(c *gin.Context, req *model.APIEndpointUpdateReq) {
	m, err := s.dao.GetByID(req.ID)
	if err != nil {
		result.Failed(c, 404, "未找到该记录")
		return
	}
	if req.Method == "" {
		req.Method = "GET"
	}
	if req.CheckInterval <= 0 {
		req.CheckInterval = 300
	}
	if req.Timeout <= 0 {
		req.Timeout = 10
	}
	if req.ExpectedCode <= 0 {
		req.ExpectedCode = 200
	}
	m.Name = req.Name
	m.URL = req.URL
	m.Method = strings.ToUpper(req.Method)
	m.Headers = req.Headers
	m.Body = req.Body
	m.CheckInterval = req.CheckInterval
	m.Timeout = req.Timeout
	m.ExpectedCode = req.ExpectedCode
	m.ExpectedBody = req.ExpectedBody
	if err := s.dao.Update(m); err != nil {
		result.Failed(c, 500, "更新失败: "+err.Error())
		return
	}
	result.Success(c, m)
}

func (s *APIEndpointServiceImpl) Delete(c *gin.Context, id uint) {
	s.dao.Delete(id)
	result.Success(c, nil)
}

func (s *APIEndpointServiceImpl) BatchDelete(c *gin.Context, ids []uint) {
	if len(ids) == 0 {
		result.Failed(c, 400, "请选择要删除的记录")
		return
	}
	s.dao.BatchDelete(ids)
	result.Success(c, nil)
}

func (s *APIEndpointServiceImpl) CheckEndpoint(c *gin.Context, id uint) {
	m, err := s.dao.GetByID(id)
	if err != nil {
		result.Failed(c, 404, "未找到该记录")
		return
	}
	s.checkEndpointInternal(m)
	s.dao.Update(m)
	result.Success(c, m)
}

func (s *APIEndpointServiceImpl) CheckAllEndpoints(c *gin.Context) {
	list, err := s.dao.GetAll()
	if err != nil {
		result.Failed(c, 500, "获取列表失败: "+err.Error())
		return
	}
	for i := range list {
		s.checkEndpointInternal(&list[i])
		s.dao.Update(&list[i])
	}
	result.Success(c, map[string]interface{}{"total": len(list)})
}

// GetAllForEval 获取所有API端点（供规则引擎使用）
func (s *APIEndpointServiceImpl) GetAllForEval() ([]model.MonitorAPIEndpoint, error) {
	return s.dao.GetAll()
}

func (s *APIEndpointServiceImpl) checkEndpointInternal(m *model.MonitorAPIEndpoint) {
	now := time.Now()
	m.CheckTime = now.Format("2006-01-02 15:04:05")
	timeout := time.Duration(m.Timeout) * time.Second
	client := &http.Client{Timeout: timeout}

	var req *http.Request
	var err error
	bodyStr := m.Body
	if bodyStr == "" {
		req, err = http.NewRequest(m.Method, m.URL, nil)
	} else {
		req, err = http.NewRequest(m.Method, m.URL, bytes.NewBufferString(bodyStr))
	}
	if err != nil {
		m.Status = 4
		m.ErrorMsg = fmt.Sprintf("请求构建失败: %v", err)
		m.LastStatusCode = 0
		m.LastResponseTime = 0
		return
	}

	// 设置请求头
	if m.Headers != "" {
		headerMap := make(map[string]string)
		content := m.Headers
		content = strings.TrimSpace(content)
		if strings.HasPrefix(content, "{") && strings.HasSuffix(content, "}") {
			content = content[1 : len(content)-1]
			pairs := strings.Split(content, ",")
			for _, pair := range pairs {
				pair = strings.TrimSpace(pair)
				parts := strings.SplitN(pair, ":", 2)
				if len(parts) == 2 {
					k := strings.Trim(strings.TrimSpace(parts[0]), "\"")
					v := strings.Trim(strings.TrimSpace(parts[1]), "\"")
					headerMap[k] = v
				}
			}
		}
		for k, v := range headerMap {
			req.Header.Set(k, v)
		}
	}
	req.Header.Set("User-Agent", "AutoOps-Monitor/1.0")

	startTime := time.Now()
	resp, err := client.Do(req)
	elapsed := time.Since(startTime).Milliseconds()
	m.LastResponseTime = elapsed

	if err != nil {
		m.Status = 4
		m.LastStatusCode = 0
		m.ErrorMsg = fmt.Sprintf("请求失败: %v", err)
		return
	}
	defer resp.Body.Close()

	m.LastStatusCode = resp.StatusCode

	if elapsed >= int64(m.Timeout)*1000 {
		m.Status = 3
		m.ErrorMsg = fmt.Sprintf("响应超时: %dms > %ds", elapsed, m.Timeout)
		return
	}

	if resp.StatusCode != m.ExpectedCode {
		m.Status = 2
		m.ErrorMsg = fmt.Sprintf("状态码异常: 期望%d, 实际%d", m.ExpectedCode, resp.StatusCode)
		return
	}

	if m.ExpectedBody != "" {
		bodyBytes, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			m.Status = 4
			m.ErrorMsg = fmt.Sprintf("读取响应体失败: %v", err)
			return
		}
		if !strings.Contains(string(bodyBytes), m.ExpectedBody) {
			m.Status = 2
			m.ErrorMsg = fmt.Sprintf("响应体不包含期望内容: %s", m.ExpectedBody)
			return
		}
	}

	m.Status = 1
	m.ErrorMsg = ""
}
