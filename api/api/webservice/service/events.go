package service

import (
	"context"
	"dodevops-api/common"
	"fmt"
	"time"
)

// Redis is only a wake-up signal; authorization and durable evidence stay in SQL.
func OperationChannel(id uint) string { return fmt.Sprintf("autoops:web:operation:%d", id) }
func (s *Service) notifyOperation(id uint) {
	if client := common.GetRedisClient(); client != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		defer cancel()
		_ = client.Publish(ctx, OperationChannel(id), fmt.Sprint(id)).Err()
	}
}
