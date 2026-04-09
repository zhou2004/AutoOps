package service

import (
"dodevops-api/api/monitor/dao"
"dodevops-api/api/monitor/model"
)

type MonitorAlertRuleStyleService interface {
CreateStyle(data *model.MonitorAlertRuleStyle) error
DeleteStyle(id uint) error
UpdateStyle(data *model.MonitorAlertRuleStyle) error
GetStyleList() ([]*model.MonitorAlertRuleStyle, error)
}

type monitorAlertRuleStyleService struct {
styleDao dao.MonitorAlertRuleStyleDao
}

func NewMonitorAlertRuleStyleService() MonitorAlertRuleStyleService {
return &monitorAlertRuleStyleService{styleDao: dao.NewMonitorAlertRuleStyleDao()}
}

func (s *monitorAlertRuleStyleService) CreateStyle(data *model.MonitorAlertRuleStyle) error {
return s.styleDao.Create(data)
}

func (s *monitorAlertRuleStyleService) DeleteStyle(id uint) error {
return s.styleDao.Delete(id)
}

func (s *monitorAlertRuleStyleService) UpdateStyle(data *model.MonitorAlertRuleStyle) error {
return s.styleDao.Update(data)
}

func (s *monitorAlertRuleStyleService) GetStyleList() ([]*model.MonitorAlertRuleStyle, error) {
return s.styleDao.GetList()
}
