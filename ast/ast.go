package ast

import (
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
)

type Span struct {
	Start, End sitter.Point
}

func SpanFromNode(n *sitter.Node) Span {
	return Span{n.StartPoint(), n.EndPoint()}
}

type File struct {
	Imports   []Import
	Protocols []Protocol
	Structs   []Struct
	Enums     []Enum
	Span      Span
}

func FileFromNode(n *sitter.Node, input []byte) File {
	var f File
	f.Span = SpanFromNode(n)
	for i := 0; i < int(n.NamedChildCount()); i++ {
		child := n.NamedChild(i).Child(0)
		switch child.Type() {
		case "import":
			f.Imports = append(f.Imports, ImportFromNode(child, input))
		case "protocol_declaration":
			f.Protocols = append(f.Protocols, ProtocolFromNode(child, input))
		case "struct_declaration":
			f.Structs = append(f.Structs, StructFromNode(child, input))
		case "enum_declaration":
			f.Enums = append(f.Enums, EnumFromNode(child, input))
		default:
			panic("Unhandled " + child.Type())
		}
	}
	return f
}

type Import struct {
	Path string
	As   string

	Span Span
}

func ImportFromNode(n *sitter.Node, input []byte) Import {
	var i Import
	i.Span = SpanFromNode(n)
	i.Path = n.ChildByFieldName("path").Content(input)
	i.Path = strings.TrimPrefix(strings.TrimSuffix(i.Path, `"`), `"`)
	i.As = n.ChildByFieldName("alias").Content(input)

	return i
}

type Protocol struct {
	Name string

	Functions []Function
	Events    []Event
	Span      Span
}

func ProtocolFromNode(n *sitter.Node, input []byte) Protocol {
	var p Protocol
	p.Span = SpanFromNode(n)
	p.Name = n.ChildByFieldName("name").Content(input)
	for i := 0; i < int(n.NamedChildCount()); i++ {
		child := n.NamedChild(i)
		switch child.Type() {
		case "func_declaration":
			p.Functions = append(p.Functions, FunctionFromNode(child, input))
		case "event_declaration":
			p.Events = append(p.Events, EventFromNode(child, input))
		case "identifier":
			continue
		default:
			panic("Unhandled " + child.Type())
		}
	}
	return p
}

type Function struct {
	Name string

	Arguments []Argument

	Returns Type
	Throws  Type
	Span    Span
}

func FunctionFromNode(n *sitter.Node, input []byte) Function {
	var f Function
	f.Span = SpanFromNode(n)

	f.Name = n.ChildByFieldName("name").Content(input)

	for i := 0; i < int(n.NamedChildCount()); i++ {
		child := n.NamedChild(i)
		switch child.Type() {
		case "arg":
			f.Arguments = append(f.Arguments, ArgumentFromNode(child, input))
		default:
			continue
		}
	}

	if child := n.ChildByFieldName("returns"); child != nil {
		f.Returns = TypeFromNode(child, input)
	}

	if child := n.ChildByFieldName("throws"); child != nil {
		f.Throws = TypeFromNode(child, input)
	}

	return f
}

type Argument struct {
	Name string
	Type Type
	Span Span
}

func ArgumentFromNode(n *sitter.Node, input []byte) Argument {
	var a Argument
	a.Span = SpanFromNode(n)

	a.Name = n.ChildByFieldName("name").Content(input)
	a.Type = TypeFromNode(n.ChildByFieldName("type"), input)

	return a
}

type Event struct {
	Name string

	Arguments []Argument
	Span      Span
}

func EventFromNode(n *sitter.Node, input []byte) Event {
	var e Event
	e.Span = SpanFromNode(n)

	e.Name = n.ChildByFieldName("name").Content(input)

	for i := 0; i < int(n.NamedChildCount()); i++ {
		child := n.NamedChild(i)
		switch child.Type() {
		case "arg":
			e.Arguments = append(e.Arguments, ArgumentFromNode(child, input))
		default:
			continue
		}
	}

	return e
}

