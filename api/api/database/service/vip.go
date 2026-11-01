package service

import (
	"context"
	cmdbdao "dodevops-api/api/cmdb/dao"
	cmdbmodel "dodevops-api/api/cmdb/model"
	"dodevops-api/api/database/model"
	"encoding/json"
	"errors"
	"fmt"
	"gorm.io/gorm"
	"net"
	"regexp"
	"time"
)

type VIPInput struct {
	CIDR      string `json:"cidr"`
	Interface string `json:"interface"`
	RouterID  int    `json:"routerId"`
	MemberIDs []uint `json:"memberIds"`
}

var interfacePattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_.-]{0,14}$`)

func (s *Service) vipMembers(c *model.Cluster, p VIPInput) ([]model.Member, error) {
	ip, subnet, e := net.ParseCIDR(p.CIDR)
	if e != nil || ip.To4() == nil || ip.IsLoopback() || ip.IsUnspecified() || !ip.IsGlobalUnicast() || !interfacePattern.MatchString(p.Interface) || p.RouterID < 1 || p.RouterID > 255 || len(p.MemberIDs) != 2 || p.MemberIDs[0] == p.MemberIDs[1] {
		return nil, errors.New("填写有效 IPv4/CIDR、网卡、VRID（1–255）及两个不同代理")
	}
	ones, _ := subnet.Mask.Size()
	if ones < 8 || ones > 30 || ip.Equal(subnet.IP) {
		return nil, errors.New("VIP 必须为可用主机地址，前缀范围 8–30")
	}
	broadcast := append(net.IP(nil), subnet.IP.To4()...)
	for i := range broadcast {
		broadcast[i] |= ^subnet.Mask[i]
	}
	if ip.Equal(broadcast) {
		return nil, errors.New("VIP 不能是广播地址")
	}
	out := []model.Member{}
	seen := map[uint]bool{}
	for _, id := range p.MemberIDs {
		m, e := member(c, id)
		if e != nil || m.Kind != "proxy" || !m.Managed || !unitPattern.MatchString(m.Service) || m.HostID == 0 || seen[m.HostID] || !subnet.Contains(net.ParseIP(m.Address)) || ip.Equal(net.ParseIP(m.Address)) {
			return nil, errors.New("VIP 需要同网段、不同主机的两个受管 SQL 代理")
		}
		seen[m.HostID] = true
		out = append(out, *m)
	}
	if out[0].Port != out[1].Port {
		return nil, errors.New("两个代理的业务端口必须相同")
	}
	var occupied int64
	if s.DB.Model(&cmdbmodel.CmdbHost{}).Where("private_ip = ? OR public_ip = ? OR ssh_ip = ?", ip.String(), ip.String(), ip.String()).Count(&occupied).Error != nil {
		return nil, errors.New("CMDB 地址冲突检查失败")
	}
	if occupied > 0 {
		return nil, errors.New("VIP 与 CMDB 主机地址冲突")
	}
	var registered []model.VirtualIP
	s.DB.Find(&registered)
	for _, v := range registered {
		if v.ClusterID == c.ID {
			if v.CIDR != p.CIDR || v.Interface != p.Interface || v.RouterID != p.RouterID || string(v.MemberIDs) != string(JSON(p.MemberIDs)) {
				return nil, errors.New("已部署的 VIP 不允许直接覆盖；请先按教程退役原配置")
			}
			continue
		}
		old, _, _ := net.ParseCIDR(v.CIDR)
		if ip.Equal(old) || v.RouterID == p.RouterID {
			return nil, errors.New("VIP 或 VRID 已被另一集群登记")
		}
	}
	return out, nil
}

const inspectVIPScript = `import json,sys,subprocess,ipaddress,socket,struct,time,pathlib,re
p=json.load(sys.stdin);iface=p['interface'];target=ipaddress.ip_interface(p['cidr']);ip=str(target.ip)
r=json.loads(subprocess.check_output(['ip','-j','-4','address','show','dev',iface]))
assert len(r)==1 and 'UP' in r[0]['flags'],'interface unavailable'
r[0]['address']=json.loads(subprocess.check_output(['ip','-j','link','show','dev',iface]))[0]['address']
addresses=[a['local'] for a in r[0]['addr_info'] if a['family']=='inet']
assert p['address'] in addresses and ipaddress.ip_address(p['address']) in target.network,'interface/subnet mismatch'
unit='autoops-db-vip-'+str(p['cluster'])
active=subprocess.run(['systemctl','is-active','--quiet',unit],stdout=subprocess.DEVNULL).returncode==0
owners=[]
if p.get('probe'):
 for port in [p['port'],p['port']+1]:
  listening=subprocess.check_output(['ss','-H','-ltn','sport','=',':'+str(port)],text=True)
  for line in listening.splitlines():
   local=line.split()[3]
   assert local in [p['address']+':'+str(port),'0.0.0.0:'+str(port),'*:'+str(port)],'another address already uses the proxy port'
 mac=bytes.fromhex(r[0]['address'].replace(':',''));sock=socket.socket(socket.AF_PACKET,socket.SOCK_RAW,socket.htons(0x0806));sock.bind((iface,0));sock.settimeout(.2)
 packet=b'\xff'*6+mac+struct.pack('!H',0x0806)+struct.pack('!HHBBH',1,0x0800,6,4,1)+mac+socket.inet_aton(p['address'])+b'\x00'*6+socket.inet_aton(ip)
 for retry in range(3):
  sock.send(packet);end=time.monotonic()+.7
  while time.monotonic()<end:
   try:data=sock.recv(2048)
   except socket.timeout:continue
   if len(data)>=42 and data[12:14]==b'\x08\x06' and data[28:32]==socket.inet_aton(ip):owners.append(':'.join('%02x'%b for b in data[22:28]))
 sock.close()
 # Never overwrite an externally owned configuration or join a known VRID.
 config=pathlib.Path('/etc/autoops-database-vip/'+str(p['cluster'])+'/keepalived.conf')
 if config.exists():assert '# AutoOps cluster '+str(p['cluster'])+'\n' in config.read_text(),'configuration ownership mismatch'
 for root in ['/etc/keepalived','/etc/autoops-database-vip']:
  for f in pathlib.Path(root).rglob('*.conf'):
   if f==config:continue
   assert not re.search(r'virtual_router_id\s+'+str(p['routerId'])+r'\b',f.read_text(errors='ignore')),'VRID already configured'
