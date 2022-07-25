package ast

import (
	"fmt"
	"regexp"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
)

type Span struct {
	Start, End sitter.Point
}

func (s Span) String() string {
	return fmt.Sprintf("%d:%d", s.Start.Row, s.Start.Column)
}

func SpanFromNode(n *sitter.Node) Span {
	return Span{n.StartPoint(), n.EndPoint()}
}

type File struct {
	Imports   []Import
	Protocols []Protocol
	Streams   []Stream
	Structs   []Struct
	Enums     []Enum
	Flagsets  []Flagset
	Span      Span
}

var (
	whitespaceOnly    = regexp.MustCompile("(?m)^[ \t]+$")
	leadingWhitespace = regexp.MustCompile("(?m)(^[ \t]*)(?:[^ \t\n])")
)

// dedent removes any common leading whitespace from every line in text.
//
// This can be used to make multiline strings to line up with the left edge of
// the display, while still presenting them in the source code in indented
// form.
func dedent(text string) string {
	var margin string

	text = whitespaceOnly.ReplaceAllString(text, "")
	indents := leadingWhitespace.FindAllStringSubmatch(text, -1)

	// Look for the longest leading string of spaces and tabs common to all
	// lines.
	for i, indent := range indents {
		if i == 0 {
			margin = indent[1]
		} else if strings.HasPrefix(indent[1], margin) {
			// Current line more deeply indented than previous winner:
			// no change (previous winner is still on top).
			continue
		} else if strings.HasPrefix(margin, indent[1]) {
			// Current line consistent with and no deeper than previous winner:
			// it's the new winner.
			margin = indent[1]
		} else {
			// Current line and previous winner have no common whitespace:
			// there is no margin.
			margin = ""
			break
		}
	}

	if margin != "" {
		text = regexp.MustCompile("(?m)^"+margin).ReplaceAllString(text, "")
	}
	return text
}

func DocumentationFromNode(n *sitter.Node, input []byte) *ItemDocumentation {
	sib := n.PrevNamedSibling()
	if n.Parent().Type() == "statement" {
		sib = n.Parent().PrevNamedSibling()
	}
	if sib == nil || sib.Type() != "comment" {
		return nil
	}

	cont := sib.Content(input)

	if !strings.HasPrefix(cont, "/**") {
		return nil
	}

	cont = strings.TrimPrefix(strings.TrimSuffix(cont, "*/"), "/**")

	lines := strings.Split(cont, "\n")
	for strings.TrimSpace(lines[0]) == "" {
		lines = lines[1:]
	}
	for strings.TrimSpace(lines[len(lines)-1]) == "" {
		lines = lines[:len(lines)-1]
	}

	cont = strings.Join(lines, "\n")
	cont = dedent(cont)
	cont = strings.TrimSpace(cont)

	return FromDocumentationComment(cont, n.Type() == "func_declaration" || n.Type() == "event_declaration" || n.Type() == "signal_declaration")
}

func CombineFiles(files ...*File) *File {
	ret := &File{}

	for _, file := range files {
		for _, I := range file.Imports {
			ret.Imports = append(ret.Imports, I)
		}
		for _, P := range file.Protocols {
			ret.Protocols = append(ret.Protocols, P)
		}
		for _, S := range file.Structs {
			ret.Structs = append(ret.Structs, S)
		}
		for _, E := range file.Enums {
			ret.Enums = append(ret.Enums, E)
		}
		for _, F := range file.Flagsets {
			ret.Flagsets = append(ret.Flagsets, F)
		}
		for _, S := range file.Streams {
			ret.Streams = append(ret.Streams, S)
		}
	}

	return ret
}

