package typechecking

import "fmt"

type Struct struct {
	Name          string
	DefinedAt     Path
	Documentation string

	Fields []Field
}

var _ Type = Struct{}

func (s Struct) isType()   {}
func (s Struct) isObject() {}
func (s Struct) Path() Path {
	return s.DefinedAt
}
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
	return fmt.Sprintf("%s", s.Name)
}

type Field struct {
	Name          string
	Documentation string
	DefinedAt     Path

	Type Type
}

var _ Field = Field{}

func (f Field) isObject() {}
func (f Field) Path() Path {
	return f.DefinedAt
}
func (f Field) Child(name string) Object {
	return nil
}
