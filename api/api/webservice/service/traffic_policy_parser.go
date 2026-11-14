package service

import (
	"errors"
	"fmt"
	"path"
	"sort"
	"strings"

	"dodevops-api/api/webservice/model"
)

// nginxNode records byte spans only. Edits never reserialize native configuration.
// Quoted strings, escaped tokens, comments and ${variables} remain opaque.
type nginxNode struct {
	Args                    []string
	Start, Open, Close, End int
	Children                []*nginxNode
}
type nginxToken struct {
	Value      string
	Start, End int
	Delimiter  byte
}

func nginxTokens(text string) ([]nginxToken, error) {
	if len(text) > 4<<20 {
		return nil, errors.New("配置文件超过解析限制")
	}
	var out []nginxToken
	for i := 0; i < len(text); {
		if strings.ContainsRune(" \r\n\t", rune(text[i])) {
			i++
			continue
		}
		if text[i] == '#' {
			for i < len(text) && text[i] != '\n' {
				i++
			}
			continue
		}
		start := i
		if strings.ContainsRune("{};", rune(text[i])) {
			out = append(out, nginxToken{Value: text[i : i+1], Start: i, End: i + 1, Delimiter: text[i]})
			i++
			continue
		}
		var b strings.Builder
		for i < len(text) {
			c := text[i]
			if strings.ContainsRune(" \r\n\t;{}#", rune(c)) {
				break
			}
			if c == '\\' {
				if i+1 >= len(text) {
					return nil, errors.New("配置包含未结束的转义")
				}
				b.WriteByte(text[i+1])
				i += 2
				continue
			}
			if c == '\'' || c == '"' {
				quote := c
				i++
				closed := false
				for i < len(text) {
					if text[i] == '\\' && i+1 < len(text) {
						b.WriteByte(text[i+1])
						i += 2
						continue
					}
					if text[i] == quote {
						i++
						closed = true
						break
					}
					b.WriteByte(text[i])
					i++
				}
				if !closed {
					return nil, errors.New("配置包含未结束的字符串")
				}
				continue
			}
			if c == '$' && i+1 < len(text) && text[i+1] == '{' {
				j := strings.IndexByte(text[i+2:], '}')
				if j < 0 {
					return nil, errors.New("配置变量未结束")
				}
				j += i + 3
				b.WriteString(text[i:j])
				i = j
				continue
			}
			b.WriteByte(c)
			i++
		}
		if i == start {
			return nil, errors.New("配置标记无法安全解析")
		}
		out = append(out, nginxToken{Value: b.String(), Start: start, End: i})
		if len(out) > 200000 {
			return nil, errors.New("配置指令超过解析限制")
		}
	}
	return out, nil
}
func parseNginx(text string) ([]*nginxNode, error) {
	tokens, err := nginxTokens(text)
	if err != nil {
		return nil, err
	}
	pos := 0
	var read func(int, bool) ([]*nginxNode, int, error)
	read = func(depth int, block bool) ([]*nginxNode, int, error) {
		if depth > 64 {
			return nil, 0, errors.New("配置嵌套超过限制")
		}
		var nodes []*nginxNode
		for pos < len(tokens) {
			if tokens[pos].Delimiter == '}' {
				if !block {
					return nil, 0, errors.New("配置包含多余的右括号")
				}
				close := tokens[pos].Start
				pos++
				return nodes, close, nil
			}
			n := &nginxNode{Start: tokens[pos].Start, Open: -1, Close: -1}
			for pos < len(tokens) && tokens[pos].Delimiter == 0 {
				n.Args = append(n.Args, tokens[pos].Value)
				pos++
			}
			if len(n.Args) == 0 || pos == len(tokens) {
				return nil, 0, errors.New("配置指令未结束")
			}
			token := tokens[pos]
			pos++
			switch token.Delimiter {
			case ';':
				n.End = token.End
			case '{':
				n.Open = token.Start
				n.Children, n.Close, err = read(depth+1, true)
				if err != nil {
					return nil, 0, err
				}
				n.End = n.Close + 1
			default:
				return nil, 0, errors.New("配置指令结构无法安全识别")
			}
			nodes = append(nodes, n)
		}
		if block {
			return nil, 0, errors.New("配置块未结束")
		}
		return nodes, len(text), nil
	}
	nodes, _, err := read(0, false)
	return nodes, err
}

