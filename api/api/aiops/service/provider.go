package service

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	model "dodevops-api/api/aiops/model"
)

const maxResponse = 4 << 20

var errIncomplete = errors.New("Stream ended before completion / 流式响应意外中断")
var errOutputLimit = errors.New("Output limit reached / 输出达到上限")

func allowedIP(ip net.IP) bool {
	return ip != nil && !ip.IsUnspecified() && !ip.IsMulticast() && !ip.IsLinkLocalUnicast() && !ip.IsLinkLocalMulticast()
}
func validateURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil || u.Hostname() == "" || (u.Scheme != "https" && u.Scheme != "http") || u.User != nil || u.RawQuery != "" || u.Fragment != "" || len(raw) > 1000 {
		return errors.New("Use an HTTP(S) URL without credentials, query or fragment / 请输入不含凭据和查询参数的 HTTP(S) 地址")
	}
	if ip := net.ParseIP(u.Hostname()); ip != nil && !allowedIP(ip) {
		return errors.New("Endpoint address is not allowed / 不允许访问此地址")
	}
	return nil
}

// Only administrators can set endpoints. Private/loopback addresses are intentional for local models.
// Resolve and validate at dial time to prevent DNS rebinding to cloud metadata/link-local addresses.
func providerClient() *http.Client {
	transport := &http.Transport{TLSHandshakeTimeout: 10 * time.Second, ResponseHeaderTimeout: 60 * time.Second, IdleConnTimeout: 30 * time.Second,
		DialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
			host, port, err := net.SplitHostPort(address)
			if err != nil {
				return nil, err
			}
			ips, err := net.DefaultResolver.LookupIPAddr(ctx, host)
			if err != nil {
				return nil, errors.New("cannot resolve endpoint")
			}
			for _, ip := range ips {
				if !allowedIP(ip.IP) {
					return nil, errors.New("endpoint address is not allowed")
				}
			}
			var last error
			for _, ip := range ips {
				conn, err := (&net.Dialer{Timeout: 10 * time.Second, KeepAlive: 30 * time.Second}).DialContext(ctx, network, net.JoinHostPort(ip.IP.String(), port))
				if err == nil {
					return conn, nil
				}
				last = err
			}
			if last == nil {
				last = errors.New("endpoint has no addresses")
			}
			return nil, last
		}}
	return &http.Client{Transport: transport, CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }}
}
func post(ctx context.Context, c model.Config, path string, body interface{}) (*http.Response, error) {
	if err := validateURL(c.BaseURL); err != nil {
		return nil, err
	}
	key, err := openSecret(c.Secret)
	if err != nil {
		return nil, err
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(c.BaseURL, "/")+path, bytes.NewReader(raw))
	if err != nil {
		return nil, errors.New("Invalid endpoint / 接口地址不合法")
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream, application/x-ndjson, application/json")
	if key != "" {
		req.Header.Set("Authorization", "Bearer "+key)
	}
	client := providerClient()
	resp, err := client.Do(req)
	client.CloseIdleConnections()
	if err != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, errors.New("Cannot connect to provider; check endpoint and network / 无法连接服务，请检查地址和网络")
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		resp.Body.Close()
		return nil, fmt.Errorf("Provider HTTP %d; check credentials, model and endpoint / 请检查密钥、模型和地址", resp.StatusCode)
	}
	return resp, nil
}
func scanSSE(reader io.Reader, handle func(string) (bool, error)) error {
	scanner := bufio.NewScanner(io.LimitReader(reader, maxResponse+1))
	scanner.Buffer(make([]byte, 4096), 1<<20)
	var data []string
	total := 0
	dispatch := func() (bool, error) {
		if len(data) == 0 {
			return false, nil
		}
		value := strings.Join(data, "\n")
		data = nil
		return handle(value)
	}
	for scanner.Scan() {
		line := scanner.Text()
		total += len(line) + 1
		if total > maxResponse {
			return errOutputLimit
		}
		if line == "" {
			done, err := dispatch()
			if err != nil {
				return err
			}
			if done {
				return nil
			}
			continue
		}
		if strings.HasPrefix(line, "data:") {
			data = append(data, strings.TrimPrefix(strings.TrimPrefix(line, "data:"), " "))
		}
	}
	if scanner.Err() != nil {
		return errors.New("Invalid or interrupted provider stream / 服务响应中断或格式错误")
	}
	if len(data) > 0 {
		done, err := dispatch()
		if err != nil {
			return err
		}
		if done {
			return nil
		}
	}
	return errIncomplete
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

func streamModel(ctx context.Context, c model.Config, messages []chatMessage, emit func(string) error) error {
	path := "/chat/completions"
	if c.Provider == "ollama" {
		path = "/api/chat"
	}
	resp, err := post(ctx, c, path, map[string]interface{}{"model": c.Model, "messages": messages, "stream": true})
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	textBytes := 0
	output := func(text string) error {
		textBytes += len(text)
		if textBytes > 1<<20 {
			return errOutputLimit
		}
		return emit(text)
	}
	if c.Provider == "ollama" {
		scanner := bufio.NewScanner(io.LimitReader(resp.Body, maxResponse+1))
		scanner.Buffer(make([]byte, 4096), 1<<20)
		total := 0
		for scanner.Scan() {
			total += len(scanner.Bytes()) + 1
			if total > maxResponse {
				return errOutputLimit
			}
			var event struct {
				Message chatMessage `json:"message"`
				Done    bool        `json:"done"`
				Reason  string      `json:"done_reason"`
				Error   string      `json:"error"`
			}
			if err := json.Unmarshal(scanner.Bytes(), &event); err != nil {
				return errors.New("Invalid Ollama stream / Ollama 响应格式错误")
			}
			if event.Error != "" {
				return errors.New("Ollama returned an error; check model / Ollama 返回错误，请检查模型")
			}
			if event.Message.Content != "" {
				if err := output(event.Message.Content); err != nil {
					return err
				}
			}
			if event.Done {
				if event.Reason != "" && event.Reason != "stop" {
					return errors.New("Model output was truncated / 模型输出未完整结束")
				}
				if textBytes == 0 {
					return errors.New("Model returned no text / 模型没有返回文本")
				}
				return nil
			}
		}
		return errIncomplete
	}
	finished := false
	return scanSSE(resp.Body, func(raw string) (bool, error) {
		if raw == "[DONE]" {
			if !finished || textBytes == 0 {
				return false, errIncomplete
			}
			return true, nil
		}
		var event struct {
			Error   json.RawMessage `json:"error"`
			Choices []struct {
				Delta struct {
					Content string `json:"content"`
					Refusal string `json:"refusal"`
				} `json:"delta"`
				FinishReason *string `json:"finish_reason"`
			} `json:"choices"`
		}
		if err := json.Unmarshal([]byte(raw), &event); err != nil {
			return false, errors.New("Invalid model stream / 模型响应格式错误")
		}
		if len(event.Error) > 0 && string(event.Error) != "null" {
			return false, errors.New("Model provider returned an error / 模型服务返回错误")
		}
		if len(event.Choices) > 0 {
			choice := event.Choices[0]
			if choice.Delta.Refusal != "" {
				return false, errors.New("Model declined the request / 模型拒绝了该请求")
			}
			if choice.Delta.Content != "" {
				if err := output(choice.Delta.Content); err != nil {
					return false, err
				}
			}
			if choice.FinishReason != nil {
				if *choice.FinishReason != "stop" {
					return false, errors.New("Model output incomplete; check token limits / 模型输出未完成，请检查输出限制")
				}
				finished = true
			}
		}
		return false, nil
	})
}
func externalWorkflow(ctx context.Context, c model.Config, user uint, prompt, locale, evidence string, emit func(model.Event) error) error {
	payload := map[string]interface{}{"prompt": prompt, "locale": locale, "context": evidence, "user": fmt.Sprintf("autoops-%d", user)}
	path := ""
	if c.Provider == "dify" {
		path = "/workflows/run"
		payload = map[string]interface{}{"inputs": map[string]string{"prompt": prompt, "context": evidence, "locale": locale}, "response_mode": "streaming", "user": fmt.Sprintf("autoops-%d", user)}
	}
	resp, err := post(ctx, c, path, payload)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if c.Provider == "webhook" {
		if strings.Contains(resp.Header.Get("Content-Type"), "text/event-stream") {
			return scanSSE(resp.Body, func(raw string) (bool, error) {
				var e model.Event
				if json.Unmarshal([]byte(raw), &e) != nil {
					return false, errors.New("Invalid webhook event / Webhook 事件格式错误")
				}
				switch e.Type {
				case "delta":
					return false, emit(model.Event{Type: "delta", Text: e.Text})
				case "done":
					return true, nil
				case "error":
					return false, errors.New("Workflow failed / 工作流执行失败")
				}
				return false, nil
			})
		}
		raw, err := io.ReadAll(io.LimitReader(resp.Body, maxResponse+1))
		if err != nil {
			return err
		}
		if len(raw) > maxResponse {
			return errOutputLimit
		}
		var result struct {
			Output string `json:"output"`
		}
		if json.Unmarshal(raw, &result) != nil || strings.TrimSpace(result.Output) == "" {
			return errors.New("Webhook must return JSON {output: string} or SSE / Webhook 返回格式不正确")
		}
		return emit(model.Event{Type: "delta", Text: result.Output})
	}
	hasText := false
	return scanSSE(resp.Body, func(raw string) (bool, error) {
		var event struct {
			Event string `json:"event"`
			Data  struct {
				Title   string                 `json:"title"`
				Text    string                 `json:"text"`
				Status  string                 `json:"status"`
				Outputs map[string]interface{} `json:"outputs"`
			} `json:"data"`
		}
		if json.Unmarshal([]byte(raw), &event) != nil {
			return false, errors.New("Invalid Dify stream / Dify 响应格式错误")
		}
		switch event.Event {
		case "error", "workflow_paused", "human_input_required":
			return false, errors.New("Workflow failed or requires interaction / 工作流失败或需要人工交互")
		case "node_started":
			return false, emit(model.Event{Type: "step", Step: event.Data.Title, Status: "running"})
		case "node_finished":
			return false, emit(model.Event{Type: "step", Step: event.Data.Title, Status: event.Data.Status})
		case "text_chunk":
			hasText = true
			return false, emit(model.Event{Type: "delta", Text: event.Data.Text})
		case "workflow_finished":
			if event.Data.Status != "succeeded" {
				return false, errors.New("Dify workflow did not succeed / Dify 工作流未成功完成")
			}
			if !hasText {
				raw, _ := json.MarshalIndent(event.Data.Outputs, "", "  ")
				if len(event.Data.Outputs) == 0 {
					return false, errors.New("Workflow returned no output / 工作流没有输出")
				}
				if err := emit(model.Event{Type: "delta", Text: string(raw)}); err != nil {
					return false, err
				}
			}
			return true, nil
		}
		return false, nil
	})
}
