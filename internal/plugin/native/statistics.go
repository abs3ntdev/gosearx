package native

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"

	"github.com/searxng/gosearx/internal/plugin"
	"github.com/searxng/gosearx/internal/result"
)

// Statistics is a keyword answerer: "<op> n1 n2 …" computes a summary statistic.
// Port of SearXNG's statistics answerer (min/max/avg/sum/prod, plus median).
type Statistics struct{}

func init() {
	plugin.Register(func() plugin.Plugin { return &Statistics{} })
}

func (p *Statistics) Name() string { return "statistics" }
func (p *Statistics) Keywords() []string {
	return []string{"min", "max", "avg", "sum", "prod", "mean", "median"}
}
func (p *Statistics) PreSearch(*plugin.SearchContext) (bool, error) { return true, nil }
func (p *Statistics) OnResult(*result.MainResult) (bool, error)     { return true, nil }

func (p *Statistics) PostSearch(sc *plugin.SearchContext) (result.EngineResults, error) {
	parts := strings.Fields(strings.TrimSpace(sc.Query))
	if len(parts) < 2 {
		return nil, nil
	}
	op := strings.ToLower(parts[0])
	nums := make([]float64, 0, len(parts)-1)
	for _, s := range parts[1:] {
		f, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return nil, nil // not a numbers query
		}
		nums = append(nums, f)
	}
	if len(nums) == 0 {
		return nil, nil
	}

	var res float64
	switch op {
	case "min":
		res = nums[0]
		for _, n := range nums {
			res = math.Min(res, n)
		}
	case "max":
		res = nums[0]
		for _, n := range nums {
			res = math.Max(res, n)
		}
	case "sum":
		for _, n := range nums {
			res += n
		}
	case "prod":
		res = 1
		for _, n := range nums {
			res *= n
		}
	case "avg", "mean":
		for _, n := range nums {
			res += n
		}
		res /= float64(len(nums))
	case "median":
		sort.Float64s(nums)
		m := len(nums) / 2
		if len(nums)%2 == 0 {
			res = (nums[m-1] + nums[m]) / 2
		} else {
			res = nums[m]
		}
	default:
		return nil, nil
	}

	ans := fmt.Sprintf("%s = %s", op, strconv.FormatFloat(res, 'g', -1, 64))
	return result.EngineResults{&result.Answer{EngineName: p.Name(), Answer: ans}}, nil
}
