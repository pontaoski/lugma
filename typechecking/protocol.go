package typechecking

import "lugmac/ast"

type Protocol struct {
	object
	Documentation *ast.ItemDocumentation

	Funcs   []*Func
	Events  []*Event
	Signals []*Signal
}

var _ Object = &Protocol{}

func (p Protocol) isObject() {}

func (p Protocol) Child(name string) Object {
	for _, fn := range p.Funcs {
		if fn.ObjectName() == name {
			return fn
		}
	}
	for _, ev := range p.Events {
		if ev.ObjectName() == name {
			return ev
		}
	}
	for _, sig := range p.Signals {
		if sig.ObjectName() == name {
			return sig
		}
	}
	return nil
}

type Func struct {
	object
	Documentation *ast.ItemDocumentation

	Arguments []*Field

	Returns Type
	Throws  Type
}

func (f Func) Child(name string) Object { return nil }

type Event struct {
	object
	Documentation *ast.ItemDocumentation

	Arguments []*Field
}

func (f Event) Child(name string) Object { return nil }

type Signal struct {
	object
	Documentation *ast.ItemDocumentation

	Arguments []*Field
}

func (f Signal) Child(name string) Object { return nil }
