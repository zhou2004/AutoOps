package service

import (
	"bufio"
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	model "dodevops-api/api/aiops/model"
	"dodevops-api/pkg/apikey"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

func HarnessAvailable() bool {
	return os.Getenv("AIOPS_HARNESS_URL") != "" && len(os.Getenv("AIOPS_HARNESS_SECRET")) >= 32 && os.Getenv("AIOPS_CALLBACK_URL") != ""
}
func (s *Service) HarnessTool(ctx context.Context, token string, r model.ToolRequest) (model.ToolCall, error) {
	if len(token) != 64 {
		return model.ToolCall{}, errors.New("Harness 授权无效")
	}
	h := sha256.Sum256([]byte(token))
	var ticket model.HarnessTicket
	if s.DB.WithContext(ctx).Where("hash = ? AND expires_at > ?", hex.EncodeToString(h[:]), time.Now().UTC()).First(&ticket).Error != nil || !ticket.ExpiresAt.After(time.Now()) {
		return model.ToolCall{}, errors.New("Harness 授权已失效")
	}
	operate := ticket.CanOperate
	if ticket.APIKeyID > 0 {
		key, err := apikey.Valid(ctx, s.DB, ticket.APIKeyID)
		if err != nil || key.UserID != ticket.UserID {
			return model.ToolCall{}, errors.New("API Key 已失效")
		}
		operate = operate && apikey.Has(key, "aiops:operate")
		ctx = apikey.WithKey(ctx, key.ID)
	}
	return s.ExecuteTool(ctx, ticket.UserID, ticket.SessionID, operate, r)
}
func (s *Service) harnessChat(ctx context.Context, user uint, session model.Session, keyID uint, canOperate bool, cfg model.Config, messages []chatMessage, emit func(model.Event) error, delta func(string) error) error {
	if !HarnessAvailable() {
		return errors.New("Harness 尚未连接，请配置桥接服务及回调地址")
	}
	bridge := strings.TrimRight(os.Getenv("AIOPS_HARNESS_URL"), "/")
	callback := strings.TrimRight(os.Getenv("AIOPS_CALLBACK_URL"), "/") + "/api/v1/aiops/harness/tools"
	if err := validateURL(bridge); err != nil {
		return err
	}
	if err := validateURL(callback); err != nil {
		return err
	}
	if err := validateURL(cfg.BaseURL); err != nil {
		return err
	}
	key, err := openSecret(cfg.Secret)
	if err != nil {
		return err
	}
	tokenBytes := make([]byte, 32)
	if _, err = rand.Read(tokenBytes); err != nil {
		return err
	}
	token := hex.EncodeToString(tokenBytes)
	hash := sha256.Sum256([]byte(token))
	deadline, ok := ctx.Deadline()
	if !ok {
		deadline = time.Now().Add(180 * time.Second)
	}
	ticket := model.HarnessTicket{Hash: hex.EncodeToString(hash[:]), UserID: user, SessionID: session.ID, APIKeyID: keyID, CanOperate: canOperate, ExpiresAt: deadline}
	if err = s.DB.WithContext(ctx).Create(&ticket).Error; err != nil {
		return err
	}
	defer func() {
		p, c := persistContext()
		defer c()
		s.DB.WithContext(p).Where("hash = ? OR expires_at < ?", ticket.Hash, time.Now().UTC()).Delete(&model.HarnessTicket{})
	}()
	// Each turn gets a fresh isolated runtime; durable conversation lives in AutoOps.
	body, _ := json.Marshal(map[string]any{"model": cfg.Model, "baseUrl": cfg.BaseURL, "apiKey": key, "messages": messages, "callbackUrl": callback, "callbackToken": token, "targets": session.Targets, "actions": session.Actions, "timeout": int(time.Until(deadline).Seconds())})
	req, err := http.NewRequestWithContext(ctx, "POST", bridge+"/run", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+os.Getenv("AIOPS_HARNESS_SECRET"))
	client := providerClient()
	defer client.CloseIdleConnections()
	resp, err := client.Do(req)
	if err != nil {
		return errors.New("无法连接 Harness 桥接服务")
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return errors.New("Harness 桥接服务拒绝请求，请检查连接配置")
	}
	scanner := bufio.NewScanner(io.LimitReader(resp.Body, maxResponse+1))
	scanner.Buffer(make([]byte, 4096), 1<<20)
	complete := false
	for scanner.Scan() {
		var event model.Event
		if json.Unmarshal(scanner.Bytes(), &event) != nil {
			return errors.New("Harness 响应格式无效")
		}
		switch event.Type {
		case "delta":
			if err = delta(event.Text); err != nil {
				return err
			}
		case "step", "tool":
			if err = emit(event); err != nil {
				return err
			}
		case "error":
			return errors.New("Harness 执行失败，请检查模型连接或桥接服务日志")
		case "done":
			complete = true
		}
	}
	if err = scanner.Err(); err != nil {
		return err
	}
	if !complete {
		return errIncomplete
	}
	return nil
}
