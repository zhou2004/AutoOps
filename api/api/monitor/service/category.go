package service

import (
	"dodevops-api/api/monitor/dao"
	"dodevops-api/api/monitor/model"
)

type MonitorCategoryService struct{ dao *dao.MonitorCategoryDao }

func NewMonitorCategoryService() *MonitorCategoryService {
	return &MonitorCategoryService{dao: dao.NewMonitorCategoryDao()}
}
func (s *MonitorCategoryService) List(scope string) ([]model.MonitorCategory, error) {
	return s.dao.List(scope)
}
func (s *MonitorCategoryService) Save(req *model.MonitorCategoryReq) error { return s.dao.Save(req) }
func (s *MonitorCategoryService) Delete(scope string, id uint) error       { return s.dao.Delete(scope, id) }
func (s *MonitorCategoryService) Assign(req *model.MonitorCategoryAssignReq) error {
	return s.dao.Assign(req)
}