type trafficNativeConfig struct {
	Files    map[string]model.File
	Roots    map[string][]*nginxNode
	HTTP     *nginxNode
	Main     string
	Servers  map[string][]*nginxNode
	HTTPLogs []string
}

func newTrafficNativeConfig(snapshot InstanceConfig) (*trafficNativeConfig, error) {
	c := &trafficNativeConfig{Files: map[string]model.File{}, Roots: map[string][]*nginxNode{}, Servers: map[string][]*nginxNode{}, Main: snapshot.ConfigPath}
	for _, f := range snapshot.Files {
		if f.Deleted {
			continue
		}
		nodes, err := parseNginx(f.Content)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", f.Path, err)
		}
		c.Files[f.Path] = f
		c.Roots[f.Path] = nodes
	}
	for _, n := range c.Roots[c.Main] {
		if len(n.Args) == 1 && n.Args[0] == "http" && n.Open >= 0 {
			if c.HTTP != nil {
				return nil, errors.New("主配置存在多个 http 块")
			}
			c.HTTP = n
		}
	}
	if c.HTTP == nil {
		return nil, errors.New("主配置需要可明确识别的 http 块")
	}
	visited := map[string]bool{}
	var walk func(string, []*nginxNode, int) error
	walk = func(file string, nodes []*nginxNode, depth int) error {
		if depth > 32 {
			return errors.New("include 嵌套超过限制")
		}
		for _, n := range nodes {
			switch n.Args[0] {
			case "server":
				if n.Open >= 0 {
					c.Servers[file] = append(c.Servers[file], n)
				}
			case "access_log":
				if n.Open < 0 {
					c.HTTPLogs = append(c.HTTPLogs, c.Files[file].Content[n.Start:n.End])
				}
			case "include":
				if len(n.Args) != 2 || n.Open >= 0 || strings.Contains(n.Args[1], "$") {
					return errors.New("动态 include 无法安全定位 HTTP 配置")
				}
				pattern := n.Args[1]
				if !strings.HasPrefix(pattern, "/") {
					pattern = path.Join(snapshot.Prefix, pattern)
				}
				matches := []string{}
				for file := range c.Files {
					matched, e := path.Match(pattern, file)
					if e != nil {
						return e
					}
					if matched {
						matches = append(matches, file)
					}
				}
				sort.Strings(matches)
				for _, target := range matches {
					if visited[target] {
						return errors.New("HTTP 配置重复或循环 include，不能自动修改")
					}
					visited[target] = true
					if e := walk(target, c.Roots[target], depth+1); e != nil {
						return e
					}
				}
			}
		}
		return nil
	}
	visited[c.Main] = true
	if err := walk(c.Main, c.HTTP.Children, 0); err != nil {
		return nil, err
	}
	return c, nil
}
func nginxDirective(nodes []*nginxNode, name string) []*nginxNode {
	var out []*nginxNode
	for _, n := range nodes {
		if n.Args[0] == name {
			out = append(out, n)
		}
	}
	return out
}
func trafficLocation(n *nginxNode) (string, string, bool) {
	if n.Args[0] != "location" || n.Open < 0 {
		return "", "", false
	}
	if len(n.Args) == 2 && strings.HasPrefix(n.Args[1], "/") {
		return n.Args[1], "", true
	}
	if len(n.Args) == 3 && (n.Args[1] == "=" || n.Args[1] == "^~") && strings.HasPrefix(n.Args[2], "/") {
		return n.Args[2], n.Args[1], true
	}
	return "", "", false
}

type nginxEdit struct {
	Start, End int
	Text       string
}

func applyNginxEdits(text string, edits []nginxEdit) (string, error) {
	sort.SliceStable(edits, func(i, j int) bool { return edits[i].Start > edits[j].Start })
	boundary := len(text)
	for _, e := range edits {
		if e.Start < 0 || e.End < e.Start || e.End > boundary {
			return "", errors.New("配置编辑范围重叠")
		}
		text = text[:e.Start] + e.Text + text[e.End:]
		boundary = e.Start
	}
	return text, nil
}