print(json.dumps({'bound':ip in addresses,'active':active,'interface':iface,'mac':r[0]['address'],'arpOwners':sorted(set(owners)),'addresses':addresses}))`

func (s *Service) inspectVIP(ctx context.Context, c *model.Cluster, p VIPInput, m model.Member, probe bool) (map[string]any, error) {
	raw, e := s.remoteJSON(ctx, m.HostID, inspectVIPScript, map[string]any{"cluster": c.ID, "cidr": p.CIDR, "interface": p.Interface, "routerId": p.RouterID, "address": m.Address, "probe": probe, "port": m.Port})
	return object(raw), e
}
func (s *Service) vipChecks(ctx context.Context, c *model.Cluster, p VIPInput, user uint) error {
	members, e := s.vipMembers(c, p)
	if e != nil {
		return e
	}
	for _, m := range members {
		if !s.IsAdmin(user) {
			ok, e := cmdbdao.NewCmdbAssetPermissionDao(s.DB).CanUserAccess(user, "host", m.HostID, "command")
			if e != nil || !ok {
				return ErrForbidden
			}
		}
	}
	observations := []map[string]any{}
	allowedMAC := map[string]bool{}
	for _, m := range members {
		r, e := s.inspectVIP(ctx, c, p, m, true)
		if e != nil {
			return errors.New("VIP 网卡、地址冲突或服务归属预检失败：" + e.Error())
		}
		observations = append(observations, r)
		if flag(r["bound"]) {
			if c.VIP == nil || !flag(r["active"]) {
				return errors.New("VIP 已绑定到未登记或未运行的服务")
			}
			allowedMAC[text(r["mac"])] = true
		}
	}
	for _, r := range observations {
		if a, ok := r["arpOwners"].([]any); ok {
			for _, v := range a {
				if !allowedMAC[text(v)] {
					return errors.New("ARP 检测发现 VIP 已被占用，禁止部署")
				}
			}
		}
	}
	return nil
}
func vipParameters(v *model.VirtualIP) VIPInput {
	p := VIPInput{CIDR: v.CIDR, Interface: v.Interface, RouterID: v.RouterID}
	_ = json.Unmarshal(v.MemberIDs, &p.MemberIDs)
	return p
}
func (s *Service) collectVIP(ctx context.Context, c *model.Cluster) error {
	if c.VIP == nil {
		return nil
	}
	v := c.VIP
	p := vipParameters(v)
	members, e := s.vipMembers(c, p)
	if e != nil {
		return e
	}
	state := "healthy"
	owners := []uint{}
	observations := map[string]any{}
	for _, m := range members {
		probe, cancel := context.WithTimeout(ctx, 5*time.Second)
		r, err := s.inspectVIP(probe, c, p, m, false)
		cancel()
		if err != nil {
			state = "unknown"
			observations[fmt.Sprint(m.ID)] = map[string]any{"error": err.Error()}
			continue
		}
		observations[fmt.Sprint(m.ID)] = r
		if flag(r["bound"]) {
			owners = append(owners, m.ID)
		}
		if !flag(r["active"]) && state != "unknown" {
			state = "degraded"
		}
	}
	if len(owners) > 1 {
		state = "conflict"
	} else if len(owners) == 0 && state != "unknown" {
		state = "unavailable"
	}
	owner := uint(0)
	if len(owners) == 1 {
		owner = owners[0]
	}
	address, _, _ := net.ParseCIDR(v.CIDR)
	normalized := address.String()
	v.Address = &normalized
	v.Status = state
	v.OwnerMemberID = owner
	v.CollectedAt = nowPtr()
	v.Observation = JSON(observations)
	return s.DB.Model(v).Select("Address", "Status", "OwnerMemberID", "CollectedAt", "Observation").Updates(v).Error
}
func (s *Service) configureVIP(ctx context.Context, c *model.Cluster, op model.Operation, p VIPInput) error {
	members, e := s.vipMembers(c, p)
	if e != nil {
		return e
	}
	return s.step(op, "部署独立 Keepalived 并验证唯一绑定和 SQL 主库入口", func(log func(string)) error {
		v := model.VirtualIP{ClusterID: c.ID, CIDR: p.CIDR, Interface: p.Interface, RouterID: p.RouterID, MemberIDs: JSON(p.MemberIDs), Status: "unknown", Observation: JSON(map[string]any{}), LastDrill: JSON(map[string]any{})}
		if c.VIP == nil {
			address, _, _ := net.ParseCIDR(p.CIDR)
			normalized := address.String()
			v.Address = &normalized
			if e := s.DB.Create(&v).Error; e != nil {
				return e
			}
			c.VIP = &v
		} else {
			v = *c.VIP
		}
		ip, _, _ := net.ParseCIDR(p.CIDR)
		deploymentLog := ""
		if _, e := s.ansible(ctx, c, members, "vip.yml", map[string]any{"vip": p, "vip_ip": ip.String(), "vip_proxies": members}, func(v string) { deploymentLog = v; log(v) }); e != nil {
			return e
		}
		if e := s.waitVIP(ctx, c, 0, 30*time.Second); e != nil {
			return e
		}
		// Preserve the direct dual-proxy endpoints and register a dedicated VIP endpoint.
		ep := model.Endpoint{ClusterID: c.ID, Name: c.Name + "/VIP", Mode: "read-write", Addresses: JSON([]string{net.JoinHostPort(ip.String(), fmt.Sprint(members[0].Port))}), Status: "healthy", VerifiedAt: nowPtr()}
		for _, m := range c.Members {
			if m.Kind == "data" {
				ep.AccountID = m.AccountID
				ep.TLSMode = m.TLSMode
				break
			}
		}
		if e := s.DB.Transaction(func(tx *gorm.DB) error {
			var existing model.Endpoint
			if tx.Where("cluster_id = ? AND name = ?", c.ID, ep.Name).First(&existing).Error == nil {
				ep.ID = existing.ID
			}
			if e := tx.Save(&ep).Error; e != nil {
				return e
			}
			return tx.Model(&model.Cluster{}).Where("id = ?", c.ID).Update("revision", gorm.Expr("revision + 1")).Error
		}); e != nil {
			return e
		}
		s.DB.Where("cluster_id = ?", c.ID).Find(&c.Endpoints)
		log(deploymentLog + "\nVIP 唯一绑定且通过该地址查询得到可写主库；漂移演练需单独执行")
		return s.registerSources(c)
	})
}
func (s *Service) verifyVIPSQL(ctx context.Context, c *model.Cluster) error {
	p := vipParameters(c.VIP)
	members, e := s.vipMembers(c, p)
	if e != nil {
		return e
	}
	ip, _, _ := net.ParseCIDR(p.CIDR)
	target := model.Member{Kind: "data", Address: ip.String(), Port: members[0].Port}
	for _, m := range c.Members {
		if m.Kind == "data" {
			target.AccountID = m.AccountID
			target.TLSMode = m.TLSMode
			break
		}
	}
	probe, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	conn, e := s.sql(probe, c, target, "")
	if e != nil {
		return e
	}
	defer conn.Close()
	query := "SELECT @@global.read_only"
	if c.Engine == "postgresql" {
		query = "SELECT pg_is_in_recovery()"
	}
	var readonly bool
	if e = conn.QueryRowContext(probe, query).Scan(&readonly); e != nil {
		return e
	}
	if readonly {
		return errors.New("VIP 路由至只读节点")
	}
	return nil
}
func (s *Service) waitVIP(ctx context.Context, c *model.Cluster, previous uint, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	successes := 0
	for {
		e := s.collectVIP(ctx, c)
		v := c.VIP
		if e == nil && v.OwnerMemberID > 0 && v.Status != "conflict" && v.Status != "unknown" && (previous == 0 || v.OwnerMemberID != previous) && s.verifyVIPSQL(ctx, c) == nil {
			successes++
		} else {
			successes = 0
		}
		if successes >= 3 {
			return nil
		}
		if time.Now().After(deadline) {
			return errors.New("VIP 唯一绑定、漂移或连续 SQL 主库连接核验未通过")
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Second):
		}
	}
}

const vipDrillScript = `import json,sys,subprocess,pathlib
p=json.load(sys.stdin);unit='autoops-db-vip-'+str(p['cluster']);config=pathlib.Path('/etc/autoops-database-vip/'+str(p['cluster'])+'/keepalived.conf')
assert config.exists() and '# AutoOps cluster '+str(p['cluster'])+'\n' in config.read_text(),'ownership check failed'
rescue=unit+'-rescue-'+str(p['operation'])
if p['action']=='stop':
 subprocess.run(['systemd-run','--unit='+rescue,'--on-active=60s','/usr/bin/systemctl','start',unit],check=True,stdout=subprocess.DEVNULL,stderr=subprocess.DEVNULL)
 subprocess.run(['systemctl','stop',unit],check=True)
