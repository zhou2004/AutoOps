package service

import (
	"dodevops-api/api/webservice/model"
	"testing"
)

func TestSitePublicationReflectsActualMembersAndRollback(t *testing.T) {
	s := fixture(t)
	spec := model.SiteSpec{Mode: "raw", Path: "/etc/nginx/site.conf", Raw: "server { listen 8080; }"}
	r := model.Resource{Name: "site", GroupID: 10, ClusterID: 1, Revision: 1, Status: "draft", Spec: JSON(spec)}
	if err := s.DB.Table(model.Tables["sites"]).Create(&r).Error; err != nil {
		t.Fatal(err)
	}
	matching := JSON(Observation{Files: []model.File{{Path: spec.Path, Content: spec.Raw}}})
	members := []model.Resource{{Observed: matching}, {Observed: JSON(Observation{})}}
	check := func(expected string) {
		t.Helper()
		s.refreshPublicationState(1, members)
		actual, err := s.Get("sites", r.ID)
		if err != nil || actual.Status != expected {
			t.Fatal(actual.Status, expected, err)
		}
	}
	check("partial")
	members[1].Observed = matching
	check("published")
	members[0].Observed, members[1].Observed = nil, nil
	s.DB.Create(&model.Lock{Key: "cluster:1", OperationID: 1})
	check("published") // Do not report intermediate deployment state as final.
	s.DB.Delete(&model.Lock{}, "`key` = ?", "cluster:1")
	check("draft") // A rollback to a revision without this file removes publication.
}
