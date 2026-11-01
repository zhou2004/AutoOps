package service

import (
	"context"
	cmdb "dodevops-api/api/cmdb/model"
	cmdbservice "dodevops-api/api/cmdb/service"
	"dodevops-api/api/database/model"
	"encoding/json"
	"fmt"
	"gorm.io/gorm"
	"os"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

func exerciseSlots(t *testing.T, ctx context.Context, s *Service, c *model.Cluster, user uint) {
	t.Helper()
	labPort := 18382
	if os.Getenv("AUTOOPS_DB_LIVE_SLOTS") == "fresh" {
		labPort = 18383
	}
	if os.Getenv("AUTOOPS_DB_LIVE_SLOTS") == "cleanup" {
		for _, m := range c.Members {
			if m.Port == labPort {
				runLiveOperation(t, ctx, s, c, user, "remove-node", Parameters{MemberID: m.ID})
				c, _ = s.Refresh(ctx, c.ID)
			}
		}
		var source cmdb.CmdbSQL
		s.DB.Where("cluster_id = ? AND connection_mode = ?", c.ID, "cluster").First(&source)
		client, e := cmdbservice.NewDatabaseOperationService(s.DB).OpenRedis(source.ID)
		if e != nil {
			t.Fatal(e)
		}
		defer client.Close()
		checked := 0
		for _, m := range c.Members {
			if m.Kind != "data" || m.Role != "primary" {
				continue
			}
			direct, e := s.redis(c, m)
			if e != nil {
				t.Fatal(e)
			}
			var cursor uint64
			for {
				keys, next, e := direct.Scan(ctx, cursor, "{slot-fixture-*}:migration-*", 100).Result()
				if e != nil {
					t.Fatal(e)
				}
				for _, key := range keys {
					value, e := client.Get(ctx, key).Result()
					want := key[strings.LastIndex(key, "-")+1:]
					if e != nil || value != want {
						t.Fatal("migration fixture corrupted")
					}
					if _, e = strconv.Atoi(value); e != nil {
						t.Fatal(e)
					}
					checked++
				}
				cursor = next
				if next == 0 {
					break
				}
			}
			direct.Close()
		}
		if checked < 402 {
			t.Fatalf("only %d of 402 migration fixture values remain", checked)
		}
		t.Logf("interrupted return migration reconciled, scale-in completed, %d fixture values verified", checked)
		return
	}
	root, e := primary(c)
	if e != nil {
		t.Fatal(e)
	}
	original := *root
	// Three additional independent processes, with one replica in each other domain.
	var added model.Member
	for _, m := range c.Members {
		if m.HostID == 530 && m.Port == labPort {
			added = m
		}
	}
	if added.ID == 0 {
		runLiveOperation(t, ctx, s, c, user, "add-node", Parameters{CandidateID: root.ID, Members: []MemberInput{{HostID: 530, Port: labPort, Role: "primary", Kind: "data"}}})
		c, _ = s.Refresh(ctx, c.ID)
		for _, m := range c.Members {
			if m.HostID == 530 && m.Port == labPort {
				added = m
			}
		}
	}
	if added.ID == 0 {
		t.Fatal("new shard identity missing")
	}
	for _, host := range []uint{528, 531} {
		exists := false
		for _, m := range c.Members {
			if m.HostID == host && m.Port == labPort {
				exists = true
			}
		}
		if !exists {
			runLiveOperation(t, ctx, s, c, user, "add-node", Parameters{CandidateID: added.ID, Members: []MemberInput{{HostID: host, Port: labPort, Role: "replica", Kind: "data"}}})
			c, _ = s.Refresh(ctx, c.ID)
		}
	}
	c, _ = s.Refresh(ctx, c.ID)
	var ranges []string
	json.Unmarshal(JSON(object(original.Metrics)["slots"]), &ranges)
	slots := []int{}
	for _, r := range ranges {
		if strings.HasPrefix(r, "[") {
			continue
		}
		var lo, hi int
		if _, e := fmt.Sscanf(r, "%d-%d", &lo, &hi); e != nil {
			continue
		}
		if hi > lo {
			slots = []int{lo, lo + 1}
			break
		}
	}
	if len(slots) != 2 {
		t.Fatal("no stable range on original shard")
	}
	var source cmdb.CmdbSQL
	s.DB.Where("cluster_id = ? AND connection_mode = ?", c.ID, "cluster").First(&source)
	client, e := cmdbservice.NewDatabaseOperationService(s.DB).OpenRedis(source.ID)
	if e != nil {
		t.Fatal(e)
	}
	defer client.Close()
	direct, e := s.redis(c, original)
	if e != nil {
		t.Fatal(e)
	}
	defer direct.Close()
	keys := map[string]string{}
	for _, slot := range slots {
		tag := ""
		for n := 0; n < 100000; n++ {
			v := fmt.Sprintf("slot-fixture-%d", n)
			if direct.ClusterKeySlot(ctx, "{"+v+"}").Val() == int64(slot) {
				tag = v
				break
			}
		}
		if tag == "" {
			t.Fatal("hash tag search failed")
		}
		for i := 0; i < 201; i++ {
			key := fmt.Sprintf("{%s}:migration-%d-%d", tag, time.Now().UnixNano(), i)
			value := fmt.Sprint(i)
			if e = client.Set(ctx, key, value, 0).Err(); e != nil {
				t.Fatal(e)
			}
			keys[key] = value
		}
	}
	c, _ = s.Refresh(ctx, c.ID)
	p, e := s.Plan(ctx, c.ID, user, "reshard", Parameters{MemberID: original.ID, CandidateID: added.ID, Slots: slots})
	if e != nil || !p.Executable {
		t.Fatal(e, p)
	}
	op, e := s.Submit(ctx, c.ID, user, p.ID, fmt.Sprintf("interrupt-slot-%d", time.Now().UnixNano()), c.Name)
	if e != nil {
		t.Fatal(e)
	}
	op.WorkerID = "integration-slot-interruption"
	s.DB.Model(op).Updates(map[string]any{"worker_id": op.WorkerID, "status": "preflight", "heartbeat_at": time.Now()})
	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	var once sync.Once
	callback := "autoops-test-slot-interrupt"
	if e = s.DB.Callback().Create().After("gorm:create").Register(callback, func(tx *gorm.DB) {
		if event, ok := tx.Statement.Dest.(*model.Event); ok && event.OperationID == op.ID && event.Kind == "slot-migrated" {
			once.Do(cancel)
		}
	}); e != nil {
		t.Fatal(e)
	}
	s.run(runCtx, *op)
	s.DB.Callback().Create().Remove(callback)
	var stopped model.Operation
	s.DB.First(&stopped, op.ID)
	if stopped.Status != "unknown" {
		t.Fatalf("interruption did not retain unknown task: %s", stopped.Status)
	}
	runLiveOperation(t, ctx, s, c, user, "reconcile", Parameters{OperationID: op.ID})
	for key, want := range keys {
		got, e := client.Get(ctx, key).Result()
		if e != nil || got != want {
			t.Fatal("key changed during interrupted migration")
		}
	}
	c, _ = s.Refresh(ctx, c.ID)
	runLiveOperation(t, ctx, s, c, user, "reshard", Parameters{MemberID: added.ID, CandidateID: original.ID, Slots: slots})
	c, _ = s.Refresh(ctx, c.ID)
	// Reassign replicas before removing their now-empty parent.
	for _, m := range c.Members {
		if m.Port != labPort || m.Role != "replica" {
			continue
		}
		var destination uint
		for _, n := range c.Members {
			if n.Kind == "data" && n.Role == "primary" && n.ID != added.ID && n.HostID != m.HostID {
				destination = n.ID
				break
			}
		}
		if destination == 0 {
			t.Fatal("no distinct domain for replica")
		}
		runLiveOperation(t, ctx, s, c, user, "assign-replica", Parameters{MemberID: m.ID, CandidateID: destination})
		c, _ = s.Refresh(ctx, c.ID)
	}
	runLiveOperation(t, ctx, s, c, user, "remove-node", Parameters{MemberID: added.ID})
	c, _ = s.Refresh(ctx, c.ID)
	for _, m := range c.Members {
		if m.Port == labPort {
			runLiveOperation(t, ctx, s, c, user, "remove-node", Parameters{MemberID: m.ID})
			c, _ = s.Refresh(ctx, c.ID)
		}
	}
	for key, want := range keys {
		got, e := client.Get(ctx, key).Result()
		if e != nil || got != want {
			t.Fatal("key missing after scale-in")
		}
		client.Del(ctx, key)
	}
	t.Logf("new shard with two replicas, interrupted two-slot migration, actual-state reconciliation and scale-in passed; verified %d values", len(keys))
}
