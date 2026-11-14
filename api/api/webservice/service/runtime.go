package service

import (
	"bytes"
	"context"
	cmdbservice "dodevops-api/api/cmdb/service"
	"dodevops-api/api/webservice/model"
	assets "dodevops-api/automation/web-platform"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"gorm.io/gorm"
)

type RemoteRequest struct {
	TrafficCursors json.RawMessage     `json:"trafficCursors,omitempty"`
	VIP            *VIPNode            `json:"vip,omitempty"`
	HealthChecks   []model.Entry       `json:"healthChecks,omitempty"`
	Logs           []map[string]string `json:"logs,omitempty"`
	Offset         int                 `json:"offset,omitempty"`
	From           string              `json:"from,omitempty"`
	To             string              `json:"to,omitempty"`
	TLS            []TLSProbe          `json:"tls,omitempty"`
	Action         string              `json:"action"`
	Spec           model.SourceSpec    `json:"spec"`
	OperationID    uint                `json:"operationId"`
	Identity       string              `json:"identity"`
	Baseline       string              `json:"baseline"`
	Files          []model.File        `json:"files,omitempty"`
	LogID          string              `json:"logId,omitempty"`
	Lines          int                 `json:"lines,omitempty"`
	Keyword        string              `json:"keyword,omitempty"`
	Rotation       string              `json:"rotation,omitempty"`
	Status         string              `json:"status,omitempty"`
	IP             string              `json:"ip,omitempty"`
	URI            string              `json:"uri,omitempty"`
}
type TLSProbe struct {
	BindingID   uint   `json:"bindingId"`
	VersionID   uint   `json:"versionId"`
	Host        string `json:"host"`
	Port        int    `json:"port"`
	ServerName  string `json:"serverName"`
	Fingerprint string `json:"fingerprint"`
}

type Observation struct {
	Certificates      []map[string]string `json:"certificates,omitempty"`
	TLSFingerprint    string              `json:"tlsFingerprint,omitempty"`
	Metrics           map[string]float64  `json:"metrics,omitempty"`
	AppliedRevisionID uint                `json:"appliedRevisionId,omitempty"`
	ExpectedDigest    string              `json:"expectedDigest,omitempty"`
	Files             []model.File        `json:"files"`
	Digest            string              `json:"digest"`
	Identity          string              `json:"identity"`
	Version           string              `json:"version"`
	Logs              []map[string]string `json:"logs"`
	State             string              `json:"state"`
	Message           string              `json:"message"`
	Observed          *Observation        `json:"observed,omitempty"`
}
type RuntimeConnector interface {
	Run(context.Context, RemoteRequest) (json.RawMessage, error)
}
type EngineProvider interface {
	Discover(context.Context, RuntimeConnector, model.SourceSpec) (Observation, error)
	RenderSite(model.SiteSpec) (string, error)
}
type Nginx struct{}

func (Nginx) RenderSite(p model.SiteSpec) (string, error) { return RenderSite(p) }
func (Nginx) Discover(ctx context.Context, r RuntimeConnector, p model.SourceSpec) (Observation, error) {
	b, e := r.Run(ctx, RemoteRequest{Action: "discover", Spec: p})
	if e != nil {
		return Observation{}, e
	}
	return decode[Observation](b)
}

type SSHRuntime struct{ DB *gorm.DB }
type boundedBuffer struct {
	bytes.Buffer
	Max int
}

