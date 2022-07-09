package typechecking

import (
	"fmt"
	"io/ioutil"
	"lugmac/ast"

	lugma "lugmac/parser"

	sitter "github.com/smacker/go-tree-sitter"
)

type Context struct {
	Environment  *Environment
	KnownModules map[string]*Module
}

func NewContext() *Context {
	return &Context{World, map[string]*Module{}}
}

func (c *Context) PushEnvironment() *Environment {
	c.Environment = &Environment{map[string]Object{}, c.Environment}
	return c.Environment
}

func (c *Context) PopEnvironment() *Environment {
	a := c.Environment
	c.Environment = c.Environment.Parent
	return a
}

func lookupType(typ ast.Type, in *Context) (Type, error) {
	switch typ := typ.(type) {
	case ast.TypeIdent:
		object, found := in.Environment.Search(typ.Name)
		if !found {
			return nil, fmt.Errorf("Type %s not found", typ.Name)
		}
		kind, ok := object.(Type)
		if !ok {
			return nil, fmt.Errorf("%s is not a type", typ.Name)
		}
		return kind, nil
	case ast.TypeArray:
		element, err := lookupType(typ.Inner, in)
		if err != nil {
			return nil, err
		}
		return ArrayType{element}, nil
	case ast.TypeDictionary:
		key, err := lookupType(typ.Key, in)
		if err != nil {
			return nil, err
		}
		if !key.Keyable() {
			return nil, fmt.Errorf("%s is not a valid type to use as a dictionary key", key)
		}

		val, err := lookupType(typ.Value, in)
		if err != nil {
			return nil, err
		}

		return DictionaryType{key, val}, nil
	case ast.TypeOptional:
		element, err := lookupType(typ.Inner, in)
		if err != nil {
			return nil, err
		}
		return OptionalType{element}, nil
	case ast.TypeSubscript:
		var element Object

		if v, ok := typ.Inner.(ast.TypeIdent); ok {
			element, ok = in.Environment.Search(v.Name)
			if !ok {
				return nil, fmt.Errorf("Object %s not found", v.Name)
			}
		} else {
			var err error
			element, err = lookupType(typ.Inner, in)
			if err != nil {
				return nil, err
			}
		}
		child := element.Child(typ.Field)
		if child == nil {
			return nil, fmt.Errorf("Object %s has no field %s", element, typ.Field)
		}
		kind, ok := child.(Type)
		if !ok {
			return nil, fmt.Errorf("Field %s on %s is not a type", typ.Field, kind)
		}
		return kind, nil
	case nil:
		return nil, nil
	default:
		panic("Unhandled AST kind")
	}
}

func fieldList(fields []ast.Field, parentPath Path, in *Context) ([]Field, error) {
	var fs []Field

	for _, field := range fields {
		var f Field

		f.Name = field.Name
		f.DefinedAt = parentPath.Appended(f.Name)
		typ, err := lookupType(field.Type, in)
		if err != nil {
			return nil, err
		}
		f.Type = typ

		fs = append(fs, f)
	}

	return fs, nil
}

func argList(fields []ast.Argument, parentPath Path, in *Context) ([]Field, error) {
	var fs []Field

	for _, field := range fields {
		var f Field

		f.Name = field.Name
		f.DefinedAt = parentPath.Appended(f.Name)
		typ, err := lookupType(field.Type, in)
		if err != nil {
			return nil, err
		}
		f.Type = typ

		fs = append(fs, f)
	}

	return fs, nil
}

func (ctx *Context) MakeModule(at string) error {
	_, err := ctx.ModuleFor(at)
	return err
}

func (ctx *Context) ModuleFor(path string) (*Module, error) {
	if v, ok := ctx.KnownModules[path]; ok {
		return v, nil
	}

	parser := sitter.NewParser()
	parser.SetLanguage(lugma.GetLanguage())

	file, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("Failed to load module at %s: %w", path, err)
	}

	tree := parser.Parse(nil, file)

	fileAST := ast.FileFromNode(tree.RootNode(), file)

	module, err := ctx.Module(&fileAST, path)
	if err != nil {
		return nil, fmt.Errorf("Failed to load module at %s: %w", path, err)
	}

	ctx.KnownModules[path] = module
	return module, nil
}

func (ctx *Context) Module(tree *ast.File, path string) (*Module, error) {
	var m Module
	m.DefinedAt = Path{path, ""}

	ctx.PushEnvironment()
	defer ctx.PopEnvironment()

	for _, imports := range tree.Imports {
		module, err := ctx.ModuleFor(imports.Path)
		if err != nil {
			return nil, err
		}
		ctx.Environment.Items[imports.As] = module
	}
	for _, item := range tree.Structs {
		var s Struct
		s.Name = item.Name
		s.DefinedAt = m.DefinedAt.Appended(item.Name)

		fields, err := fieldList(item.Fields, s.DefinedAt, ctx)
		if err != nil {
			return nil, err
		}
		s.Fields = fields

		m.Structs = append(m.Structs, s)
		ctx.Environment.Items[item.Name] = s
	}
	for _, item := range tree.Enums {
		var e Enum
		e.Name = item.Name
		e.DefinedAt = m.DefinedAt.Appended(item.Name)

		for _, cas := range item.Cases {
			var c Case
			c.Name = item.Name
			c.DefinedAt = e.DefinedAt.Appended(cas.Name)

			args, err := argList(cas.Values, c.DefinedAt, ctx)
			if err != nil {
				return nil, err
			}

			c.Fields = args
			e.Cases = append(e.Cases, c)
		}

		m.Enums = append(m.Enums, e)
		ctx.Environment.Items[item.Name] = e
	}
	for _, item := range tree.Protocols {
		var p Protocol
		p.Name = item.Name
		p.DefinedAt = m.DefinedAt.Appended(item.Name)

		for _, fn := range item.Functions {
			var f Func

			f.Name = fn.Name
			f.DefinedAt = p.DefinedAt.Appended(f.Name)
			args, err := argList(fn.Arguments, f.DefinedAt, ctx)
			if err != nil {
				return nil, err
			}
			f.Arguments = args
			f.Returns, err = lookupType(fn.Returns, ctx)
			if err != nil {
				return nil, err
			}
			f.Throws, err = lookupType(fn.Throws, ctx)
			if err != nil {
				return nil, err
			}

			p.Funcs = append(p.Funcs, f)
		}
		for _, ev := range item.Events {
			var e Event
			e.Name = ev.Name
			e.DefinedAt = p.DefinedAt.Appended(e.Name)

			args, err := argList(ev.Arguments, e.DefinedAt, ctx)
			if err != nil {
				return nil, err
			}
			e.Arguments = args

			p.Events = append(p.Events, e)
		}

		m.Protocols = append(m.Protocols, p)
		ctx.Environment.Items[item.Name] = p
	}

	return &m, nil
}