elif p['action']=='restore':
 subprocess.run(['systemctl','start',unit],check=True)
 subprocess.run(['systemctl','stop',rescue+'.timer'],stdout=subprocess.DEVNULL,stderr=subprocess.DEVNULL)
else:raise RuntimeError('invalid drill action')
print(json.dumps({'action':p['action'],'unit':unit,'rescue':rescue}))`

func (s *Service) verifyVIP(ctx context.Context, c *model.Cluster, op model.Operation) error {
	if c.VIP == nil {
		return errors.New("尚未部署 VIP")
	}
	return s.step(op, "VIP 漂移演练（自动恢复保护）", func(log func(string)) (err error) {
		if err = s.waitVIP(ctx, c, 0, 10*time.Second); err != nil {
			return err
		}
		if c.VIP.Status != "healthy" {
			return errors.New("两个 Keepalived 节点必须均在线")
		}
		before := c.VIP.OwnerMemberID
		owner, e := member(c, before)
		if e != nil {
			return e
		}
		input := map[string]any{"cluster": c.ID, "operation": op.ID, "action": "stop"}
		// Restore even on cancellation; independent systemd timer survives API/worker failure.
		defer func() {
			restore, cancel := context.WithTimeout(context.WithoutCancel(ctx), 15*time.Second)
			defer cancel()
			input["action"] = "restore"
			_, e := s.remoteMutation(restore, owner.HostID, vipDrillScript, input)
			if e != nil {
				err = errors.Join(err, errors.New("原节点恢复未确认；独立 60 秒定时器仍负责恢复，请核对服务"))
			}
		}()
		start := time.Now()
		if _, err = s.remoteMutation(ctx, owner.HostID, vipDrillScript, input); err != nil {
			return err
		}
		err = s.waitVIP(ctx, c, before, 40*time.Second)
		driftOwner := c.VIP.OwnerMemberID
		elapsed := time.Since(start).Milliseconds()
		restore, cancel := context.WithTimeout(context.WithoutCancel(ctx), 15*time.Second)
		input["action"] = "restore"
		_, restoreErr := s.remoteMutation(restore, owner.HostID, vipDrillScript, input)
		if restoreErr == nil {
			restoreErr = s.collectVIP(restore, c)
			if c.VIP.Status != "healthy" {
				restoreErr = errors.New("原服务启动后两端状态仍未恢复健康")
			}
		}
		cancel()
		err = errors.Join(err, restoreErr)
		report := map[string]any{"previousOwner": before, "currentOwner": driftOwner, "elapsedMs": elapsed, "originalServiceRestored": restoreErr == nil, "checkedAt": time.Now(), "passed": err == nil, "scope": "VIP 唯一绑定与连续三次 SQL 主库连接；不是数据库写入 RPO 验收"}
		updates := map[string]any{"last_drill": JSON(report)}
		if err == nil {
			updates["verified_at"] = time.Now()
		}
		if e := s.DB.Model(&model.VirtualIP{}).Where("cluster_id = ?", c.ID).Updates(updates).Error; e != nil {
			return e
		}
		log(string(JSON(report)))
		return err
	})
}
