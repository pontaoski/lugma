package typechecking

import "lugmac/ast"

type Stream struct {
	object
	Documentation *ast.ItemDocumentation

	Events  []*Event
	Signals []*Signal
}

var _ Object = &Stream{}

func (p Stream) isObject() {}

func (p Stream) Child(name string) Object {
	for _, ev := range p.Events {
		if ev.ObjectName() == name {
			return ev
		}
	}
	for _, sig := range p.Signals {
		if sig.ObjectName() == name {
			return sig
		}
	}
	return nil
}
