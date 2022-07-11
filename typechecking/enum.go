package typechecking

import (
	"fmt"
	"lugmac/ast"
)

type Enum struct {
	object
	Documentation *ast.ItemDocumentation

	Cases []*Case
}

var _ Type = &Enum{}

func (e Enum) isType()   {}
func (e Enum) isObject() {}
func (e Enum) Child(name string) Object {
	for _, cas := range e.Cases {
		if cas.ObjectName() == name {
			return cas
		}
	}
	return nil
}
func (e Enum) Keyable() bool {
	return false
}
func (e Enum) String() string {
	return fmt.Sprintf("%s", e.ObjectName())
}
func (e Enum) Simple() bool {
	for _, esac := range e.Cases {
		if len(esac.Fields) > 0 {
			return false
		}
	}
	return true
}

type Case struct {
	object
	Documentation *ast.ItemDocumentation

	Fields []*Field
}

var _ Object = &Case{}

func (c Case) isObject() {}
func (c Case) Child(name string) Object {
	for _, field := range c.Fields {
		if field.Name == name {
			return field
		}
	}
	return nil
}
