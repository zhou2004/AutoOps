package service

import (
	"dodevops-api/api/database/model"
	"testing"
)

func TestCapabilitiesIncludeEnglishLabelsWithoutChangingActions(t *testing.T) {
	s := testService(t)
	for action := range operationLabels {
		if operationLabelsEnglish[action] == "" {
			t.Errorf("missing English display label for %s", action)
		}
	}
	var cluster model.Cluster
	if err := s.DB.Preload("Members").First(&cluster, 1).Error; err != nil {
		t.Fatal(err)
	}
	capabilities := s.Capabilities(1, &cluster)
	for _, capability := range capabilities {
		if capability.Label != operationLabels[capability.Action] || capability.LabelEn != operationLabelsEnglish[capability.Action] || capability.LabelEn == "" {
			t.Errorf("inconsistent localized labels: %+v", capability)
		}
		if capability.Permission == "" {
			t.Errorf("capability lost permission: %s", capability.Action)
		}
	}
}