func (b *boundedBuffer) Write(p []byte) (int, error) {
	if b.Len()+len(p) > b.Max {
		return 0, errors.New("remote output limit exceeded")
	}
	return b.Buffer.Write(p)
}
func quote(v string) string { return "'" + strings.ReplaceAll(v, "'", "'\"'\"'") + "'" }
func (r *SSHRuntime) Run(parent context.Context, req RemoteRequest) (json.RawMessage, error) {
	ctx, cancel := context.WithTimeout(parent, 3*time.Minute)
	defer cancel()
	client, e := cmdbservice.ResolveAssetSSH(r.DB, "host", req.Spec.HostID, req.Spec.ConnectionID)
	if e != nil {
		return nil, e
	}
	defer client.Close()
	stop := context.AfterFunc(ctx, func() { client.Close() })
	defer stop()
	session, e := client.NewSession()
	if e != nil {
		return nil, e
	}
	defer session.Close()
	session.Stdin = bytes.NewReader(JSON(req))
	out := &boundedBuffer{Max: 8 << 20}
	errout := &boundedBuffer{Max: 32 << 10}
	session.Stdout = out
	session.Stderr = errout
	code := "import base64;exec(compile(base64.b64decode('" + base64.StdEncoding.EncodeToString([]byte(assets.Runner)) + "'),'<autoops-web>','exec'))"
	command := "python3 -c " + quote(code)
	if req.Spec.Sudo {
		command = "sudo -n " + command
	}
	runErr := session.Run(command)
	var response struct {
		OK    bool            `json:"ok"`
		Data  json.RawMessage `json:"data"`
		Error string          `json:"error"`
	}
	if e = json.Unmarshal(out.Bytes(), &response); e != nil {
		return nil, fmt.Errorf("remote result unavailable: %w", errors.Join(runErr, ctx.Err(), e))
	}
	if !response.OK {
		// Candidate trees are staged under /tmp; show the real file path so the
		// message matches what the page displays.
		return nil, errors.New(stagingPath.ReplaceAllString(response.Error, ""))
	}
	if runErr != nil {
		return nil, runErr
	}
	return response.Data, nil
}
func (s *Service) Discover(ctx context.Context, u, id uint) (model.Resource, error) {
	r, e := s.Get("sources", id)
	if e != nil {
		return r, e
	}
	if e = s.Asset(u, "web_source", id, "web_view"); e != nil {
		return r, e
	}
	p, e := decode[model.SourceSpec](r.Spec)
	if e != nil {
		return r, e
	}
	ob, e := (Nginx{}).Discover(ctx, s.Remote, p)
	if e != nil {
		// Preserve the last successful observation and record the failed check.
		// A registered source does not imply that an Nginx instance exists.
		previous, _ := decode[Observation](r.Observed)
		previous.State = "unavailable"
		previous.Message = e.Error()
		if len(previous.Message) > 3000 {
			previous.Message = previous.Message[:3000]
		}
		updated := s.DB.Table(model.Tables["sources"]).Where("id = ? AND revision = ?", r.ID, r.Revision).Updates(map[string]any{"status": "unavailable", "observed": JSON(previous), "collected_at": time.Now().UTC()})
		if updated.Error != nil {
			return r, errors.Join(e, updated.Error)
		}
		if updated.RowsAffected != 1 {
			return r, ErrConflict
		}
		return r, e
	}
	canonical := digest([]byte(fmt.Sprintf("%d:%s:%s", p.HostID, p.Runtime, ob.Identity)))
	now := time.Now().UTC()
	e = s.DB.Transaction(func(tx *gorm.DB) error {
		updated := tx.Table(model.Tables["sources"]).Where("id = ? AND revision = ?", r.ID, r.Revision).Updates(map[string]any{"status": "healthy", "observed": JSON(ob), "collected_at": now})
		if updated.Error != nil {
			return updated.Error
		}
		if updated.RowsAffected != 1 {
			return ErrConflict
		}
		var i model.Resource
		result := tx.Table(model.Tables["instances"]).Where("source_id = ?", r.ID).First(&i)
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			i = model.Resource{Name: r.Name, GroupID: r.GroupID, SourceID: r.ID, Engine: r.Engine, Environment: r.Environment, Revision: 1, Management: "observe", Spec: r.Spec, LocatorHash: &canonical, CreatedBy: u}
			if e := tx.Table(model.Tables["instances"]).Create(&i).Error; e != nil {
				return e
			}
		} else if result.Error != nil {
			return result.Error
		}
		status := "healthy"
		old, _ := decode[Observation](i.Observed)
		ob.ExpectedDigest = old.ExpectedDigest
		if ob.ExpectedDigest == "" && i.Management == "operate" {
			ob.ExpectedDigest = old.Digest
		}
		if i.Management == "operate" && ob.ExpectedDigest != "" && ob.ExpectedDigest != ob.Digest {
			status = "drifted"
		}
		if e := tx.Table(model.Tables["instances"]).Where("id = ?", i.ID).Updates(map[string]any{"status": status, "observed": JSON(ob), "collected_at": now, "spec": r.Spec}).Error; e != nil {
			return e
		}
		return nil
	})
	if e != nil {
		return r, e
	}
	return s.Get("sources", id)
}

var imageDigestPattern = regexp.MustCompile(`^(?:[A-Za-z0-9][A-Za-z0-9._/:-]*@)?sha256:[a-f0-9]{64}$`)
