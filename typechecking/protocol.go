package typechecking

import "lugmac/ast"

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
