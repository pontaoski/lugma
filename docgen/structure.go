package docgen

import (
	"fmt"
	lugmaast "lugmac/ast"
	"lugmac/ast/extension"
	"lugmac/typechecking"
	"reflect"
	"strings"

	"github.com/yuin/goldmark/ast"
)

type Section struct {
	Title string
	Items []Item
}

func (s Section) dump(i int) {
	println(strings.Repeat("  ", i) + "* " + s.Title)
	for _, item := range s.Items {
		item.dump(i)
	}
}

type Item struct {
	Object   typechecking.Object
	Children []Section
}

func (item Item) dump(i int) {
	if item.Object != nil {
		println(strings.Repeat("  ", i) + item.Object.ObjectName())
	} else {
		println(strings.Repeat("  ", i) + "<blank name>")
	}

	for _, section := range item.Children {
		section.dump(i + 1)
	}
}

func (sect Section) renderTableOfContentsTo(sb *strings.Builder, currently typechecking.Object) {
	sb.WriteString(fmt.Sprintf(`<div class="depth-spacer flex flex-row"> <span class="codicon codicon-chevron-right opacity-0"> </span> <h3>%s</h3></div>`, sect.Title))
	for _, item := range sect.Items {
		item.renderTableOfContentsTo(sb, currently)
	}
}

func (item Item) renderTableOfContentsTo(sb *strings.Builder, currently typechecking.Object) {
	if item.Object != nil {
		url := resolveURL(item.Object, currently.Path().String())

		sb.WriteString(fmt.Sprintf(`<a href="%s" class="depth-spacer flex flex-row text-ellipsis overflow-hidden`, url))
		if typechecking.IsParentOf(item.Object, currently) {
			sb.WriteString(" is-open ")
		} else {
			sb.WriteString(" is-closed ")
		}
		if item.Object == currently {
			sb.WriteString(" is-current ")
		}
		sb.WriteString(`">`)
		sb.WriteString(`<span class="codicon codicon-chevron-right`)
		if len(item.Children) == 0 {
			sb.WriteString(" opacity-0 ")
		}
		sb.WriteString(`"> </span>`)
		switch item.Object.(type) {
		case *typechecking.Enum, *typechecking.Flagset:
			sb.WriteString(`<span class="codicon codicon-symbol-enum symbol-enum"></span>`)
		case *typechecking.Flag, *typechecking.Case:
			sb.WriteString(`<span class="codicon codicon-symbol-enum-member symbol-enum-member"></span>`)
		case *typechecking.Protocol:
			sb.WriteString(`<span class="codicon codicon-symbol-class symbol-class"></span>`)
		case *typechecking.Struct:
			sb.WriteString(`<span class="codicon codicon-symbol-structure symbol-structure"></span>`)
		case *typechecking.Event, *typechecking.Signal:
			sb.WriteString(`<span class="codicon codicon-symbol-event symbol-event"></span>`)
		case *typechecking.Func:
			sb.WriteString(`<span class="codicon codicon-symbol-method symbol-method"></span>`)
		case *typechecking.Field:
			sb.WriteString(`<span class="codicon codicon-symbol-field symbol-field"></span>`)
		}
		sb.WriteString(fmt.Sprintf(`%s</a>`, item.Object.ObjectName()))
	}
	sb.WriteString("<ul>")
	for _, sect := range item.Children {
		sb.WriteString("<li>")
		sect.renderTableOfContentsTo(sb, currently)
		sb.WriteString("</li>")
	}
	sb.WriteString("</ul>")
}

func DefaultStructureForStruct(strukt *typechecking.Struct) Item {
	var fields []Item

	for _, field := range strukt.Fields {
		fields = append(fields, Item{
			Object: field,
		})
	}

	return Item{
		Object: strukt,
		Children: []Section{
			{
				Title: "Fields",
				Items: fields,
			},
		},
	}
}

