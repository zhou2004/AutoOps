package service

import (
	"context"
	"dodevops-api/api/database/model"
	"fmt"
	"os"
	"strconv"
	"testing"
	"time"
)

func exerciseMembership(t *testing.T, ctx context.Context, s *Service, c *model.Cluster, user uint) {
	t.Helper()
	root, e := primary(c)
	if e != nil {
		t.Fatal(e)
	}
	port := 17380
	if c.Engine == "postgresql" {
		port = 15434
	}
	if c.Engine == "mysql" {
		port = 13307
	}
	if override := os.Getenv("AUTOOPS_DB_LIVE_MEMBER_PORT"); override != "" {
		port, e = strconv.Atoi(override)
		if e != nil || (c.Engine == "mysql" && (port < 13307 || port > 13309)) || (c.Engine != "mysql" && (port < 15434 || port > 15436)) {
			t.Fatal("test expansion port outside isolated allowlist")
		}
	}
	key := fmt.Sprintf("membership-%d", time.Now().UnixNano())
	database, table := "postgres", "public.autoops_membership_probe"
	if c.Engine == "mysql" {
		database = "autoops_database_lab"
		table = "autoops_membership_probe"
	}
	if c.Engine == "redis" {
		r, e := s.redis(c, *root)
		if e != nil {
			t.Fatal(e)
		}
		e = r.Set(ctx, key, "verified", 0).Err()
		r.Close()
		if e != nil {
			t.Fatal(e)
		}
	} else {
		db, e := s.sql(ctx, c, *root, database)
		if e != nil {
			t.Fatal(e)
		}
		_, e = db.ExecContext(ctx, "CREATE TABLE IF NOT EXISTS "+table+" (id varchar(191) PRIMARY KEY)")
		if e == nil {
			placeholder := "$1"
			if c.Engine == "mysql" {
				placeholder = "?"
			}
			_, e = db.ExecContext(ctx, "INSERT INTO "+table+" VALUES ("+placeholder+")", key)
		}
		db.Close()
		if e != nil {
			t.Fatal(e)
		}
	}
	if os.Getenv("AUTOOPS_DB_LIVE_MEMBERSHIP") == "resume" {
		var pending *model.Member
		for i, m := range c.Members {
			if m.HostID == 531 && m.Port == port && m.BaselinePending {
				pending = &c.Members[i]
			}
		}
		if pending == nil {
			t.Fatal("no owned incomplete baseline")
		}
		runLiveOperation(t, ctx, s, c, user, "rebuild-replica", Parameters{MemberID: pending.ID})
	} else {
		runLiveOperation(t, ctx, s, c, user, "add-node", Parameters{CandidateID: root.ID, Members: []MemberInput{{HostID: 531, Port: port, Role: "replica"}}})
	}
	fresh, e := s.Refresh(ctx, c.ID)
	if e != nil {
		t.Fatal(e)
	}
	var added *model.Member
	for i, m := range fresh.Members {
		if m.HostID == 531 && m.Port == port {
			added = &fresh.Members[i]
		}
	}
	if added == nil || added.Role != "replica" || added.State != "healthy" {
		t.Fatal("new replica not healthy")
	}
	e = waitUntil(ctx, 30*time.Second, func() bool {
		if c.Engine == "redis" {
			r, e := s.redis(fresh, *added)
			if e != nil {
				return false
			}
			defer r.Close()
			v, e := r.Get(ctx, key).Result()
			return e == nil && v == "verified"
		}
		db, e := s.sql(ctx, fresh, *added, database)
		if e != nil {
			return false
		}
		defer db.Close()
		var n int
		placeholder := "$1"
		if c.Engine == "mysql" {
			placeholder = "?"
		}
		e = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM "+table+" WHERE id="+placeholder, key).Scan(&n)
		return e == nil && n == 1
	})
	if e != nil {
		t.Fatal("new replica has no baseline fixture", e)
	}
	t.Logf("%s new member %d replication and fixture verified", c.Engine, added.ID)
	runLiveOperation(t, ctx, s, fresh, user, "remove-node", Parameters{MemberID: added.ID})
	s.event(c.ID, user, 0, "integration-report", "info", fmt.Sprintf("新增副本 #%d 的真实基线与复制数据通过验证，随后通过受控退役移出", added.ID))
}
