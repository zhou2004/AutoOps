package service

import (
	"context"
	cmdb "dodevops-api/api/cmdb/model"
	"dodevops-api/api/webservice/model"
	"dodevops-api/common/util"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"
)

type vipRuntime struct {
	requests   []RemoteRequest
	state      map[uint]VIPObservation
	failID     uint
	failSwitch bool
	journals   map[uint]map[uint]string
}

func (f *vipRuntime) Run(_ context.Context, r RemoteRequest) (json.RawMessage, error) {
	f.requests = append(f.requests, r)
	if r.Action == "discover" {
		return JSON(Observation{Digest: "base", Identity: "nginx", Files: []model.File{{Path: "/etc/nginx/nginx.conf", Content: "events {} http {}"}}}), nil
	}
	if !strings.HasPrefix(r.Action, "vip-") {
		return JSON(map[string]any{"status": 200}), nil
	}
	if f.journals == nil {
		f.journals = map[uint]map[uint]string{}
	}
	if f.journals[r.OperationID] == nil {
		f.journals[r.OperationID] = map[uint]string{}
	}
	id := r.VIP.InstanceID
	o := f.state[id]
	o.Healthy = true
	o.Address = fmt.Sprintf("192.168.80.%d", 50+id)
	o.MAC = fmt.Sprintf("00:00:00:00:00:%02d", id)
	switch r.Action {
	case "vip-reconcile":
		state := f.journals[r.OperationID][id]
		if state == "" && o.Owned {
			return nil, errors.New("VIP service exists without this operation journal")
		}
		if state == "" {
			state = "not-started"
		}
		o.State = state
	case "vip-rollback":
		o.Owned = false
		o.Active = false
		o.Bound = false
		o.ConfigDigest = ""
		o.State = "rolled-back"
	case "vip-configure":
		if id == f.failID {
			return nil, errors.New("lost VIP response")
		}
		f.journals[r.OperationID][id] = "succeeded"
		o.Owned = true
		o.Active = true
		o.ConfigDigest = "vip-config"
		hasOwner := false
		for _, v := range f.state {
			hasOwner = hasOwner || v.Bound
		}
		o.Bound = !hasOwner
	case "vip-switch":
		f.journals[r.OperationID][id] = "stopped"
		o.Active = false
		o.Bound = false
		for _, next := range r.VIP.Config.InstanceIDs {
			if next == id {
				continue
			}
			v := f.state[next]
			v.Bound = true
			f.state[next] = v
			break
		}
	case "vip-resume":
		o.Active = true
	}
	f.state[id] = o
	if r.Action == "vip-switch" && f.failSwitch {
		return nil, errors.New("switch response lost")
	}
	if o.Bound {
		o.ARPOwners = []string{o.MAC}
	}
	return JSON(o), nil
}
func vipFixture(t *testing.T) (*Service, *vipRuntime, *model.VIPSpec) {
	s := fixture(t)
	if err := s.DB.AutoMigrate(&cmdb.CmdbHost{}, &cmdb.CmdbAssetConnection{}); err != nil {
		t.Fatal(err)
	}
	f := &vipRuntime{state: map[uint]VIPObservation{}}
	s.Remote = f
	for i := uint(1); i <= 3; i++ {
		if err := s.DB.Create(&cmdb.CmdbHost{ID: i, HostName: fmt.Sprint("node-", i), GroupID: 10, SSHIP: fmt.Sprintf("192.168.80.%d", 50+i), CreateTime: util.HTime{Time: time.Now()}}).Error; err != nil {
			t.Fatal(err)
		}
		if err := s.DB.Create(&cmdb.CmdbAssetConnection{ID: i, AssetType: "host", AssetID: i, Protocol: "ssh", Address: fmt.Sprintf("192.168.80.%d", 50+i), Enabled: true}).Error; err != nil {
			t.Fatal(err)
		}
		spec := model.SourceSpec{HostID: i, ConnectionID: i, Runtime: "host", Binary: "/usr/sbin/nginx", Prefix: "/etc/nginx", ConfigPath: "/etc/nginx/nginx.conf", HealthURL: "http://127.0.0.1:18080/__autoops_health", ExpectedStatus: 200}
		source := model.Resource{ID: i, Name: fmt.Sprint("node-", i), GroupID: 10, Revision: 1, Spec: JSON(spec)}
		instance := model.Resource{ID: i, Name: source.Name, GroupID: 10, Revision: 1, SourceID: i, ClusterID: 1, Management: "operate", Spec: source.Spec}
		if i == 1 {
			s.DB.Table(model.Tables["sources"]).Where("id = ?", i).Updates(map[string]any{"spec": source.Spec})
			s.DB.Table(model.Tables["instances"]).Where("id = ?", i).Updates(map[string]any{"spec": source.Spec, "management": "operate"})
		} else {
			s.DB.Table(model.Tables["sources"]).Create(&source)
			s.DB.Table(model.Tables["instances"]).Create(&instance)
		}
	}
	p := &model.VIPSpec{CIDR: "192.168.80.110/24", Interface: "ens33", RouterID: 151, InstanceIDs: []uint{1, 2, 3}, PreferredInstanceID: 1}
	return s, f, p
}
func TestVIPValidationRejectsAmbiguousAndUnsafeTargets(t *testing.T) {
	_, _, base := vipFixture(t)
	if err := validateVIP(base); err != nil {
		t.Fatal(err)
	}
	cases := []model.VIPSpec{*base, *base, *base, *base, *base, *base, *base}
	cases[0].CIDR = "192.168.80.0/24"
	cases[1].CIDR = "192.168.80.255/24"
	cases[2].Interface = "ens33\nother"
	cases[3].InstanceIDs = []uint{1, 1}
	cases[4].RouterID = 256
	cases[5].CIDR = "127.0.0.1/24"
	cases[6].PreferredInstanceID = 99
	for _, p := range cases {
		if validateVIP(&p) == nil {
			t.Fatalf("accepted invalid VIP %#v", p)
		}
	}
}
func TestVIPPlanDeploySwitchAndImmutableConfiguration(t *testing.T) {
	s, f, p := vipFixture(t)
	ctx := context.Background()
	plan, err := s.Plan(ctx, 1, PlanInput{TargetType: "clusters", TargetID: 1, Action: "configure-vip", Parameters: Parameters{VIP: p}})
	if err != nil || !plan.Executable {
		t.Fatalf("plan %v %s", err, plan.Checks)
	}
	for _, r := range f.requests {
		if r.Action == "vip-configure" {
			t.Fatal("plan mutated remote state")
		}
	}
	op, err := s.Submit(ctx, 1, plan.ID, plan.IdempotencyKey, true)
	if err != nil {
		t.Fatal(err)
	}
	s.execute(ctx, op)
	s.DB.First(&op, op.ID)
	if op.Status != "succeeded" {
		t.Fatalf("deploy %s %s", op.Status, op.Message)
	}
	c, _ := s.Get("clusters", 1)
	spec, _ := decode[model.ClusterSpec](c.Spec)
	if spec.VIP == nil || len(spec.VIP.InstanceIDs) != 3 {
		t.Fatal("VIP not persisted")
	}
	c.Spec = JSON(model.ClusterSpec{VIP: &model.VIPSpec{CIDR: "192.168.80.111/24"}})
	if _, err = s.Save(ctx, 1, "clusters", c, ""); err == nil {
		t.Fatal("ordinary save changed deployed VIP")
	}
	plan, err = s.Plan(ctx, 1, PlanInput{TargetType: "clusters", TargetID: 1, Action: "switch-vip"})
	if err != nil || !plan.Executable {
		t.Fatalf("switch plan %v %s", err, plan.Checks)
	}
	op, err = s.Submit(ctx, 1, plan.ID, plan.IdempotencyKey, true)
	if err != nil {
		t.Fatal(err)
	}
	s.execute(ctx, op)
	s.DB.First(&op, op.ID)
	if op.Status != "succeeded" {
		t.Fatalf("switch %s %s", op.Status, op.Message)
	}
	if !f.state[2].Bound || !f.state[1].Active {
		t.Fatal("old service was not restored as backup")
	}
	for _, r := range f.requests {
		if r.Action == "stop" || r.Action == "reload" || r.Action == "restart" {
			t.Fatal("VIP changed nginx lifecycle")
		}
	}
}
func TestVIPLostResponseRetainsClusterAndAddressLocks(t *testing.T) {
	s, f, p := vipFixture(t)
	f.failID = 2
	ctx := context.Background()
	plan, err := s.Plan(ctx, 1, PlanInput{TargetType: "clusters", TargetID: 1, Action: "configure-vip", Parameters: Parameters{VIP: p}})
	if err != nil || !plan.Executable {
		t.Fatal(err, string(plan.Checks))
	}
	op, err := s.Submit(ctx, 1, plan.ID, plan.IdempotencyKey, true)
	if err != nil {
		t.Fatal(err)
	}
	s.execute(ctx, op)
	s.DB.First(&op, op.ID)
	if op.Status != "unknown" {
		t.Fatal(op.Status, op.Message)
	}
	var count int64
	s.DB.Model(&model.Lock{}).Where("operation_id = ?", op.ID).Count(&count)
	if count != 3 {
		t.Fatalf("expected cluster/address/VRID locks, got %d", count)
	}
	for _, r := range f.requests {
		if r.Action == "vip-configure" && r.VIP.InstanceID == 3 {
			t.Fatal("continued batch after uncertain response")
		}
	}
}
func TestVIPRejectsNonAdminAndSameHostMembers(t *testing.T) {
	s, _, p := vipFixture(t)
	ctx := context.Background()
	c, _ := s.Get("clusters", 1)
	if err := s.prepareVIP(ctx, 2, c, PlanInput{Action: "configure-vip", Parameters: Parameters{VIP: p}}, &[]Target{}); !errors.Is(err, ErrForbidden) {
		t.Fatal(err)
	}
	source, _ := s.Get("sources", 2)
	sp, _ := decode[model.SourceSpec](source.Spec)
	sp.HostID = 1
	s.DB.Table(model.Tables["sources"]).Where("id = ?", 2).Update("spec", JSON(sp))
	plan, err := s.Plan(ctx, 1, PlanInput{TargetType: "clusters", TargetID: 1, Action: "configure-vip", Parameters: Parameters{VIP: p}})
	if err != nil || plan.Executable {
		t.Fatalf("same-host members accepted: %v %s", err, plan.Checks)
	}
}

