package conf

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"

	"github.com/tnnmigga/nett/infra/process"
)

func init() {
	RegInitFn(ckeckServer)
	fname := process.Argv.Str("-c", "configs.jsonc")
	b := loadLocalFile(fname)
	LoadFromJSON(b)
	err := afterLoad()
	if err != nil {
		panic(err)
	}
}

func loadLocalFile(fname string) []byte {
	file, err := os.OpenFile(fname, os.O_RDONLY, 0)
	if err != nil {
		panic(err)
	}
	b, err := io.ReadAll(file)
	if err != nil {
		panic(err)
	}
	return b
}

var (
	confs map[string]any = map[string]any{}
	fns   []func() error
)

var errConfigNotFound error = errors.New("configs not found")

func LoadFromJSON(b []byte) {
	b = uncomment(b)
	err := json.Unmarshal(b, &confs)
	if err != nil {
		panic(fmt.Errorf("LoadFromJSON unmarshal error %v", err))
	}
}

func uncomment(b []byte) []byte {
	reg := regexp.MustCompile(`/\*{1,2}[\s\S]*?\*/`)
	b = reg.ReplaceAll(b, []byte("\n"))
	reg = regexp.MustCompile(`\s//[\s\S]*?\n`)
	return reg.ReplaceAll(b, []byte("\n"))
}

func RegInitFn(fn func() error) {
	fns = append(fns, fn)
}

// func transform(oc map[string]any, nc map[string]any, prefix []string) {
// 	for k, v := range oc {
// 		switch v := v.(type) {
// 		case map[string]any:
// 			transform(v, nc, append(prefix, k))
// 		default:
// 			nc[strings.Join(append(prefix, k), ".")] = oc[k]
// 		}
// 	}
// }

func afterLoad() error {
	for _, fn := range fns {
		if err := fn(); err != nil {
			return err
		}
	}
	return nil
}

type vType interface {
	float64 | bool | string | map[string]any | []any
}

func Any[T vType](name string) (v T, ok bool) {
	path := strings.Split(name, ".")
	var next any = confs
	for _, n := range path {
		tmp, ok := next.(map[string]any)
		if !ok {
			return v, false
		}
		next, ok = tmp[n]
		if !ok {
			return v, false
		}
	}
	// 类型错误触发panic中断
	return next.(T), true
}

func Int(name string, default_ ...int) int {
	v, ok := Any[float64](name)
	if ok {
		return int(v)
	}
	if len(default_) > 0 {
		return default_[0]
	}
	panic(errConfigNotFound)
}

func Int64(name string, default_ ...int64) int64 {
	v, ok := Any[float64](name)
	if ok {
		return int64(v)
	}
	if len(default_) > 0 {
		return default_[0]
	}
	panic(errConfigNotFound)
}

func Int32(name string, default_ ...int32) int32 {
	v, ok := Any[float64](name)
	if ok {
		return int32(v)
	}
	if len(default_) > 0 {
		return default_[0]
	}
	panic(errConfigNotFound)
}

func Uint64(name string, default_ ...uint64) uint64 {
	v, ok := Any[float64](name)
	if ok {
		return uint64(v)
	}
	if len(default_) > 0 {
		return default_[0]
	}
	panic(errConfigNotFound)
}

func Uint32(name string, default_ ...uint32) uint32 {
	v, ok := Any[float64](name)
	if ok {
		return uint32(v)
	}
	if len(default_) > 0 {
		return default_[0]
	}
	panic(errConfigNotFound)
}

func String(name string, default_ ...string) string {
	v, ok := Any[string](name)
	if ok {
		return string(v)
	}
	if len(default_) > 0 {
		return default_[0]
	}
	panic(errConfigNotFound)
}

func Float64(name string, default_ ...float64) float64 {
	v, ok := Any[float64](name)
	if ok {
		return v
	}
	if len(default_) > 0 {
		return default_[0]
	}
	panic(errConfigNotFound)
}

func Bool(name string, default_ ...bool) bool {
	v, ok := Any[bool](name)
	if ok {
		return v
	}
	if len(default_) > 0 {
		return default_[0]
	}
	panic(errConfigNotFound)
}

func Array[T vType](name string, default_ ...[]T) []T {
	a, ok := Any[[]any](name)
	if ok {
		ar := make([]T, len(a))
		for i, v := range a {
			ar[i] = v.(T)
		}
		return ar
	}
	if len(default_) > 0 {
		return default_[0]
	}
	panic(errConfigNotFound)
}

func Map[T vType](name string, default_ ...map[string]T) map[string]T {
	a, ok := Any[map[string]any](name)
	if ok {
		m := make(map[string]T, len(a))
		for k, v := range a {
			m[k] = v.(T)
		}
		return m
	}
	if len(default_) > 0 {
		return default_[0]
	}
	panic(errConfigNotFound)
}
