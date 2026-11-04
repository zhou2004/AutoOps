package service

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"dodevops-api/api/database/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Real child processes share a private on-disk metadata database. Killing the
// executor must preserve its uncertain job and lock across a process restart.
func TestDatabaseWorkerProcessRestart(t *testing.T) {
	if path := os.Getenv("AUTOOPS_TEST_WORKER_PROCESS_DB"); path != "" {
		db, e := gorm.Open(sqlite.Open(path+"?_busy_timeout=5000"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
		if e != nil {
			t.Fatal(e)
		}
		raw, _ := db.DB()
		raw.SetMaxOpenConns(1)
		defer raw.Close()
		New(db).RunWorker(context.Background(), fmt.Sprintf("test-child-%d", os.Getpid()))
		return
	}
	s := testService(t)
	var paths []struct{ File string }
	if e := s.DB.Raw("PRAGMA database_list").Scan(&paths).Error; e != nil || len(paths) == 0 {
		t.Fatal("metadata path", e)
	}
	var calls, unexpected atomic.Int32
	reached := make(chan struct{}, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/health" {
			unexpected.Add(1)
		}
		if calls.Add(1) > 1 {
			// A pending-result lock now permits read-only collection. It does
			// not permit the old task to be claimed or executed again.
			http.Error(w, "test health unavailable", http.StatusServiceUnavailable)
			return
		}
		select {
		case reached <- struct{}{}:
		default:
		}
		<-r.Context().Done()
	}))
	defer server.Close()
	host, port, _ := net.SplitHostPort(strings.TrimPrefix(server.URL, "http://"))
	p, _ := strconv.Atoi(port)
	s.DB.Where("cluster_id = ?", 1).Delete(&model.Member{})
	if e := s.DB.Create(&model.Member{ClusterID: 1, Kind: "etcd", Address: host, Port: p, Metrics: JSON(map[string]any{})}).Error; e != nil {
		t.Fatal(e)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	plan, e := s.Plan(ctx, 1, 1, "refresh", Parameters{})
	if e != nil || !plan.Executable {
		t.Fatal("plan", e)
	}
	first, e := s.Submit(ctx, 1, 1, plan.ID, "process-crash-first", "one")
	if e != nil {
		t.Fatal(e)
	}
	// This test kills an executing worker, not a collector reading the queued
	// cluster. A fresh sample defers the initial collector until after claim.
	s.DB.Model(&model.Cluster{}).Where("id = ?", 1).Update("collected_at", time.Now())
	start := func() *exec.Cmd {
		cmd := exec.Command(os.Args[0], "-test.run=^TestDatabaseWorkerProcessRestart$")
		cmd.Env = append(os.Environ(), "AUTOOPS_TEST_WORKER_PROCESS_DB="+paths[0].File)
		if e := cmd.Start(); e != nil {
			t.Fatal(e)
		}
		t.Cleanup(func() { _ = cmd.Process.Kill(); _ = cmd.Wait() })
		return cmd
	}
	child := start()
	select {
	case <-reached:
	case <-ctx.Done():
		t.Fatal("worker did not contact controlled endpoint")
	}
	if e = child.Process.Kill(); e != nil {
		t.Fatal(e)
	}
	_ = child.Wait()
	// Advance the test's heartbeat fixture, not the clock or any real worker's lease.
	s.DB.Model(&model.Operation{}).Where("id = ?", first.ID).Update("heartbeat_at", time.Now().Add(-2*time.Minute))
	s.reconcileAbandoned()
	if e = s.DB.First(first, first.ID).Error; e != nil || first.Status != "unknown" {
		t.Fatal("crashed task not uncertain", e)
	}
	previousWorker := first.WorkerID
	previousHeartbeat := *first.HeartbeatAt
	var locks int64
	s.DB.Model(&model.OperationLock{}).Where("cluster_id = ? AND operation_id = ?", 1, first.ID).Count(&locks)
	if locks != 1 {
		t.Fatal("crashed operation lost its exclusion lock")
	}
	plan, e = s.Plan(ctx, 2, 1, "adopt", Parameters{Management: "operate"})
	if e != nil || !plan.Executable {
		t.Fatal("second plan", e)
	}
	second, e := s.Submit(ctx, 2, 1, plan.ID, "process-restart-second", "two")
	if e != nil {
		t.Fatal(e)
	}
	// The fresh timestamp above only isolates the first execution. Make this
	// fixture due for collection when the replacement process starts.
	s.DB.Model(&model.Cluster{}).Where("id = ?", 1).Update("collected_at", time.Now().Add(-time.Minute))
	_ = start()
	if e = waitUntil(ctx, 8*time.Second, func() bool { s.DB.First(second, second.ID); return second.Status == "succeeded" }); e != nil {
		t.Fatal("new worker did not execute persisted queue", e)
	}
	s.DB.First(first, first.ID)
	if first.Status != "unknown" || first.WorkerID != previousWorker || !first.HeartbeatAt.Equal(previousHeartbeat) || unexpected.Load() != 0 {
		t.Fatal("restarted worker replayed an uncertain request")
	}
	if e = waitUntil(ctx, 8*time.Second, func() bool { return calls.Load() > 1 }); e != nil {
		t.Fatal("read-only health collection did not resume for uncertain task", e)
	}
	s.DB.Model(&model.OperationLock{}).Where("cluster_id = ? AND operation_id = ?", 1, first.ID).Count(&locks)
	if locks != 1 {
		t.Fatal("read-only collection released the uncertain operation lock")
	}
	t.Log("SIGKILL and new process: persisted queue and read-only collection resumed; uncertain operation was not reclaimed and its lock remained")
}
