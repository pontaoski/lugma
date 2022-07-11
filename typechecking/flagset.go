package typechecking

import "fmt"

type Flagset struct {
	Name          string
	Documentation string
	DefinedAt     Path

	Optional bool
	Flags    []Flag
}

var _ Type = Flagset{}

func (f Flagset) isObject() {}
func (f Flagset) isType()   {}
func (f Flagset) String() string {
	if f.Optional {
		return fmt.Sprintf("flagset %s: optional", f.Name)
	}
	return fmt.Sprintf("flagset %s", f.Name)
}
func (f Flagset) Path() Path {
	return f.DefinedAt
}
func (Flagset) Child(name string) Object { return nil }
func (Flagset) Keyable() bool {
	return false
}

type Flag struct {
	Name          string
	Documentation string
}
