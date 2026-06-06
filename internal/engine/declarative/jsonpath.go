// jsonpath.go implements the slash-path JSON query used by the declarative JSON
// engine, ported from json_engine.py's parse()/do_query(). A query like
// "results/items" recursively descends decoded JSON collecting all values whose
// key path matches, tolerating intermediate nesting (arrays/objects).
package declarative

import (
	"strconv"
	"strings"
)

// jsonQuery returns all values in data matching the slash-separated path.
func jsonQuery(data any, path string) []any {
	return doQuery(data, parsePath(path))
}

// jsonQueryOne returns the first match or nil.
func jsonQueryOne(data any, path string) any {
	res := jsonQuery(data, path)
	if len(res) == 0 {
		return nil
	}
	return res[0]
}

func parsePath(s string) []string {
	var out []string
	for _, p := range strings.Split(s, "/") {
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

// doQuery ports json_engine.do_query: recursive descent matching key paths.
func doQuery(data any, q []string) []any {
	var ret []any
	if len(q) == 0 {
		return ret
	}
	qkey := q[0]
	for _, pair := range iterate(data) {
		key, value := pair.key, pair.val
		if len(q) == 1 {
			if key == qkey {
				ret = append(ret, value)
			} else if isIterable(value) {
				ret = append(ret, doQuery(value, q)...)
			}
		} else {
			if !isIterable(value) {
				continue
			}
			if key == qkey {
				ret = append(ret, doQuery(value, q[1:])...)
			} else {
				ret = append(ret, doQuery(value, q)...)
			}
		}
	}
	return ret
}

type kv struct {
	key string
	val any
}

// iterate yields (key, value) pairs for maps (string keys) and slices (index
// keys as strings), mirroring json_engine.iterate.
func iterate(data any) []kv {
	switch v := data.(type) {
	case map[string]any:
		out := make([]kv, 0, len(v))
		for k, val := range v {
			out = append(out, kv{k, val})
		}
		return out
	case []any:
		out := make([]kv, 0, len(v))
		for i, val := range v {
			out = append(out, kv{itoa(i), val})
		}
		return out
	default:
		return nil
	}
}

func isIterable(data any) bool {
	switch data.(type) {
	case map[string]any, []any:
		return true
	default:
		return false
	}
}

func itoa(i int) string { return strconv.Itoa(i) }

// toString best-efforts a JSON scalar into a string (ports utils.to_string).
func toString(v any) string {
	switch s := v.(type) {
	case string:
		return s
	case float64:
		if s == float64(int64(s)) {
			return strconv.FormatInt(int64(s), 10)
		}
		return strconv.FormatFloat(s, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(s)
	case nil:
		return ""
	default:
		return ""
	}
}
