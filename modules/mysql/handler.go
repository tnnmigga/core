package mysql

import (
	"github.com/tnnmigga/nett/core"
	"github.com/tnnmigga/nett/msgbus"
)

func (m *module) initHandler() {
	msgbus.RegisterRPC(m, m.onExecSQL)
	msgbus.RegisterRPC(m, m.onRawSQL)
	msgbus.RegisterRPC(m, m.onExecGORM)
	msgbus.RegisterRPC(m, m.onFirst)
}

func (m *module) onExecSQL(req *ExecSQL, resolve func(any), reject func(error)) {
	core.GoWithGroup(req.GroupKey, func() {
		m.semaphore.P()
		defer m.semaphore.V()
		err := m.gormDB.Exec(req.SQL, req.Args...).Error
		if err != nil {
			reject(err)
			return
		}
		resolve(SQLExecOK)
	})
}

func (m *module) onRawSQL(req *RawSQL, resolve func(any), reject func(error)) {
	core.GoWithGroup(req.GroupKey, func() {
		m.semaphore.P()
		defer m.semaphore.V()
		var raws []map[string]any
		err := m.gormDB.Raw(req.SQL, req.Args...).Scan(&raws).Error
		if err != nil {
			reject(err)
			return
		}
		resolve(Raws(raws))
	})
}

func (m *module) onFirst(req *First, resolve func(any), reject func(error)) {
	core.GoWithGroup(req.GroupKey, func() {
		m.semaphore.P()
		defer m.semaphore.V()
		var res map[string]any
		err := m.gormDB.Table(req.Table).Where(req.Where, req.Args...).Select(req.Select).Limit(1).Scan(&res).Error
		if err != nil {
			reject(err)
			return
		}
		resolve(Raw(res))
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
