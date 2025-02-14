package payload

import (
	"encoding/json"
	"fmt"
	"github.com/cortezaproject/corteza-server/pkg/filter"
	"github.com/cortezaproject/corteza-server/pkg/handle"
	"github.com/jmoiron/sqlx/types"
	"github.com/spf13/cast"
	"strconv"
	"strings"
	"time"
)

func Uint64toa(i uint64) string {
	return strconv.FormatUint(i, 10)
}

func Uint64stoa(uu []uint64) []string {
	ss := make([]string, len(uu))
	for i, u := range uu {
		ss[i] = Uint64toa(u)
	}

	return ss
}

// ParseUint64 parses an string to uint64
func ParseUint64(s string) uint64 {
	if s == "" {
		return 0
	}
	i, _ := strconv.ParseUint(s, 10, 64)
	return i
}

// ParseUint64s parses a slice of strings into a slice of uint64s
func ParseUint64s(ss []string) []uint64 {
	uu := make([]uint64, len(ss))
	for i, s := range ss {
		uu[i] = ParseUint64(s)
	}

	return uu
}

func ParseJSONTextWithErr(s string) (types.JSONText, error) {
	result := &types.JSONText{}
	err := fmt.Errorf("error parsing JSONText: %w", result.Scan(s))
	return *result, err
}

func ParseISODateWithErr(s string) (time.Time, error) {
	return time.Parse(time.RFC3339, s)
}

func ParseISODatePtrWithErr(s string) (*time.Time, error) {
	t, err := ParseISODateWithErr(s)
	if err != nil {
		return nil, err
	}

	return &t, nil
}

// ParseInt parses a string to int
func ParseInt(s string) int {
	if s == "" {
		return 0
	}
	i, _ := strconv.Atoi(s)
	return i
}

// ParseUInt parses a string to uint64
func ParseUint(s string) uint {
	if s == "" {
		return 0
	}
	i, _ := strconv.ParseUint(s, 10, 32)
	return uint(i)
}

// ParseUint64s parses a slice of strings into a slice of uint64s
func ParseUints(ss []string) []uint {
	uu := make([]uint, len(ss))
	for i, s := range ss {
		uu[i] = ParseUint(s)
	}

	return uu
}

// ParseInt64 parses a string to int64
func ParseInt64(s string) int64 {
	if s == "" {
		return 0
	}
	i, _ := strconv.ParseInt(s, 10, 64)
	return i
}

// parseUInt64 parses a string to uint64
func ParseBool(s string) bool {
	return cast.ToBool(s)
}

func ParseFilterState(s string) filter.State {
	return filter.State(cast.ToUint(s))
}

// similar to labels.ParseStrings but for map[string]any
func ParseMeta(ss []string) (m map[string]any, err error) {
	if len(ss) == 0 {
		return nil, nil
	}

	m = make(map[string]any)

	for _, s := range ss {
		if strings.HasPrefix(s, "{") && strings.HasSuffix(s, "}") {
			// assume json
			if err = json.Unmarshal([]byte(s), &m); err != nil {
				return nil, err
			}

			continue
		}

		kv := strings.SplitN(s, "=", 2)
		if !handle.IsValid(kv[0]) {
			return nil, fmt.Errorf("invalid metadata key format")
		}

		if len(kv) == 2 {
			m[kv[0]] = kv[1]
		} else {
			m[kv[0]] = nil
		}
	}

	return m, nil
}
