package ast

import sitter "github.com/smacker/go-tree-sitter"

type File struct {
	Protocols []Protocol
	Structs   []Struct
	Enums     []Enum
}

func FileFromNode(n *sitter.Node, input []byte) File {
	var f File
	for i := 0; i < int(n.NamedChildCount()); i++ {
		child := n.NamedChild(i).Child(0)
		switch child.Type() {
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

type Protocol struct {
	Name string

	Functions []Function
	Events    []Event
}

func ProtocolFromNode(n *sitter.Node, input []byte) Protocol {
	var p Protocol
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
}

func FunctionFromNode(n *sitter.Node, input []byte) Function {
	var f Function

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
}

func ArgumentFromNode(n *sitter.Node, input []byte) Argument {
	var a Argument

	a.Name = n.ChildByFieldName("name").Content(input)
	a.Type = TypeFromNode(n.ChildByFieldName("type"), input)

	return a
}

type Event struct {
	Name string

	Arguments []Argument
}

func EventFromNode(n *sitter.Node, input []byte) Event {
	var e Event

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
}

func StructFromNode(n *sitter.Node, input []byte) Struct {
	var s Struct

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
}

func FieldFromNode(n *sitter.Node, input []byte) Field {
	var f Field

	f.Name = n.ChildByFieldName("name").Content(input)
	f.Type = TypeFromNode(n.ChildByFieldName("type"), input)

	return f
}

type Enum struct {
	Name string

	Cases []Case
}

func EnumFromNode(n *sitter.Node, input []byte) Enum {
	var e Enum

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
}

func CaseFromNode(n *sitter.Node, input []byte) Case {
	var c Case

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
}

func TypeFromNode(n *sitter.Node, input []byte) Type {
	switch n.ChildCount() {
	case 1: // ident
		return TypeIdent{n.Child(0).Content(input)}
	case 3: // array
		return TypeArray{TypeFromNode(n.Child(1), input)}
	case 5: // dict
		return TypeDictionary{TypeFromNode(n.Child(1), input), TypeFromNode(n.Child(3), input)}
	case 2: // optional
		return TypeOptional{TypeFromNode(n.Child(0), input)}
	default:
		panic("Unhandled " + n.String())
	}
}

type TypeIdent struct {
	Name string
}

func (TypeIdent) isType() {}

type TypeArray struct {
	Inner Type
}

func (TypeArray) isType() {}

type TypeDictionary struct {
	Key   Type
	Value Type
}

func (TypeDictionary) isType() {}

type TypeOptional struct {
	Inner Type
}

func (TypeOptional) isType() {}