func DefaultStructureForEnum(enum *typechecking.Enum) Item {
	var fields []Item

	for _, field := range enum.Cases {
		fields = append(fields, Item{
			Object: field,
		})
	}

	return Item{
		Object: enum,
		Children: []Section{
			{
				Title: "Cases",
				Items: fields,
			},
		},
	}
}

func DefaultStructureForProtocol(protocol *typechecking.Protocol) Item {
	var funcs []Item
	var events []Item
	var signals []Item

	for _, fn := range protocol.Funcs {
		funcs = append(funcs, Item{Object: fn})
	}
	for _, ev := range protocol.Events {
		events = append(events, Item{Object: ev})
	}
	for _, sig := range protocol.Signals {
		signals = append(signals, Item{Object: sig})
	}

	ret := Item{
		Object: protocol,
	}

	if len(funcs) > 0 {
		ret.Children = append(ret.Children, Section{
			Title: "Methods",
			Items: funcs,
		})
	}
	if len(events) > 0 {
		ret.Children = append(ret.Children, Section{
			Title: "Events",
			Items: events,
		})
	}
	if len(signals) > 0 {
		ret.Children = append(ret.Children, Section{
			Title: "Signals",
			Items: signals,
		})
	}

	return ret
}

func DefaultStructureForFlagset(flagset *typechecking.Flagset) Item {
	var flags []Item

	for _, flag := range flagset.Flags {
		flags = append(flags, Item{
			Object: flag,
		})
	}

	return Item{
		Object: flagset,
		Children: []Section{
			{
				Title: "Flags",
				Items: flags,
			},
		},
	}
}

func SummaryFor(object typechecking.Object) string {
	if docs := DocumentationItemFor(object); docs != nil {
		return string(docs.Summary.Text(docs.Source))
	}
	return ""
}

func DocumentationItemFor(object typechecking.Object) *lugmaast.ItemDocumentation {
	switch t := object.(type) {
	case *typechecking.Enum:
		return t.Documentation
	case *typechecking.Struct:
		return t.Documentation
	case *typechecking.Protocol:
		return t.Documentation
	case *typechecking.Flagset:
		return t.Documentation
	case *typechecking.Func:
		return t.Documentation
	case *typechecking.Signal:
		return t.Documentation
	case *typechecking.Event:
		return t.Documentation
	default:
		return nil
	}
}

func flatten(item ast.Node) []ast.Node {
	var ret []ast.Node

	ret = append(ret, item)

	for i := item.FirstChild(); i != nil; i = i.NextSibling() {
		ret = append(ret, flatten(i)...)
	}

	return ret
}

func flattenMap(item []ast.Node) []ast.Node {
	var ret []ast.Node

	for _, item := range item {
		ret = append(ret, flatten(item)...)
	}

	return ret
}

func StructureFor(object typechecking.Object) Item {
	if docs := DocumentationItemFor(object); docs != nil && docs.HasCustomStructure {
		return CustomStructureFor(object, docs)
	}

	return DefaultStructureFor(object)
}

func CustomStructureFor(object typechecking.Object, docs *lugmaast.ItemDocumentation) Item {
	var secs []Section
	var currentlyBuilding Section
	for _, item := range flattenMap(docs.CustomStructure) {
		if item.Kind() == ast.KindHeading {
			if len(currentlyBuilding.Items) != 0 {
				secs = append(secs, currentlyBuilding)
				currentlyBuilding = Section{}
			}
			currentlyBuilding.Title = string(item.Text(docs.Source))
		} else if item.Kind() == extension.SymbolLinkKind {
			name := string(item.(*extension.SymbolLinkNode).Symbol)
			obj := object.Child(name)
			if obj == nil {
				panic("unexpectedly null object " + name)
			}
			currentlyBuilding.Items = append(currentlyBuilding.Items, Item{
				Object: obj,
			})
		}
	}
	if len(currentlyBuilding.Items) != 0 {
		secs = append(secs, currentlyBuilding)
		currentlyBuilding = Section{}
	}
	return Item{Object: object, Children: secs}
}

func hkeyword(s string) string {
	return fmt.Sprintf(`<span class="code-keyword">%s</span>`, s)
}