type Struct struct {
	Name string

	Fields []Field
	Span   Span
}

func StructFromNode(n *sitter.Node, input []byte) Struct {
	var s Struct
	s.Span = SpanFromNode(n)

	s.Name = n.ChildByFieldName("name").Content(input)

	for i := 0; i < int(n.NamedChildCount()); i++ {
		child := n.NamedChild(i)
		switch child.Type() {
		case "field_declaration":
			s.Fields = append(s.Fields, FieldFromNode(child, input))
		case "identifier":
			continue
		default:
			panic("Unhandled " + child.Type())
		}
	}

	return s
}

type Field struct {
	Name string

	Type Type
	Span Span
}

func FieldFromNode(n *sitter.Node, input []byte) Field {
	var f Field
	f.Span = SpanFromNode(n)

	f.Name = n.ChildByFieldName("name").Content(input)
	f.Type = TypeFromNode(n.ChildByFieldName("type"), input)

	return f
}

type Enum struct {
	Name string

	Cases []Case
	Span  Span
}

func EnumFromNode(n *sitter.Node, input []byte) Enum {
	var e Enum
	e.Span = SpanFromNode(n)

	e.Name = n.ChildByFieldName("name").Content(input)

	for i := 0; i < int(n.NamedChildCount()); i++ {
		child := n.NamedChild(i)
		switch child.Type() {
		case "case_declaration":
			e.Cases = append(e.Cases, CaseFromNode(child, input))
		case "identifier":
			continue
		default:
			panic("Unhandled " + child.Type())
		}
	}

	return e
}

type Case struct {
	Name string

	Values []Argument
	Span   Span
}

func CaseFromNode(n *sitter.Node, input []byte) Case {
	var c Case
	c.Span = SpanFromNode(n)

	c.Name = n.ChildByFieldName("name").Content(input)

	for i := 0; i < int(n.NamedChildCount()); i++ {
		child := n.NamedChild(i)
		switch child.Type() {
		case "arg":
			c.Values = append(c.Values, ArgumentFromNode(child, input))
		case "identifier":
			continue
		default:
			panic("Unhandled " + child.Type())
		}
	}

	return c
}

type Type interface {
	isType()
	GetSpan() Span
}

func TypeFromNode(n *sitter.Node, input []byte) Type {
	switch n.ChildCount() {
	case 1: // ident
		return TypeIdent{n.Child(0).Content(input), SpanFromNode(n)}
	case 3: // array or subscript
		if n.Child(1).Type() == "." { // subscript
			return TypeSubscript{TypeFromNode(n.Child(0), input), n.Child(2).Content(input), SpanFromNode(n)}
		} else { // array
			return TypeArray{TypeFromNode(n.Child(1), input), SpanFromNode(n)}
		}
	case 5: // dict
		return TypeDictionary{TypeFromNode(n.Child(1), input), TypeFromNode(n.Child(3), input), SpanFromNode(n)}
	case 2: // optional
		return TypeOptional{TypeFromNode(n.Child(0), input), SpanFromNode(n)}
	default:
		panic("Unhandled " + n.String())
	}
}

type TypeIdent struct {
	Name string

	Span Span
}

func (t TypeIdent) GetSpan() Span { return t.Span }
func (TypeIdent) isType()         {}

type TypeArray struct {
	Inner Type

	Span Span
}

func (t TypeArray) GetSpan() Span { return t.Span }
func (TypeArray) isType()         {}

type TypeSubscript struct {
	Inner Type

	Field string

	Span Span
}

func (t TypeSubscript) GetSpan() Span { return t.Span }
func (TypeSubscript) isType()         {}

type TypeDictionary struct {
	Key   Type
	Value Type

	Span Span
}

func (t TypeDictionary) GetSpan() Span { return t.Span }
func (TypeDictionary) isType()         {}

type TypeOptional struct {
	Inner Type

	Span Span
}

func (t TypeOptional) GetSpan() Span { return t.Span }
func (TypeOptional) isType()         {}