func TestVIPUnknownRecoveryRollsBackOnlyFrozenDeployment(t *testing.T) {
	s, f, p := vipFixture(t)
	f.failID = 2
	ctx := context.Background()
	plan, err := s.Plan(ctx, 1, PlanInput{TargetType: "clusters", TargetID: 1, Action: "configure-vip", Parameters: Parameters{VIP: p}})
	if err != nil || !plan.Executable {
		t.Fatal(err, string(plan.Checks))
	}
	op, err := s.Submit(ctx, 1, plan.ID, plan.IdempotencyKey, true)
	if err != nil {
		t.Fatal(err)
	}
	s.execute(ctx, op)
	recovery, err := s.Plan(ctx, 1, PlanInput{TargetType: "clusters", TargetID: 1, Action: "reconcile", Parameters: Parameters{OperationID: op.ID}})
	if err != nil || !recovery.Executable {
		t.Fatal(err, string(recovery.Checks))
	}
	job, err := s.Submit(ctx, 1, recovery.ID, recovery.IdempotencyKey, true)
	if err != nil {
		t.Fatal(err)
	}
	s.execute(ctx, job)
	s.DB.First(&op, op.ID)
	s.DB.First(&job, job.ID)
	if op.Status != "rolled-back" || job.Status != "succeeded" {
		t.Fatalf("%s/%s: %s", op.Status, job.Status, job.Message)
	}
	var locks int64
	s.DB.Model(&model.Lock{}).Count(&locks)
	if locks != 0 {
		t.Fatal("recovery retained locks")
	}
	for _, state := range f.state {
		if state.Active || state.Bound {
			t.Fatal("VIP service survived rollback")
		}
	}
}
func TestVIPRecoveryRejectsChangedSourceBeforeRemoteResume(t *testing.T) {
	s, f, p := vipFixture(t)
	f.failID = 2
	ctx := context.Background()
	plan, err := s.Plan(ctx, 1, PlanInput{TargetType: "clusters", TargetID: 1, Action: "configure-vip", Parameters: Parameters{VIP: p}})
	if err != nil || !plan.Executable {
		t.Fatal(err)
	}
	op, err := s.Submit(ctx, 1, plan.ID, plan.IdempotencyKey, true)
	if err != nil {
		t.Fatal(err)
	}
	s.execute(ctx, op)
	recovery, err := s.Plan(ctx, 1, PlanInput{TargetType: "clusters", TargetID: 1, Action: "reconcile", Parameters: Parameters{OperationID: op.ID}})
	if err != nil || !recovery.Executable {
		t.Fatal(err)
	}
	job, err := s.Submit(ctx, 1, recovery.ID, recovery.IdempotencyKey, true)
	if err != nil {
		t.Fatal(err)
	}
	s.DB.Table(model.Tables["sources"]).Where("id = ?", 1).Update("revision", 9)
	f.requests = nil
	s.execute(ctx, job)
	s.DB.First(&job, job.ID)
	if job.Status != "failed" {
		t.Fatal(job.Status)
	}
	for _, r := range f.requests {
		if r.Action == "vip-resume" || r.Action == "vip-rollback" {
			t.Fatal("recovery acted on changed source")
		}
	}
	var locks int64
	s.DB.Model(&model.Lock{}).Where("operation_id = ?", op.ID).Count(&locks)
	if locks != 3 {
		t.Fatal("original locks lost")
	}
}

