package typechecking

import (
	"fmt"
	"lugmac/ast"
)

type Struct struct {
	object
	Documentation *ast.ItemDocumentation

	Fields []*Field
}

var _ Type = &Struct{}

func (s Struct) isType()   {}
func (s Struct) isObject() {}
func (s Struct) Child(name string) Object {
	for _, field := range s.Fields {
		if field.Name == name {
			return field
		}
	}
	return nil
}
func (s Struct) Keyable() bool {
	return false
}
func (s Struct) String() string {
	return fmt.Sprintf("%s", s.ObjectName())
}

type Field struct {
	Name          string
	Documentation *ast.ItemDocumentation
	DefinedAt     Path
	InParent      Object
	InEnv         *Environment

	Type Type
}

var _ Object = Field{}

func (f Field) Env() *Environment {
	return f.InEnv
}
func (f Field) isObject() {}
func (f Field) ObjectName() string {
	return f.Name
}
func (f Field) Path() Path {
	return f.DefinedAt
}
func (f Field) Parent() Object {
	return f.InParent
}
func (f Field) Child(name string) Object {
	return nil
}
