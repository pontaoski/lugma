package typechecking

import "fmt"

type Object interface {
	isObject()
	Child(name string) Object
	Path() Path
}

type Path struct {
	ModulePath   string
	InModulePath string
}

func (p Path) String() string {
	return fmt.Sprintf("%s%s", p.ModulePath, p.InModulePath)
}
func (p Path) Appended(path string) Path {
	return Path{p.ModulePath, p.InModulePath + "/" + path}
}
