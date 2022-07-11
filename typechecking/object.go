package typechecking

import "fmt"

type Object interface {
	isObject()
	ObjectName() string
	Parent() Object
	Child(name string) Object
	Path() Path
	Env() *Environment
}

type object struct {
	name      string
	definedAt Path
	inParent  Object
	inEnv     *Environment
}

func newObject(name string, definedAt Path, inparent Object, inEnv *Environment) object {
	return object{
		name:      name,
		definedAt: definedAt,
		inParent:  inparent,
		inEnv:     inEnv,
	}
}

func (o *object) isObject() {}
func (o *object) ObjectName() string {
	return o.name
}
func (o *object) Parent() Object {
	return o.inParent
}
func (o *object) Path() Path {
	return o.definedAt
}
func (o *object) Env() *Environment {
	return o.inEnv
}

func IsParentOf(par Object, child Object) bool {
	for child != nil {
		if par == child {
			return true
		}
		child = child.Parent()
	}
	return false
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
