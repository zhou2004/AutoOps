package service

import (
	"context"
	"dodevops-api/api/webservice/model"
	"sort"
	"strings"
)

// Only expose existing environment values from resources visible to this user.
func (s *Service) Environments(ctx context.Context, user uint) ([]string, error) {
	if err := s.Module(ctx, user, "read"); err != nil {
		return nil, err
	}
	values := map[string]bool{}
	for kind, table := range model.Tables {
		if (kind == "dns-accounts" || kind == "acme-accounts") && !s.Admin(user) {
			continue
		}
		var rows []model.Resource
		if err := s.DB.WithContext(ctx).Table(table).Select("id,source_id,cluster_id,group_id,environment,spec").Where("environment <> ?", "").Find(&rows).Error; err != nil {
			return nil, err
		}
		for _, row := range rows {
			if s.Authorize(user, kind, row, readAction(kind)) == nil && strings.TrimSpace(row.Environment) != "" {
				values[row.Environment] = true
			}
		}
	}
	result := make([]string, 0, len(values))
	for value := range values {
		result = append(result, value)
	}
	sort.Strings(result)
	return result, nil
}
