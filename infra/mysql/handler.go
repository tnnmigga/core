package mysql

import (
	"github.com/tnnmigga/nett/core"
	"github.com/tnnmigga/nett/msgbus"
)

func (m *module) initHandler() {
	msgbus.RegisterRPC(m, m.onExecSQL)
	msgbus.RegisterRPC(m, m.onExecGORM)
}

func (m *module) onExecSQL(req *ExecSQL, resolve func(any), reject func(error)) {
	core.GoWithGroup(req.GroupKey, func() {
		m.semaphore.P()
		defer m.semaphore.V()
		var raws Raws
		err := m.gormDB.Raw(req.SQL, req.Args...).Scan(&raws).Error
		if err != nil {
			reject(err)
			return
		}
		resolve(raws)
	})
}

func (m *module) onExecGORM(req *ExecGORM, resolve func(any), reject func(error)) {
	core.GoWithGroup(req.GroupKey, func() {
		m.semaphore.P()
		defer m.semaphore.V()
		res, err := req.GORM(m.gormDB)
		if err != nil {
			reject(err)
			return
		}
		resolve(res)
	})
}
