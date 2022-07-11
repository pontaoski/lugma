package typechecking

import (
	"fmt"
	"lugmac/ast"
)

type Flagset struct {
	object
	Documentation *ast.ItemDocumentation

	Optional bool
	Flags    []*Flag
}

var _ Type = &Flagset{}

func (f Flagset) isType() {}
func (f Flagset) String() string {
	if f.Optional {
		return fmt.Sprintf("flagset %s: optional", f.ObjectName())
	}
	return fmt.Sprintf("flagset %s", f.ObjectName())
}
func (f Flagset) Child(name string) Object {
	for _, flag := range f.Flags {
		if flag.ObjectName() == name {
			return flag
		}
	}
	return nil
}
func (Flagset) Keyable() bool {
	return false
}

type Flag struct {
	object

	Documentation *ast.ItemDocumentation
}

var _ Object = &Flag{}

func (f Flag) Child(name string) Object {
	return nil
}
