package typechecking

type Workspace struct {
	Name      string
	DefinedAt Path
	InEnv     *Environment

	Modules map[string]*Module
}

func (w *Workspace) Child(name string) Object {
	return w.Modules[name]
}
func (w *Workspace) Env() *Environment {
	return w.InEnv
}
func (w *Workspace) ObjectName() string {
	return w.Name
}
func (*Workspace) Parent() Object {
	return nil
}
func (w *Workspace) Path() Path {
	return w.DefinedAt
}
func (*Workspace) isObject() {
}

type Module struct {
	Name        string
	DefinedAt   Path
	InEnv       *Environment
	InWorkspace *Workspace

	Imports map[string]*Module

	Structs   []*Struct
	Enums     []*Enum
	Protocols []*Protocol
	Streams   []*Stream
	Flagsets  []*Flagset
}

var _ Object = Module{}

func (m Module) Env() *Environment {
	return m.InEnv
}
func (m Module) ObjectName() string {
	return m.Name
}
func (m Module) isObject() {}
func (m Module) Path() Path {
	return m.DefinedAt
}
func (m Module) Parent() Object {
	return m.InWorkspace
}
func (m Module) Child(name string) Object {
	for _, strct := range m.Structs {
		if strct.ObjectName() == name {
			return strct
		}
	}
	for _, enum := range m.Enums {
		if enum.ObjectName() == name {
			return enum
		}
	}
	for _, protocol := range m.Protocols {
		if protocol.ObjectName() == name {
			return protocol
		}
	}
	for _, flagset := range m.Flagsets {
		if flagset.ObjectName() == name {
			return flagset
		}
	}
	for _, stream := range m.Streams {
		if stream.ObjectName() == name {
			return stream
		}
	}
	if v, ok := m.Imports[name]; ok {
		return v
	}
	return nil
}
