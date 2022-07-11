package typechecking

import "fmt"

type Type interface {
	Object

	isType()
	Keyable() bool
	String() string
}

type PrimitiveType int

// Env implements Type
func (PrimitiveType) Env() *Environment {
	panic("unimplemented")
}

const (
	UInt8 PrimitiveType = iota
	UInt16
	UInt32
	UInt64

	Int8
	Int16
	Int32
	Int64

	String
	Bytes

	Bool
)

func (p PrimitiveType) isObject() {}
func (p PrimitiveType) isType()   {}
func (p PrimitiveType) Child(name string) Object {
	return nil
}
func (p PrimitiveType) Keyable() bool {
	return true
}
func (p PrimitiveType) String() string {
	switch p {
	case UInt8:
		return "UInt8"
	case UInt16:
		return "UInt16"
	case UInt32:
		return "UInt32"
	case UInt64:
		return "UInt64"
	case Int8:
		return "Int8"
	case Int16:
		return "Int16"
	case Int32:
		return "Int32"
	case Int64:
		return "Int64"
	case String:
		return "String"
	case Bytes:
		return "Bytes"
	case Bool:
		return "Bool"
	default:
		panic("Bad primitive type")
	}
}
func (p PrimitiveType) Parent() Object {
	return nil
}
func (p PrimitiveType) Path() Path {
	return Path{"", p.String()}
}
func (p PrimitiveType) ObjectName() string {
	return "Array"
}

var _ Type = PrimitiveType(0)

type ArrayType struct {
	Element Type
}

func (a ArrayType) isObject() {}
func (a ArrayType) isType()   {}
func (a ArrayType) Child(name string) Object {
	switch name {
	case "Element":
		return a.Element
	default:
		return nil
	}
}
func (a ArrayType) Env() *Environment {
	return nil
}
func (a ArrayType) Keyable() bool {
	return false
}
func (a ArrayType) String() string {
	return fmt.Sprintf("[%s]", a.Element)
}
func (a ArrayType) Path() Path {
	return Path{"", "Array"}
}
func (a ArrayType) Parent() Object {
	return nil
}
func (d ArrayType) ObjectName() string {
	return "Array"
}

var _ Type = ArrayType{}

type DictionaryType struct {
	Key     Type
	Element Type
}

func (d DictionaryType) isObject()         {}
func (d DictionaryType) isType()           {}
func (d DictionaryType) Env() *Environment { return nil }
func (d DictionaryType) Child(name string) Object {
	switch name {
	case "Element":
		return d.Element
	case "Key":
		return d.Key
	default:
		return nil
	}
}
func (d DictionaryType) Keyable() bool {
	return false
}
func (d DictionaryType) String() string {
	return fmt.Sprintf("[%s: %s]", d.Key, d.Element)
}
func (d DictionaryType) Path() Path {
	return Path{"", "Dictionary"}
}
func (d DictionaryType) Parent() Object {
	return nil
}
func (d DictionaryType) ObjectName() string {
	return "Dictionary"
}

var _ Type = DictionaryType{}

type OptionalType struct {
	Element Type
}

func (o OptionalType) isObject()         {}
func (o OptionalType) isType()           {}
func (o OptionalType) Env() *Environment { return nil }
func (o OptionalType) Child(name string) Object {
	switch name {
	case "Element":
		return o.Element
	default:
		return nil
	}
}
func (o OptionalType) Keyable() bool {
	return false
}
func (o OptionalType) String() string {
	return fmt.Sprintf("%s?", o.Element)
}
func (o OptionalType) Path() Path {
	return Path{"", "Optional"}
}
func (o OptionalType) Parent() Object {
	return nil
}
func (o OptionalType) ObjectName() string {
	return "Optional"
}

var _ Type = OptionalType{}