func TestVIPSwitchAndVerifyRecoveryOnlyRequireMutatedNodeJournal(t *testing.T) {
	for _, action := range []string{"switch-vip", "verify-vip"} {
		t.Run(action, func(t *testing.T) {
			s, f, p := vipFixture(t)
			ctx := context.Background()
			plan, err := s.Plan(ctx, 1, PlanInput{TargetType: "clusters", TargetID: 1, Action: "configure-vip", Parameters: Parameters{VIP: p}})
			if err != nil || !plan.Executable {
				t.Fatal(err)
			}
			deployed, err := s.Submit(ctx, 1, plan.ID, plan.IdempotencyKey, true)
			if err != nil {
				t.Fatal(err)
			}
			s.execute(ctx, deployed)
			plan, err = s.Plan(ctx, 1, PlanInput{TargetType: "clusters", TargetID: 1, Action: action})
			if err != nil || !plan.Executable {
				t.Fatal(err, string(plan.Checks))
			}
			op, err := s.Submit(ctx, 1, plan.ID, plan.IdempotencyKey, true)
			if err != nil {
				t.Fatal(err)
			}
			if action == "switch-vip" {
				f.failSwitch = true
				s.execute(ctx, op)
			} else {
				s.DB.Model(&op).Update("status", "unknown")
			}
			recovery, err := s.Plan(ctx, 1, PlanInput{TargetType: "clusters", TargetID: 1, Action: "reconcile", Parameters: Parameters{OperationID: op.ID}})
			if err != nil || !recovery.Executable {
				t.Fatal(err, string(recovery.Checks))
			}
			job, err := s.Submit(ctx, 1, recovery.ID, recovery.IdempotencyKey, true)
			if err != nil {
				t.Fatal(err)
			}
			f.requests = nil
			s.execute(ctx, job)
			s.DB.First(&job, job.ID)
			s.DB.First(&op, op.ID)
			if job.Status != "succeeded" || op.Status != "succeeded" {
				t.Fatalf("%s/%s: %s", job.Status, op.Status, job.Message)
			}
			for _, r := range f.requests {
				if r.Action == "vip-reconcile" && (action == "verify-vip" || r.VIP.InstanceID != 1) {
					t.Fatal("required a journal from an unchanged node")
				}
			}
		})
	}
}
