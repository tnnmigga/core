package mysql

import (
	"github.com/tnnmigga/nett/conc"
	"github.com/tnnmigga/nett/idef"
	"github.com/tnnmigga/nett/modules/basic"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

const MaxConcurrency = 0xFF

type module struct {
	*basic.Module
	semaphore conc.Semaphore // 控制并发数
	gormDB    *gorm.DB
	mysqlDSN  string
}

func New(name idef.ModName, dsn string) idef.IModule {
	m := &module{
		Module:    basic.New(name, basic.DefaultMQLen),
		semaphore: conc.NewSemaphore(MaxConcurrency),
		mysqlDSN:  dsn,
	}
	m.initHandler()
	m.Before(idef.ServerStateRun, m.beforeRun)
	return m
}

func (m *module) beforeRun() error {
	mdb := mysql.Open(m.mysqlDSN)
	db, err := gorm.Open(mdb, &gorm.Config{
		Logger: gormLogger{},
	})
	if err != nil {
		return err
	}
	m.gormDB = db
	return nil
}
