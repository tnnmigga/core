package mysql

func (m *module) onExecSQL(req *ExecSQL, resolve func(any), reject func(error)) {
	m.semaphore.P()
	defer m.semaphore.V()
	var raws Raws
	err := m.db.Raw(req.SQL, req.Args...).Scan(&raws).Error
	if err != nil {
		reject(err)
		return
	}
	resolve(raws)
}

func (m *module) onExecGORM(req *ExecGORM, resolve func(any), reject func(error)) {
	m.semaphore.P()
	defer m.semaphore.V()
	
	resolve(nil)
}