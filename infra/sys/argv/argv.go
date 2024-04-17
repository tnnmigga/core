package argv

import (
	"os"

	"github.com/tnnmigga/nett/utils"
	"golang.org/x/exp/constraints"
)

// 查找命令行参数中的指定参数并解析成整型
func Int[T constraints.Integer](default_ T, names ...string) T {
	value := Str("", names...)
	if value == "" {
		return default_
	}
	return utils.Integer[T](value)
}

// 查找命令行参数中的指定参数
func Str(default_ string, names ...string) string {
	for _, name := range names {
		if index := utils.Index(os.Args[1:], name); index != -1 {
			return os.Args[index+2]
		}
	}
	return default_
}

// 查找是否存在指定名称的命令行参数
func Find(names ...string) bool {
	for _, name := range names {
		if utils.Index(os.Args[1:], name) != -1 {
			return true
		}
	}
	return false
}
