package config

import "fmt"

// Development's usual `go run main.go` must start both halves of the database
// task system. Deployed API replicas explicitly use external workers.
func EmbeddedDatabaseWorker(setting, serverMode string) (bool, error) {
	switch setting {
	case "", "auto":
		return serverMode == "debug", nil
	case "embedded":
		return true, nil
	case "external":
		return false, nil
	default:
		return false, fmt.Errorf("DATABASE_WORKER_MODE/--database-worker 必须为 auto、embedded 或 external")
	}
}
