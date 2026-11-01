package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"dodevops-api/api/database/model"
)

const remoteQuiescence = `import json,sys,pathlib,os
p=json.load(sys.stdin);busy=[]
for proc in pathlib.Path('/proc').glob('[0-9]*'):
 try:
  if int(proc.name) in (os.getpid(),os.getppid()):continue
  cmd=(proc/'cmdline').read_bytes()
 except (OSError,ValueError):continue
 if (b'Ansiball'+b'Z_') in cmd or (b'pgback'+b'rest') in cmd and (b'backup' in cmd or b'restore' in cmd):busy.append(proc.name)
print(json.dumps({'idle':not busy,'activeProcessCount':len(busy)}))`

func (s *Service) reconcileOperation(ctx context.Context, c *model.Cluster, op model.Operation, p Parameters) error {
	var previous model.Operation
	if s.DB.Where("id = ? AND cluster_id = ? AND worker_id = ?", p.OperationID, c.ID, fmt.Sprintf("superseded-by-%d", op.ID)).First(&previous).Error != nil {
		return ErrConflict
	}
	chain := []uint{previous.ID}
	for previous.Action == "reconcile" {
		if len(chain) > 64 {
			return errors.New("任务核验链过长")
		}
		var previousParams Parameters
		if json.Unmarshal(previous.Parameters, &previousParams) != nil {
			return errors.New("原任务参数损坏")
		}
		previous = model.Operation{}
		if s.DB.Where("id = ? AND cluster_id = ?", previousParams.OperationID, c.ID).First(&previous).Error != nil {
			return ErrConflict
		}
		chain = append(chain, previous.ID)
	}
	if e := s.Authorize(op.UserID, c.ID, actionPermission(previous.Action)); e != nil {
		return e
	}
	var original Parameters
	if json.Unmarshal(previous.Parameters, &original) != nil {
		return errors.New("原任务参数损坏")
	}
	hosts := map[uint]bool{}
	for _, m := range c.Members {
		if m.HostID != 0 {
			hosts[m.HostID] = true
		}
	}
	if original.HostID > 0 {
		hosts[original.HostID] = true
	}
	for _, m := range original.Members {
		if m.HostID > 0 {
			hosts[m.HostID] = true
		}
	}
	if len(hosts) == 0 {
		return errors.New("无法通过受管主机证明旧执行已结束，保留互斥锁")
	}
	if e := s.step(op, "推进远端执行世代并确认旧命令结束", func(log func(string)) error {
		root, e := automationRoot()
		if e != nil {
			return e
		}
		dir, e := os.MkdirTemp("", "autoops-reconcile-")
		if e != nil {
			return e
		}
		defer os.RemoveAll(dir)
		if _, _, e = s.guardedAutomation(ctx, c, hosts, root, dir); e != nil {
			return e
		}
		for host := range hosts {
			raw, e := s.remoteJSON(ctx, host, remoteQuiescence, map[string]any{})
			if e != nil || !flag(object(raw)["idle"]) {
				return errors.New("仍检测到远端安装、备份或恢复进程，不能接手")
			}
		}
		log("远端排他锁已确认旧模块结束；较旧任务不能再启动新模块")
		return nil
	}); e != nil {
		return e
	}
	// Retired data sources preserve audit IDs. A stopped, disabled service must
	// not reappear through ORM association writes from an older cluster snapshot.
	for _, m := range c.Members {
		if m.SourceID == 0 {
			continue
		}
		var retired bool
		if s.DB.Table("cmdb_sql").Where("id = ?", m.SourceID).Pluck("retired", &retired).Error == nil && retired {
			raw, e := s.remoteJSON(ctx, m.HostID, `import json,sys,subprocess
p=json.load(sys.stdin)
r=subprocess.run(['systemctl','show',p['service'],'-p','ActiveState','-p','MainPID','-p','UnitFileState'],check=True,capture_output=True,text=True)
s=dict(x.split('=',1) for x in r.stdout.splitlines() if '=' in x)
assert s.get('ActiveState') in ('inactive','failed') and s.get('MainPID')=='0' and s.get('UnitFileState')=='disabled'
print(json.dumps({'retired':True}))`, m)
			if e != nil || !flag(object(raw)["retired"]) {
				return errors.New("退役实例的服务停止状态未确认")
			}
			if e = s.DB.Delete(&m).Error; e != nil {
				return e
			}
		}
	}
	if previous.Action == "add-node" && c.Engine == "mysql" {
		for _, in := range original.Members {
			s.DB.Model(&model.Member{}).Where("cluster_id = ? AND host_id = ? AND port = ? AND source_id = 0", c.ID, in.HostID, in.Port).Update("baseline_pending", true)
		}
	}
	fresh, e := s.Refresh(ctx, c.ID)
	if e != nil {
		return e
	}
	c = fresh
	h := object(c.Health)
	resolvedStatus := "succeeded"
	resolvedMessage := fmt.Sprintf("实际结果由核验任务 #%d 确认，未重放原切换或恢复命令", op.ID)
	switch previous.Action {
	case "configure-vip", "verify-vip":
		if c.VIP == nil {
			return errors.New("缺少 VIP 登记，不能确认旧部署状态")
		}
		if e := s.collectVIP(ctx, c); e != nil {
			return e
		}
		if c.VIP.Status == "unknown" || c.VIP.Status == "conflict" {
			return errors.New("VIP 绑定未知或冲突，保留执行锁")
		}
		if previous.Action == "verify-vip" && c.VIP.Status != "healthy" {
			return errors.New("漂移演练原服务尚未恢复，请等待独立恢复定时器并检查服务")
		}
		resolvedStatus = "failed"
		resolvedMessage = "旧 VIP 执行已结束、两端绑定状态已核对；保留已安装的独立服务，请重新预检配置或漂移验证，不重放故障注入"

	case "update-patroni":
		if h["database"] != "healthy" || h["replication"] != "healthy" || h["control"] != "healthy" {
			return errors.New("控制组件更新尚未收敛，请先恢复服务与复制；不重放切换或重启")
		}
		for _, m := range c.Members {
			if m.Kind == "patroni" && m.Version != "4.1.5" {
				resolvedStatus = "failed"
				resolvedMessage = "旧更新进程已结束且复制健康，控制版本尚未全部更新；请生成新的更新计划"
			}
		}
	case "configure-fencing":
		// This only reapplies declarative hook material after proving that the
		// old installer exited; it never invokes the fencing/recovery command.
		if e = s.configureFencing(ctx, c, op); e != nil {
			return e
		}
	case "configure-semisync":
		for _, m := range c.Members {
			if m.Kind != "data" {
				continue
			}
			if m.State != "healthy" || (m.Role == "replica" && (!flag(object(m.Metrics)["read_only"]) || !flag(object(m.Metrics)["super_read_only"]))) {
				return errors.New("需先确认副本只读保护，保留现场")
			}
		}
		resolvedStatus = "failed"
		resolvedMessage = "半同步安装执行已结束、服务与只读保护正常；未证明全部参数和应答协商完成，请生成新的配置计划"
	case "balance-primaries":
		if h["database"] != "healthy" || h["replication"] != "healthy" {
			return errors.New("原生切换尚未收敛，不能重复分布主节点")
		}
		if len(placementWarnings(c)) > 0 {
			resolvedStatus = "failed"
			resolvedMessage = "原生切换已收敛，当前主节点分布尚不满足故障域保护；请基于实际拓扑重新预检"
		}
	case "enter-maintenance", "exit-maintenance":
		if c.Engine == "mysql" {
			if e = s.waitMySQLRecovery(ctx, c); e != nil {
				return e
			}
		}
		resolvedStatus = "failed"
		resolvedMessage = "原维护请求已结束；请查看原生维护状态并重新提交明确的进入或退出计划，不重放中断请求"
	case "provision":
		if h["database"] != "healthy" || h["replication"] == "degraded" || h["control"] == "degraded" {
			return errors.New("数据或控制节点未完成安装；保留现场，请修复步骤日志所示问题后重新核验")
		}
		proxies := []model.Member{}
		for _, m := range c.Members {
			if m.Kind == "proxy" {
				if !m.Managed {
					return errors.New("不能覆盖接管代理配置")
				}
				proxies = append(proxies, m)
			}
		}
		if len(proxies) > 0 {
			if e = s.step(op, "完成已登记的独立代理", func(log func(string)) error { _, e := s.ansible(ctx, c, proxies, "endpoint.yml", nil, log); return e }); e != nil {
				return e
			}
		}
		if e = s.registerSources(c); e != nil {
			return e
		}
		if e = s.waitEndpoints(ctx, c); e != nil {
			return e
		}
	case "add-node":
		complete := true
		for _, in := range original.Members {
			found := false
			for _, m := range c.Members {
				if m.Kind == "data" && m.HostID == in.HostID && m.Port == in.Port {
					found = m.State == "healthy" && (m.Role == "replica" || c.Topology == "redis-cluster" && in.Role == "primary" && m.Role == "primary")
					if found {
						s.DB.Model(&model.Member{}).Where("id = ?", m.ID).Update("baseline_pending", false)
					}
				}
			}
			complete = complete && found
		}
		if complete {
			if e = s.registerSources(c); e != nil {
				return e
			}
		} else {
			resolvedStatus = "failed"
			resolvedMessage = "旧新增节点执行已停止，基线尚未建立；保留独立节点与目录，MySQL 可通过重建副本计划完成基线，未重放初始化"
		}
	case "remove-node":
		if _, err := member(c, original.MemberID); err == nil {
			resumed := op
			resumed.Action = "remove-node"
			if e = s.membership(ctx, c, resumed, original); e != nil {
				return e
			}
		}
	case "reshard":
		if c.Topology != "redis-cluster" {
			return errors.New("原任务不属于 Redis Cluster")
		}
		// The previous worker cannot submit another batch after ownership changed.
		// Wait out a request already sent (MIGRATE timeout is 5 seconds), then
		// inspect Redis itself. This also works after a crash without FinishedAt.
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(10 * time.Second):
		}
		for _, m := range c.Members {
			if m.Kind != "data" || m.Role != "primary" {
				continue
			}
			client, e := s.redis(c, m)
			if e != nil {
				return e
			}
			raw, e := client.ClientList(ctx).Result()
			client.Close()
			if e != nil {
				return e
			}
			if strings.Contains(raw, "cmd=migrate ") {
				return errors.New("Redis 仍在执行 MIGRATE，不能接手")
			}
		}
		if e = s.reshard(ctx, c, op, original); e != nil {
			return e
		}
	case "switchover", "recover":
		if previous.Action == "recover" && original.CandidateID > 0 {
			if e = s.waitRecovered(ctx, c.ID, original.CandidateID); e != nil {
				return e
			}
			if e = s.waitEndpoints(ctx, c); e != nil {
				return e
			}
			break
		}
		if h["replication"] != "healthy" || h["database"] != "healthy" {
			return errors.New("角色或复制尚未收敛，不重放切换请求；请检查原生管理器事件")
		}
		if original.CandidateID > 0 {
			m, e := member(c, original.CandidateID)
			if e != nil || m.Role != "primary" {
				return errors.New("原定候选未成为主库，不能确认原切换成功")
			}
		}
		if e = s.waitEndpoints(ctx, c); e != nil {
			return e
		}
	case "rejoin":
		m, err := member(c, original.MemberID)
		if err != nil {
			return err
		}
		if m.Role == "replica" && m.State == "healthy" && h["database"] == "healthy" && h["replication"] == "healthy" {
			break
		}
		for _, node := range c.Members {
			if node.Kind == "data" && (node.State != "healthy" || !flag(object(node.Metrics)["read_only"]) || !flag(object(node.Metrics)["super_read_only"])) {
				return errors.New("重加入未收敛；需确认所有数据节点均处于保护性只读才能结束失败任务")
			}
		}
		resolvedStatus = "failed"
		resolvedMessage = fmt.Sprintf("核验任务 #%d 确认旧执行结束且所有节点只读；复制拓扑未收敛，需独立 GTID 修复计划", op.ID)
	case "resume-writes":
		root, err := primary(c)
		if err != nil || root.State != "healthy" || h["replication"] != "healthy" {
			return errors.New("复制根节点或复制状态尚未收敛")
		}
		if flag(object(root.Metrics)["read_only"]) {
			resolvedStatus = "failed"
			resolvedMessage = fmt.Sprintf("核验任务 #%d 确认主库仍为保护性只读，未恢复写入；可重新预检", op.ID)
		} else if e = s.waitEndpoints(ctx, c); e != nil {
			return e
		}
	case "configure-endpoint":
		if e = s.waitEndpoints(ctx, c); e != nil {
			// The exclusive remote epoch above proves no old installer can still
			// change these services. Keep installed proxies and explicitly record
			// failed routing verification; a fresh repair plan can now proceed.
			resolvedStatus = "failed"
			resolvedMessage = fmt.Sprintf("核验任务 #%d 确认代理配置执行已停止，但入口角色检查未通过；保留代理服务，请修复数据库角色或生成新的入口计划", op.ID)
		}
	case "backup":
		var b model.Backup
		if s.DB.Where("operation_id = ?", previous.ID).First(&b).Error != nil {
			return errors.New("缺少备份任务记录，不能核验结果")
		}
		if b.Status != "complete" {
			// Remote exclusive epochs and process inspection prove the old backup
			// executor has stopped. Preserve its files, never advertise them as usable.
			if e = s.DB.Model(&b).Update("status", "failed").Error; e != nil {
				return e
			}
			resolvedStatus = "failed"
			resolvedMessage = fmt.Sprintf("核验任务 #%d 确认远端备份执行已结束，但没有完整备份清单；保留现场，可生成新的备份计划", op.ID)
		}
	case "restore", "verify-backup":
		var b model.Backup
		if s.DB.Where("id = ? AND cluster_id = ?", original.BackupID, c.ID).First(&b).Error != nil {
			return errors.New("备份记录缺失，无法核验隔离恢复")
		}
		var manifest struct {
			Members []model.Member `json:"members"`
		}
		if json.Unmarshal(b.Manifest, &manifest) != nil || len(manifest.Members) == 0 {
			return errors.New("备份成员清单无效")
		}
		reports := map[string]json.RawMessage{}
		for _, m := range manifest.Members {
			raw, err := s.remoteJSON(ctx, original.HostID, `import json,sys,pathlib,subprocess
p=json.load(sys.stdin);unit='autoops-restore-'+str(p['operation'])+'-'+str(p['member'])
r=subprocess.run(['systemctl','show',unit,'-p','ActiveState','-p','MainPID'],capture_output=True,text=True,check=True)
state=dict(line.split('=',1) for line in r.stdout.splitlines() if '=' in line)
assert state.get('ActiveState') in ('inactive','failed') and state.get('MainPID')=='0','restore instance still running'
root=pathlib.Path('/var/lib/autoops-database-restores')/str(p['cluster'])/str(p['operation'])/str(p['member'])
report=json.load(open(root/'report.json')) if (root/'report.json').exists() else None
if report:assert report.get('path')==str(root) and report.get('service')==unit and report.get('state')=='stopped-after-verification'
print(json.dumps({'stopped':True,'report':report}))`, map[string]any{"cluster": c.ID, "operation": previous.ID, "member": m.ID})
			if err != nil {
				return errors.New("隔离恢复实例尚未停止或报告身份不符，保留执行锁")
			}
			report := JSON(object(raw)["report"])
			if flag(object(report)["verified"]) {
				reports[fmt.Sprintf("member_%d", m.ID)] = report
			}
		}
		if len(reports) != len(manifest.Members) {
			resolvedStatus = "failed"
			resolvedMessage = fmt.Sprintf("核验任务 #%d 确认隔离恢复进程已停止，但缺少本次完整验证报告；保留恢复目录，可创建新的隔离恢复计划", op.ID)
		} else {
			verification := object(b.Manifest)
			verification["restoreReport"] = reports
			if e = s.DB.Model(&b).Updates(map[string]any{"verified_at": nowPtr(), "manifest": JSON(verification)}).Error; e != nil {
				return e
			}
			if e = s.DB.Model(&model.Cluster{}).Where("id = ?", c.ID).Update("restore_verified_at", nowPtr()).Error; e != nil {
				return e
			}
		}
	case "expire-backup":
		var b model.Backup
		if s.DB.Where("id = ? AND cluster_id = ?", original.BackupID, c.ID).First(&b).Error != nil {
			return errors.New("备份记录不存在")
		}
		if b.Status != "expired" {
			if e = s.expireBackup(ctx, c, op, original); e != nil {
				return e
			}
		}
	case "configure":
		// After proving the old executor ended, inspect actual values. A partial
		// change is a failed operation with visible drift, never blindly replayed.
		if e = s.waitConfiguration(ctx, configurationScope(c, original.MemberID), original.Changes); e != nil {
			resolvedStatus = "failed"
			resolvedMessage = "旧参数执行已结束，但期望值未全部生效；保留配置差异，请重新预检"
		}
		s.collectConfigurationStatus(ctx, c)
	case "restart", "rebuild-replica":
		if h["database"] != "healthy" || h["replication"] != "healthy" {
			return errors.New("实例或复制未收敛，保留现场")
		}
		// Healthy state proves that maintenance can continue, not that an
		// interrupted installer necessarily ran its intended restart/rebuild.
		resolvedStatus = "failed"
		resolvedMessage = "旧执行已结束，实例与复制已恢复健康；无法证明中断步骤完整完成，可重新生成维护计划"
	case "assign-replica":
		node, err := member(c, original.MemberID)
		target, err2 := member(c, original.CandidateID)
		if err != nil || err2 != nil || node.Role != "replica" || node.ParentID != target.NativeID || node.State != "healthy" {
			return errors.New("分片副本关系尚未收敛，不重放配置")
		}
	case "refresh", "collect-storage":
		// Both actions are read-only and safe to collect again.
		readOp := op
		readOp.Action = previous.Action
		if e = s.execute(ctx, c, readOp, original); e != nil {
			return e
		}
	default:
		return errors.New("该操作需按具体步骤核对结果；当前保留执行锁与现场，不进行通用回滚或重放")
	}
	return s.DB.Model(&model.Operation{}).Where("id IN ? AND status = ?", chain, "reconciling").Updates(map[string]any{"status": resolvedStatus, "message": resolvedMessage, "finished_at": time.Now().UTC()}).Error
}
