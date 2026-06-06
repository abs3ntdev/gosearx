// serialize.go round-trips EngineResults to/from JSON for the response cache.
// Each result is stored as its concrete JSON plus its Kind discriminator, and
// reconstructed via FromMap (the same path the Lua/script tiers use), so every
// result type is supported without per-type marshalers.
package result

import "encoding/json"

// Marshal serializes engine results to a cacheable JSON blob.
func Marshal(results EngineResults) ([]byte, error) {
	maps := make([]map[string]any, 0, len(results))
	for _, r := range results {
		b, err := json.Marshal(r)
		if err != nil {
			continue
		}
		var m map[string]any
		if json.Unmarshal(b, &m) != nil {
			continue
		}
		// ensure the discriminator is present (some types omit a "type" field)
		if _, ok := m["type"]; !ok {
			m["type"] = string(r.Kind())
		}
		maps = append(maps, m)
	}
	return json.Marshal(maps)
}

// Unmarshal reconstructs engine results from a cached JSON blob. engineName is
// applied when a stored result lacks one.
func Unmarshal(data []byte, engineName string) (EngineResults, error) {
	var maps []map[string]any
	if err := json.Unmarshal(data, &maps); err != nil {
		return nil, err
	}
	out := make(EngineResults, 0, len(maps))
	for _, m := range maps {
		eng := engineName
		if e, ok := m["engine"].(string); ok && e != "" {
			eng = e
		}
		if r := FromMap(eng, m); r != nil {
			out = append(out, r)
		}
	}
	return out, nil
}
