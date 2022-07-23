package typechecking

import (
	"fmt"
	"io/ioutil"
	"lugmac/ast"
	"path"

	lugma "lugmac/parser"

	sitter "github.com/smacker/go-tree-sitter"
)

type ImportResolver interface {
	ModuleFor(context *Context, import_ string, from string) (*Module, error)
}

type Context struct {
	Environment    *Environment
	ImportResolver ImportResolver
}

func NewContext(i ImportResolver) *Context {
	return &Context{World, i}
}

type fileImportResolver struct {
}

func (f *fileImportResolver) ModuleFor(ctx *Context, path string, from string) (*Module, error) {
	parser := sitter.NewParser()
	parser.SetLanguage(lugma.GetLanguage())

	file, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to load module at %s: %w", path, err)
	}

	tree := parser.Parse(nil, file)

	fileAST := ast.FileFromNode(tree.RootNode(), file)

	module, err := ctx.Module(&fileAST, path)
	if err != nil {
		return nil, fmt.Errorf("failed to load module at %s: %w", path, err)
	}

	return module, nil
}

var FileImportResolver ImportResolver = &fileImportResolver{}

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

func fieldList(fields []ast.Field, parentPath Path, parent Object, in *Context) ([]*Field, error) {
	var fs []*Field

	for _, field := range fields {
		f := &Field{}

		f.Name = field.Name
		f.DefinedAt = parentPath.Appended(f.Name)
		f.InParent = parent

		typ, err := lookupType(field.Type, in)
		if err != nil {
			return nil, err
		}
		f.Type = typ

		fs = append(fs, f)
	}

	return fs, nil
}

func argList(fields []ast.Argument, parentPath Path, parent Object, in *Context) ([]*Field, error) {
	var fs []*Field

	for _, field := range fields {
		f := &Field{}
		f.Name = field.Name
		f.DefinedAt = parentPath.Appended(f.Name)
		f.InParent = parent

		typ, err := lookupType(field.Type, in)
		if err != nil {
			return nil, err
		}
		f.Type = typ

		fs = append(fs, f)
	}

	return fs, nil
}

func (ctx *Context) ModuleFor(path, from string) (*Module, error) {
	module, err := ctx.ImportResolver.ModuleFor(ctx, path, from)
	if err != nil {
		return nil, err
	}

	return module, nil
}

func (ctx *Context) MultiFileModule(trees []*ast.File, w *Workspace, modpath string) (*Module, error) {
	m := &Module{}
	m.DefinedAt = Path{modpath, ""}
	m.Name = path.Base(modpath)

	ctx.PushEnvironment()
	defer ctx.PopEnvironment()

	megaFile := ast.CombineFiles(trees...)

	err := ctx.doSingleModule(m, w, megaFile)
	if err != nil {
		return nil, err
	}

	return m, nil
}

func (ctx *Context) doSingleModule(m *Module, w *Workspace, tree *ast.File) error {
	m.InWorkspace = w
	for _, imports := range tree.Imports {
		module, err := ctx.ModuleFor(imports.Path, m.DefinedAt.ModulePath)
		if err != nil {
			return err
		}
		ctx.Environment.Items[imports.As] = module
	}
	for _, item := range tree.Structs {
		s := &Struct{}
		s.object = newObject(item.Name, m.DefinedAt.Appended(item.Name), m, ctx.Environment)
		s.Documentation = item.Documentation

		fields, err := fieldList(item.Fields, s.Path(), s, ctx)
		if err != nil {
			return err
		}
		s.Fields = fields

		m.Structs = append(m.Structs, s)
		ctx.Environment.Items[item.Name] = s
	}
	for _, item := range tree.Enums {
		e := &Enum{}
		e.object = newObject(item.Name, m.DefinedAt.Appended(item.Name), m, ctx.Environment)
		e.Documentation = item.Documentation

		for _, cas := range item.Cases {
			c := &Case{}
			c.object = newObject(cas.Name, e.Path().Appended(cas.Name), e, ctx.Environment)
			c.Documentation = cas.Documentation

			args, err := argList(cas.Values, c.Path(), c, ctx)
			if err != nil {
				return err
			}

			c.Fields = args
			e.Cases = append(e.Cases, c)
		}

		m.Enums = append(m.Enums, e)
		ctx.Environment.Items[item.Name] = e
	}
	for _, item := range tree.Flagsets {
		fs := &Flagset{}
		fs.object = newObject(item.Name, m.DefinedAt.Appended(item.Name), m, ctx.Environment)
		fs.Documentation = item.Documentation

		for _, flag := range item.Flags {
			f := &Flag{}
			f.object = newObject(flag.Name, f.Path().Appended(flag.Name), f, ctx.Environment)
			f.Documentation = flag.Documentation

			fs.Flags = append(fs.Flags, f)
		}

		m.Flagsets = append(m.Flagsets, fs)
		ctx.Environment.Items[item.Name] = fs
	}
	for _, item := range tree.Protocols {
		p := &Protocol{}
		p.object = newObject(item.Name, m.DefinedAt.Appended(item.Name), m, ctx.Environment)
		p.Documentation = item.Documentation

		for _, fn := range item.Functions {
			f := &Func{}
			f.object = newObject(fn.Name, p.Path().Appended(fn.Name), p, ctx.Environment)
			f.Documentation = fn.Documentation

			args, err := argList(fn.Arguments, f.Path(), f, ctx)
			if err != nil {
				return err
			}
			f.Arguments = args
			f.Returns, err = lookupType(fn.Returns, ctx)
			if err != nil {
				return err
			}
			f.Throws, err = lookupType(fn.Throws, ctx)
			if err != nil {
				return err
			}

			p.Funcs = append(p.Funcs, f)
		}
		for _, ev := range item.Events {
			e := &Event{}
			e.object = newObject(ev.Name, p.Path().Appended(ev.Name), p, ctx.Environment)
			e.Documentation = ev.Documentation

			args, err := argList(ev.Arguments, e.Path(), e, ctx)
			if err != nil {
				return err
			}
			e.Arguments = args

			p.Events = append(p.Events, e)
		}
		for _, sig := range item.Signals {
			s := &Signal{}
			s.object = newObject(sig.Name, p.Path().Appended(sig.Name), p, ctx.Environment)
			s.Documentation = sig.Documentation

			args, err := argList(sig.Arguments, s.Path(), s, ctx)
			if err != nil {
				return err
			}
			s.Arguments = args

			p.Signals = append(p.Signals, s)
		}

		m.Protocols = append(m.Protocols, p)
		ctx.Environment.Items[item.Name] = p
	}

	return nil
}

func (ctx *Context) Module(tree *ast.File, modpath string) (*Module, error) {
	m := &Module{}
	m.DefinedAt = Path{modpath, ""}
	m.Name = path.Base(modpath)

	ctx.PushEnvironment()
	defer ctx.PopEnvironment()

	err := ctx.doSingleModule(m, nil, tree)
	if err != nil {
		return nil, err
	}

	return m, nil
}
