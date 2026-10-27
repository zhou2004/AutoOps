package service

import (
	"errors"
	"strings"
	"unicode"
)

// Read SQL words outside comments/literals. A semicolon is a statement delimiter
// only outside those regions. Executable MySQL version comments are not allowed.
func sqlWords(statement string) ([]string, error) {
	if strings.TrimSpace(statement) == "" || len(statement) > 256*1024 || strings.ContainsRune(statement, 0) {
		return nil, errors.New("SQL 语句为空或超过 256 KiB")
	}
	words := []string{}
	ended := false
	for i := 0; i < len(statement); {
		c := statement[i]
		if unicode.IsSpace(rune(c)) {
			i++
			continue
		}
		if c == '#' || (c == '-' && i+2 < len(statement) && statement[i+1] == '-' && unicode.IsSpace(rune(statement[i+2]))) {
			for i < len(statement) && statement[i] != '\n' {
				i++
			}
			continue
		}
		if c == '/' && i+1 < len(statement) && statement[i+1] == '*' {
			if i+2 < len(statement) && statement[i+2] == '!' {
				return nil, errors.New("不支持可执行注释")
			}
			end := strings.Index(statement[i+2:], "*/")
			if end < 0 {
				return nil, errors.New("SQL 注释未结束")
			}
			i += end + 4
			continue
		}
		if ended {
			return nil, errors.New("一次只能执行一条 SQL 语句")
		}
		if c == ';' {
			ended = true
			i++
			continue
		}
		if c == '\'' || c == '"' || c == '`' {
			quote := c
			i++
			closed := false
			for i < len(statement) {
				if statement[i] == '\\' {
					i += 2
					continue
				}
				if statement[i] == quote {
					if i+1 < len(statement) && statement[i+1] == quote {
						i += 2
						continue
					}
					i++
					closed = true
					break
				}
				i++
			}
			if !closed {
				return nil, errors.New("SQL 字符串或标识符未结束")
			}
			continue
		}
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || c == '_' {
			start := i
			for i < len(statement) {
				v := statement[i]
				if !((v >= 'a' && v <= 'z') || (v >= 'A' && v <= 'Z') || (v >= '0' && v <= '9') || v == '_') {
					break
				}
				i++
			}
			words = append(words, strings.ToUpper(statement[start:i]))
			continue
		}
		i++
	}
	if len(words) == 0 {
		return nil, errors.New("无法识别 SQL 操作")
	}
	return words, nil
}
