package typechecking

import "fmt"

type Enum struct {
	Name      string
	DefinedAt Path

	Cases []Case
}

var _ Type = Enum{}

func (e Enum) isType()   {}
func (e Enum) isObject() {}
func (e Enum) Path() Path {
	return e.DefinedAt
}
func (e Enum) Child(name string) Object {
	for _, cas := range e.Cases {
		if cas.Name == name {
			return cas
		}
	}
	return nil
}
func (e Enum) Keyable() bool {
	return false
}
func (e Enum) String() string {
	return fmt.Sprintf("%s", e.Name)
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
	Name      string
	DefinedAt Path

	Fields []Field
}

var _ Object = Case{}

func (c Case) Path() Path {
	return c.DefinedAt
}
func (c Case) isObject() {}
func (c Case) Child(name string) Object {
	for _, field := range c.Fields {
		if field.Name == name {
			return field
		}
	}
	return nil
}
