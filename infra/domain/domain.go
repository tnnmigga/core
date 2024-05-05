package domain

import "github.com/tnnmigga/core/idef"

type Root interface {
	PutCase(index int, useCase any)
	GetCase(index int) any
}

type root struct {
	idef.IModule
	useCases []any
}

func New(m idef.IModule, maxCaseIndex int) Root {
	return &root{
		IModule:  m,
		useCases: make([]any, maxCaseIndex),
	}
}

func (p *root) PutCase(caseIndex int, useCase any) {
	p.useCases[caseIndex] = useCase
}

func (p *root) GetCase(caseIndex int) any {
	return p.useCases[caseIndex]
}
