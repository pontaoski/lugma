package typechecking

type Protocol struct {
	Name      string
	DefinedAt Path

	Funcs  []Func
	Events []Event
}

var _ Object = Protocol{}

func (p Protocol) isObject() {}
func (p Protocol) Path() Path {
	return p.DefinedAt
}
func (p Protocol) Child(name string) Object {
	for _, fn := range p.Funcs {
		if fn.Name == name {
			return fn
		}
	}
	for _, ev := range p.Events {
		if ev.Name == name {
			return ev
		}
	}
	return nil
}

type Func struct {
	Name      string
	DefinedAt Path

	Arguments []Field

	Returns Type
	Throws  Type
}

func (f Func) isObject()                {}
func (f Func) Child(name string) Object { return nil }
func (f Func) Path() Path {
	return f.DefinedAt
}

type Event struct {
	Name      string
	DefinedAt Path

	Arguments []Field
}

func (f Event) isObject()                {}
func (f Event) Child(name string) Object { return nil }
func (f Event) Path() Path {
	return f.DefinedAt
}
