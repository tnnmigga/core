package mysql

import (
	"github.com/tnnmigga/nett/basic"
	"github.com/tnnmigga/nett/core"
	"github.com/tnnmigga/nett/idef"
	"gorm.io/gorm"
)

type module struct {
	*basic.Module
	semaphore *core.Semaphore // 控制并发数
	db        *gorm.DB
}

func New() idef.IModule {
	m := &module{}
	return m
}
