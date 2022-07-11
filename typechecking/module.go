package typechecking

type Module struct {
	Name      string
	DefinedAt Path
	InEnv     *Environment

	Imports map[string]*Module

	Structs   []*Struct
	Enums     []*Enum
	Protocols []*Protocol
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
	return nil
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
	if v, ok := m.Imports[name]; ok {
		return v
	}
	return nil
}
