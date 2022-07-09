package typechecking

type Module struct {
	DefinedAt Path

	Structs   []Struct
	Enums     []Enum
	Protocols []Protocol
	Flagsets  []Flagset
}

var _ Object = Module{}

func (m Module) isObject() {}
func (m Module) Path() Path {
	return m.DefinedAt
}
func (m Module) Child(name string) Object {
	for _, strct := range m.Structs {
		if strct.Name == name {
			return strct
		}
	}
	for _, enum := range m.Enums {
		if enum.Name == name {
			return enum
		}
	}
	for _, protocol := range m.Protocols {
		if protocol.Name == name {
			return protocol
		}
	}
	for _, flagset := range m.Flagsets {
		if flagset.Name == name {
			return flagset
		}
	}
	return nil
}