func FileFromNode(n *sitter.Node, input []byte) File {
	var f File
	f.Span = SpanFromNode(n)
	for i := 0; i < int(n.NamedChildCount()); i++ {
		if n.NamedChild(i).Type() == "comment" {
			continue
		}

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
		case "stream_declaration":
			f.Streams = append(f.Streams, StreamFromNode(child, input))
		case "flagset_declaration":
			f.Flagsets = append(f.Flagsets, FlagsetFromNode(child, input))
		case "comment":
			continue
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
	Name          string
	Documentation *ItemDocumentation

	Functions []Function
	Span      Span
}

func ProtocolFromNode(n *sitter.Node, input []byte) Protocol {
	var p Protocol
	p.Span = SpanFromNode(n)
	p.Name = n.ChildByFieldName("name").Content(input)
	p.Documentation = DocumentationFromNode(n, input)

	for i := 0; i < int(n.NamedChildCount()); i++ {
		child := n.NamedChild(i)
		switch child.Type() {
		case "func_declaration":
			p.Functions = append(p.Functions, FunctionFromNode(child, input))
		case "identifier", "comment":
			continue
		default:
			panic("Unhandled " + child.Type())
		}
	}
	return p
}

type Stream struct {
	Name          string
	Documentation *ItemDocumentation

	Events  []Event
	Signals []Signal
	Span    Span
}

func StreamFromNode(n *sitter.Node, input []byte) Stream {
	var s Stream
	s.Span = SpanFromNode(n)
	s.Name = n.ChildByFieldName("name").Content(input)
	s.Documentation = DocumentationFromNode(n, input)

	for i := 0; i < int(n.NamedChildCount()); i++ {
		child := n.NamedChild(i)
		switch child.Type() {
		case "event_declaration":
			s.Events = append(s.Events, EventFromNode(child, input))
		case "signal_declaration":
			s.Signals = append(s.Signals, SignalFromNode(child, input))
		case "identifier", "comment":
			continue
		default:
			panic("Unhandled " + child.Type())
		}
	}

	return s
}

type Function struct {
	Name          string
	Documentation *ItemDocumentation

	Arguments []Argument

	Returns Type
	Throws  Type
	Span    Span
}

func FunctionFromNode(n *sitter.Node, input []byte) Function {
	var f Function
	f.Span = SpanFromNode(n)

	f.Name = n.ChildByFieldName("name").Content(input)
	f.Documentation = DocumentationFromNode(n, input)

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
	Name          string
	Documentation *ItemDocumentation

	Arguments []Argument
	Span      Span
}

func EventFromNode(n *sitter.Node, input []byte) Event {
	var e Event
	e.Span = SpanFromNode(n)
	e.Documentation = DocumentationFromNode(n, input)

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

type Signal struct {
	Name          string
	Documentation *ItemDocumentation

	Arguments []Argument
	Span      Span
}

func SignalFromNode(n *sitter.Node, input []byte) Signal {
	var s Signal
	s.Span = SpanFromNode(n)
	s.Documentation = DocumentationFromNode(n, input)

	s.Name = n.ChildByFieldName("name").Content(input)

	for i := 0; i < int(n.NamedChildCount()); i++ {
		child := n.NamedChild(i)
		switch child.Type() {
		case "arg":
			s.Arguments = append(s.Arguments, ArgumentFromNode(child, input))
		default:
			continue
		}
	}

	return s
}

type Struct struct {
	Name          string
	Documentation *ItemDocumentation

	Fields []Field
	Span   Span
}

func StructFromNode(n *sitter.Node, input []byte) Struct {
	var s Struct
	s.Span = SpanFromNode(n)
	s.Documentation = DocumentationFromNode(n, input)

	s.Name = n.ChildByFieldName("name").Content(input)

	for i := 0; i < int(n.NamedChildCount()); i++ {
		child := n.NamedChild(i)
		switch child.Type() {
		case "field_declaration":
			s.Fields = append(s.Fields, FieldFromNode(child, input))
		case "identifier", "comment":
			continue
		default:
			panic("Unhandled " + child.Type())
		}
	}

	return s
}

type Field struct {
	Name          string
	Documentation *ItemDocumentation

	Type Type
	Span Span
}

func FieldFromNode(n *sitter.Node, input []byte) Field {
	var f Field
	f.Span = SpanFromNode(n)
	f.Documentation = DocumentationFromNode(n, input)

	f.Name = n.ChildByFieldName("name").Content(input)
	f.Type = TypeFromNode(n.ChildByFieldName("type"), input)

	return f
}

type Enum struct {
	Name          string
	Documentation *ItemDocumentation

	Cases []Case
	Span  Span
}

func EnumFromNode(n *sitter.Node, input []byte) Enum {
	var e Enum
	e.Span = SpanFromNode(n)
	e.Documentation = DocumentationFromNode(n, input)

	e.Name = n.ChildByFieldName("name").Content(input)

	for i := 0; i < int(n.NamedChildCount()); i++ {
		child := n.NamedChild(i)
		switch child.Type() {
		case "case_declaration":
			e.Cases = append(e.Cases, CaseFromNode(child, input))
		case "identifier", "comment":
			continue
		default:
			panic("Unhandled " + child.Type())
		}
	}

	return e
}

type Case struct {
	Name          string
	Documentation *ItemDocumentation

	Values []Argument
	Span   Span
}

func CaseFromNode(n *sitter.Node, input []byte) Case {
	var c Case
	c.Span = SpanFromNode(n)
	c.Documentation = DocumentationFromNode(n, input)

	c.Name = n.ChildByFieldName("name").Content(input)

	for i := 0; i < int(n.NamedChildCount()); i++ {
		child := n.NamedChild(i)
		switch child.Type() {
		case "arg":
			c.Values = append(c.Values, ArgumentFromNode(child, input))
		case "identifier", "comment":
			continue
		default:
			panic("Unhandled " + child.Type())
		}
	}

	return c
}

type Flagset struct {
	Name          string
	Documentation *ItemDocumentation

	Optional bool
	Flags    []Flag

	Span Span
}

type Flag struct {
	Name          string
	Documentation *ItemDocumentation

	Span Span
}

func FlagsetFromNode(n *sitter.Node, input []byte) Flagset {
	var f Flagset
	f.Name = n.ChildByFieldName("name").Content(input)
	f.Span = SpanFromNode(n)
	f.Documentation = DocumentationFromNode(n, input)

	if n.ChildByFieldName("optional") != nil {
		f.Optional = true
	}

	for i := 0; i < int(n.NamedChildCount()); i++ {
		child := n.NamedChild(i)
		switch child.Type() {
		case "identifier", "optional", "comment":
			continue
		case "flag_declaration":
			f.Flags = append(f.Flags, Flag{child.ChildByFieldName("name").Content(input), DocumentationFromNode(child, input), SpanFromNode(child)})
		default:
			panic("Unhandled node " + child.Type())
		}
	}

	return f
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
