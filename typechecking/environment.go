package typechecking

type Environment struct {
	Items map[string]Object

	Parent *Environment
}

func (e *Environment) Search(name string) (Object, bool) {
	if v, ok := e.Items[name]; ok {
		return v, true
	}
	if e.Parent == nil {
		return nil, false
	}
	return e.Parent.Search(name)
}

var World = &Environment{
	Items: map[string]Object{
		"UInt8":  UInt8,
		"UInt16": UInt16,
		"UInt32": UInt32,
		"UInt64": UInt64,

		"Int8":  Int8,
		"Int16": Int16,
		"Int32": Int32,
		"Int64": Int64,

		"String": String,
		"Bytes":  Bytes,

		"Bool": Bool,
	},
}
