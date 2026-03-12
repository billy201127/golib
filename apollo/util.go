package apollo

import (
	"bufio"
	"strconv"
	"strings"
)

func parsePropertiesInline(content string) map[string]string {
	result := make(map[string]string)
	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "!") {
			continue
		}
		idx := strings.IndexAny(line, "=:")
		if idx <= 0 {
			continue
		}
		key := strings.TrimSpace(line[:idx])
		val := strings.TrimSpace(line[idx+1:])
		result[key] = val
	}
	return result
}

func buildNestedMap(props map[string]string) map[string]interface{} {
	var root interface{} = map[string]interface{}{}
	for key, val := range props {
		segments := parseKeyPath(key)
		if len(segments) == 0 {
			continue
		}
		root = setValue(root, segments, parseValue(val))
	}
	if m, ok := root.(map[string]interface{}); ok {
		return m
	}
	return map[string]interface{}{}
}

type pathPart struct {
	key     string
	index   int
	isIndex bool
}

func setValue(node interface{}, segments []pathPart, value interface{}) interface{} {
	if len(segments) == 0 {
		return node
	}
	part := segments[0]
	isLast := len(segments) == 1

	if part.isIndex {
		list, ok := node.([]interface{})
		if !ok {
			list = []interface{}{}
		}
		if len(list) <= part.index {
			list = append(list, make([]interface{}, part.index-len(list)+1)...)
		}
		if isLast {
			list[part.index] = value
			return list
		}
		list[part.index] = setValue(list[part.index], segments[1:], value)
		return list
	}

	currMap, ok := node.(map[string]interface{})
	if !ok {
		currMap = make(map[string]interface{})
	}
	if isLast {
		currMap[part.key] = value
		return currMap
	}
	currMap[part.key] = setValue(currMap[part.key], segments[1:], value)
	return currMap
}

func parseKeyPath(key string) []pathPart {
	parts := make([]pathPart, 0)
	buf := strings.Builder{}
	flushKey := func() {
		if buf.Len() == 0 {
			return
		}
		parts = append(parts, pathPart{key: buf.String()})
		buf.Reset()
	}
	for i := 0; i < len(key); i++ {
		ch := key[i]
		switch ch {
		case '.':
			flushKey()
		case '[':
			flushKey()
			end := strings.IndexByte(key[i+1:], ']')
			if end == -1 {
				continue
			}
			idxRaw := key[i+1 : i+1+end]
			if idx, err := strconv.Atoi(idxRaw); err == nil && idx >= 0 {
				parts = append(parts, pathPart{index: idx, isIndex: true})
			}
			i = i + end + 1
		case ']':
			// skip
		default:
			buf.WriteByte(ch)
		}
	}
	flushKey()
	return parts
}

func parseValue(raw string) interface{} {
	value := strings.TrimSpace(raw)
	if value == "" {
		return ""
	}
	if strings.HasPrefix(value, "[") && strings.HasSuffix(value, "]") {
		inner := strings.TrimSpace(value[1 : len(value)-1])
		if inner == "" {
			return []interface{}{}
		}
		parts := splitComma(inner)
		items := make([]interface{}, 0, len(parts))
		for _, part := range parts {
			items = append(items, parseValue(part))
		}
		return items
	}

	if parsed, ok := parseScalar(value); ok {
		return parsed
	}

	return value
}

func parseScalar(value string) (interface{}, bool) {
	if strings.EqualFold(value, "null") || strings.EqualFold(value, "~") {
		return nil, true
	}
	if strings.EqualFold(value, "true") {
		return true, true
	}
	if strings.EqualFold(value, "false") {
		return false, true
	}
	if i, err := strconv.ParseInt(value, 10, 64); err == nil {
		return i, true
	}
	if f, err := strconv.ParseFloat(value, 64); err == nil {
		return f, true
	}
	if (strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"")) ||
		(strings.HasPrefix(value, "'") && strings.HasSuffix(value, "'")) {
		trimmed := strings.TrimSuffix(strings.TrimPrefix(value, value[:1]), value[:1])
		return trimmed, true
	}
	return nil, false
}

func splitComma(value string) []string {
	result := make([]string, 0)
	start := 0
	inQuotes := false
	quoteChar := byte(0)
	for i := 0; i < len(value); i++ {
		ch := value[i]
		if ch == '\\' && i+1 < len(value) {
			i++
			continue
		}
		if ch == '\'' || ch == '"' {
			if inQuotes && ch == quoteChar {
				inQuotes = false
				quoteChar = 0
			} else if !inQuotes {
				inQuotes = true
				quoteChar = ch
			}
			continue
		}
		if ch == ',' && !inQuotes {
			part := strings.TrimSpace(value[start:i])
			result = append(result, part)
			start = i + 1
		}
	}
	if start <= len(value) {
		part := strings.TrimSpace(value[start:])
		if part != "" {
			result = append(result, part)
		}
	}
	return result
}
