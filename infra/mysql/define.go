package mysql

import "gorm.io/gorm"

type Raw map[string]any
type Raws []Raw

// 直接根据sql执行
// 支持跨进程调用
type ExecSQL struct {
	SQL  string
	Args []any
}

// 使用gorm执行
// 不支持跨进程调用
// 传入一个执行函数进程gorm操作
// 返回需要的结果为RPC回调函数需要的参数
// 传入的函数需要注意并发安全最好只执行基础的gorm操作然后在回调函数中处理结果
type ExecGORM struct {
	GORM func(*gorm.DB) (any, error)
}
