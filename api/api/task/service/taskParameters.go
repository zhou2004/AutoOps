package service

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"
)

var taskParameterName = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

func validateTaskParameters(values map[string]string) error {
	for key, value := range values {
		if !taskParameterName.MatchString(key) {
			return fmt.Errorf("参数名 %q 无效，请使用字母、数字和下划线，且不能以数字开头", key)
		}
		if strings.ContainsRune(value, 0) {
			return fmt.Errorf("参数 %s 不能包含空字符", key)
		}
	}
	return nil
}

func mergeTaskParameters(layers ...map[string]string) map[string]string {
	result := map[string]string{}
	for _, layer := range layers {
		for key, value := range layer {
			result[key] = value
		}
	}
	return result
}

func mustParametersJSON(values map[string]string) string {
	data, _ := json.Marshal(values)
	return string(data)
}

// Parameters are data, never script substitutions. Quoting preserves spaces,
// quotes, newlines and shell metacharacters without evaluating them.
func taskParameterExports(values map[string]string) string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	var result strings.Builder
	for _, key := range keys {
		fmt.Fprintf(&result, "export %s=%s; ", key, shellQuote(values[key]))
	}
	return result.String()
}