func hcode(s string) string {
	return fmt.Sprintf(`<span class="code">%s</span>`, s)
}

func hfield(object *typechecking.Field) string {
	return fmt.Sprintf(`%s: <span class="code-type">%s</span>`, object.Name, object.Type.String())
}

func hfields(objects []*typechecking.Field) string {
	var fields []string
	for _, obj := range objects {
		fields = append(fields, hfield(obj))
	}
	return strings.Join(fields, ", ")
}

func hparens(s string) string {
	return fmt.Sprintf(`(%s)`, s)
}

func hitem(s string) string {
	return fmt.Sprintf(`<span class="code-item-name">%s</span>`, s)
}

func HTMLSignatureFor(object typechecking.Object, currently typechecking.Object) string {
	switch t := object.(type) {
	case *typechecking.Func:
		return hcode(hkeyword("func") + " " + hitem(object.ObjectName()) + hparens(hfields(t.Arguments)))
	case *typechecking.Signal:
		return hcode(hkeyword("signal") + " " + hitem(object.ObjectName()) + hparens(hfields(t.Arguments)))
	case *typechecking.Event:
		return hcode(hkeyword("event") + " " + hitem(object.ObjectName()) + hparens(hfields(t.Arguments)))
	case *typechecking.Case:
		return hcode(hkeyword("case") + " " + hitem(object.ObjectName()) + hparens(hfields(t.Fields)))
	case *typechecking.Field:
		return hcode(hkeyword("let") + " " + hitem(object.ObjectName()))
	case *typechecking.Struct:
		return hcode(hkeyword("struct") + " " + hitem(object.ObjectName()))
	case *typechecking.Protocol:
		return hcode(hkeyword("protocol") + " " + hitem(object.ObjectName()))
	default:
		panic("bad object type " + reflect.TypeOf(object).String())
	}
}

func IsStructuralObject(object typechecking.Object) bool {
	switch object.(type) {
	case *typechecking.Module:
		return true
	case *typechecking.Enum:
		return true
	case *typechecking.Struct:
		return true
	case *typechecking.Protocol:
		return true
	case *typechecking.Flagset:
		return true
	default:
		return false
	}
}

func DefaultStructureFor(object typechecking.Object) Item {
	switch t := object.(type) {
	case *typechecking.Module:
		return DefaultStructureForModule(t)
	case *typechecking.Enum:
		return DefaultStructureForEnum(t)
	case *typechecking.Struct:
		return DefaultStructureForStruct(t)
	case *typechecking.Protocol:
		return DefaultStructureForProtocol(t)
	case *typechecking.Flagset:
		return DefaultStructureForFlagset(t)
	default:
		panic("bad item type")
	}
}

func DefaultStructureForModule(m *typechecking.Module) Item {
	structSection := Section{Title: "Structs"}
	for _, strukt := range m.Structs {
		structSection.Items = append(structSection.Items, StructureFor(strukt))
	}

	enumSection := Section{Title: "Enums"}
	for _, enum := range m.Enums {
		enumSection.Items = append(enumSection.Items, StructureFor(enum))
	}

	protocolSection := Section{Title: "Protocols"}
	for _, protocol := range m.Protocols {
		protocolSection.Items = append(protocolSection.Items, StructureFor(protocol))
	}

	flagsetSection := Section{Title: "Flagsets"}
	for _, flagset := range m.Flagsets {
		flagsetSection.Items = append(flagsetSection.Items, StructureFor(flagset))
	}

	ret := Item{}

	if len(structSection.Items) > 0 {
		ret.Children = append(ret.Children, structSection)
	}
	if len(enumSection.Items) > 0 {
		ret.Children = append(ret.Children, enumSection)
	}
	if len(protocolSection.Items) > 0 {
		ret.Children = append(ret.Children, protocolSection)
	}
	if len(flagsetSection.Items) > 0 {
		ret.Children = append(ret.Children, flagsetSection)
	}

	return ret
}
