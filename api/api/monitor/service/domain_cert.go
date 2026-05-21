package service

import (
	"crypto/tls"
	"fmt"
	"math"
	"net"
	"strconv"
	"strings"
	"time"

	"dodevops-api/api/monitor/dao"
	"dodevops-api/api/monitor/model"
	"dodevops-api/common/result"

	"github.com/gin-gonic/gin"
)

type DomainCertServiceInterface interface {
	GetList(c *gin.Context, req *model.DomainCertListReq)
	GetByID(c *gin.Context, id uint)
	Add(c *gin.Context, req *model.DomainCertAddReq)
	Update(c *gin.Context, req *model.DomainCertUpdateReq)
	Delete(c *gin.Context, id uint)
	BatchDelete(c *gin.Context, ids []uint)
	CheckCert(c *gin.Context, id uint)
	CheckAllCerts(c *gin.Context)
}

type DomainCertServiceImpl struct {
	dao *dao.DomainCertDao
}

func NewDomainCertService() DomainCertServiceInterface {
	return &DomainCertServiceImpl{
		dao: dao.NewDomainCertDao(),
	}
}

func (s *DomainCertServiceImpl) GetList(c *gin.Context, req *model.DomainCertListReq) {
	list, total, err := s.dao.GetList(req)
	if err != nil {
		result.Failed(c, 500, "查询列表失败: "+err.Error())
		return
	}
	result.Success(c, map[string]interface{}{"list": list, "total": total})
}

func (s *DomainCertServiceImpl) GetByID(c *gin.Context, id uint) {
	cert, err := s.dao.GetByID(id)
	if err != nil {
		result.Failed(c, 404, "未找到该域名记录")
		return
	}
	result.Success(c, cert)
}

func (s *DomainCertServiceImpl) Add(c *gin.Context, req *model.DomainCertAddReq) {
	domain := strings.TrimSpace(req.Domain)
	if domain == "" {
		result.Failed(c, 400, "域名不能为空")
		return
	}
	existing, _ := s.dao.GetByDomain(domain)
	if existing != nil {
		result.Failed(c, 400, "该域名已存在")
		return
	}
	port := req.Port
	if port <= 0 {
		port = 443
	}
	now := time.Now().Format("2006-01-02 15:04:05")
	cert := &model.DomainCert{
		Domain: domain, Port: port, Status: 1,
		RemainingDays: -1, CheckTime: now,
	}
	if err := s.dao.Create(cert); err != nil {
		result.Failed(c, 500, "添加失败: "+err.Error())
		return
	}
	s.checkCertInternal(cert)
	s.dao.Update(cert)
	result.Success(c, cert)
}

func (s *DomainCertServiceImpl) Update(c *gin.Context, req *model.DomainCertUpdateReq) {
	cert, err := s.dao.GetByID(req.ID)
	if err != nil {
		result.Failed(c, 404, "未找到该域名记录")
		return
	}
	domain := strings.TrimSpace(req.Domain)
	if domain == "" {
		result.Failed(c, 400, "域名不能为空")
		return
	}
	if domain != cert.Domain {
		existing, _ := s.dao.GetByDomain(domain)
		if existing != nil && existing.ID != req.ID {
			result.Failed(c, 400, "该域名已存在")
			return
		}
	}
	port := req.Port
	if port <= 0 {
		port = 443
	}
	cert.Domain = domain
	cert.Port = port
	if err := s.dao.Update(cert); err != nil {
		result.Failed(c, 500, "更新失败: "+err.Error())
		return
	}
	result.Success(c, cert)
}

func (s *DomainCertServiceImpl) Delete(c *gin.Context, id uint) {
	s.dao.Delete(id)
	result.Success(c, nil)
}

func (s *DomainCertServiceImpl) BatchDelete(c *gin.Context, ids []uint) {
	if len(ids) == 0 {
		result.Failed(c, 400, "请选择要删除的记录")
		return
	}
	s.dao.BatchDelete(ids)
	result.Success(c, nil)
}

func (s *DomainCertServiceImpl) CheckCert(c *gin.Context, id uint) {
	cert, err := s.dao.GetByID(id)
	if err != nil {
		result.Failed(c, 404, "未找到该域名记录")
		return
	}
	s.checkCertInternal(cert)
	s.dao.Update(cert)
	result.Success(c, s.toCheckResult(cert))
}

func (s *DomainCertServiceImpl) CheckAllCerts(c *gin.Context) {
	list, err := s.dao.GetAll()
	if err != nil {
		result.Failed(c, 500, "获取域名列表失败: "+err.Error())
		return
	}
	var results []model.DomainCertCheckResult
	for i := range list {
		s.checkCertInternal(&list[i])
		s.dao.Update(&list[i])
		results = append(results, s.toCheckResult(&list[i]))
	}
	result.Success(c, map[string]interface{}{"total": len(list), "results": results})
}

// GetAllForEval 获取所有域名证书记录（供规则引擎使用）
func (s *DomainCertServiceImpl) GetAllForEval() ([]model.DomainCert, error) {
	return s.dao.GetAll()
}

func (s *DomainCertServiceImpl) checkCertInternal(cert *model.DomainCert) {
	now := time.Now()
	cert.CheckTime = now.Format("2006-01-02 15:04:05")
	addr := net.JoinHostPort(cert.Domain, strconv.Itoa(cert.Port))

	conn, err := tls.DialWithDialer(
		&net.Dialer{Timeout: 10 * time.Second},
		"tcp", addr,
		&tls.Config{InsecureSkipVerify: true},
	)
	if err != nil {
		cert.Status = 4
		cert.ErrorMsg = fmt.Sprintf("连接失败: %v", err)
		cert.RemainingDays = -1
		return
	}
	defer conn.Close()

	state := conn.ConnectionState()
	if len(state.PeerCertificates) == 0 {
		cert.Status = 4
		cert.ErrorMsg = "未获取到服务器证书"
		cert.RemainingDays = -1
		return
	}

	peerCert := state.PeerCertificates[0]
	cert.Issuer = peerCert.Issuer.CommonName
	if cert.Issuer == "" && len(peerCert.Issuer.Organization) > 0 {
		cert.Issuer = peerCert.Issuer.Organization[0]
	}
	cert.Subject = peerCert.Subject.CommonName
	cert.NotBefore = peerCert.NotBefore.Format("2006-01-02 15:04:05")
	cert.NotAfter = peerCert.NotAfter.Format("2006-01-02 15:04:05")

	remaining := peerCert.NotAfter.Sub(now)
	remainingDays := int(math.Ceil(remaining.Hours() / 24))

	if remainingDays < 0 {
		cert.RemainingDays = 0
		cert.Status = 3
		cert.ErrorMsg = fmt.Sprintf("证书已于 %s 过期", cert.NotAfter)
	} else if remainingDays <= 30 {
		cert.RemainingDays = remainingDays
		cert.Status = 2
		cert.ErrorMsg = fmt.Sprintf("证书将在 %d 天后过期(%s)", remainingDays, cert.NotAfter)
	} else {
		cert.RemainingDays = remainingDays
		cert.Status = 1
		cert.ErrorMsg = ""
	}
}

func (s *DomainCertServiceImpl) toCheckResult(cert *model.DomainCert) model.DomainCertCheckResult {
	return model.DomainCertCheckResult{
		Domain: cert.Domain, Port: cert.Port, Issuer: cert.Issuer,
		Subject: cert.Subject, NotBefore: cert.NotBefore, NotAfter: cert.NotAfter,
		RemainingDays: cert.RemainingDays, Status: cert.Status, ErrorMsg: cert.ErrorMsg,
	}
}