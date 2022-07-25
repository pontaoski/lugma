package typescript

import (
	"fmt"
	"io/fs"
	"io/ioutil"
	"lugmac/backends"
	"lugmac/modules"
	"lugmac/typechecking"
	"os"
	"path"

	"github.com/iancoleman/strcase"
	"github.com/urfave/cli/v2"
)

type TypescriptBackend struct {
}

var _ backends.Backend = TypescriptBackend{}

func (ts TypescriptBackend) TSTypeOf(lugma typechecking.Type, module typechecking.Path, in *typechecking.Context) string {
	switch k := lugma.(type) {
	case typechecking.PrimitiveType:
		switch k {
		case typechecking.UInt8, typechecking.UInt16, typechecking.UInt32, typechecking.Int8, typechecking.Int16, typechecking.Int32:
			return "number"
		case typechecking.Int64, typechecking.UInt64, typechecking.String, typechecking.Bytes:
			return "string"
		case typechecking.Bool:
			return "bool"
		default:
			panic("unhandled primitive " + k.String())
		}
	case typechecking.ArrayType:
		return fmt.Sprintf("Array<%s>", ts.TSTypeOf(k.Element, module, in))
	case typechecking.DictionaryType:
		return fmt.Sprintf("[%s: %s]", ts.TSTypeOf(k.Key, module, in), ts.TSTypeOf(k.Element, module, in))
	case typechecking.OptionalType:
		return fmt.Sprintf("(%s|null|undefined)", ts.TSTypeOf(k.Element, module, in))
	case *typechecking.Struct, *typechecking.Enum:
		if k.Path().ModulePath == module.ModulePath {
			return k.String()
		}
		return fmt.Sprintf("TODO")
	default:
		panic("unhandled " + k.String())
	}
}

func init() {
	backends.RegisterBackend(TypescriptBackend{})
}

func (ts TypescriptBackend) GenerateCommand() *cli.Command {
	return &cli.Command{
		Name:    "typescript",
		Aliases: []string{"ts"},
		Usage:   "Generate TypeScript modules for Lugma",
		Flags:   backends.StandardFlags,
		Action: func(cCtx *cli.Context) error {
			w, err := modules.LoadWorkspaceFrom(cCtx.String("workspace"))
			if err != nil {
				return err
			}
			err = w.GenerateModules()
			if err != nil {
				return err
			}

			outdir := cCtx.String("outdir")
			err = os.MkdirAll(path.Join(outdir), 0750)
			if err != nil {
				return err
			}

			for _, prod := range w.Module.Products {
				mod := w.KnownModules[prod.Name]

				result, err := ts.Generate(mod, w.Context)
				if err != nil {
					return err
				}

				err = ioutil.WriteFile(path.Join(outdir, mod.Name+".ts"), []byte(result), fs.ModePerm)
				if err != nil {
					return err
				}
			}

			return nil
		},
	}
}

func (ts TypescriptBackend) Generate(mod *typechecking.Module, in *typechecking.Context) (string, error) {
	build := backends.Filebuilder{}

	build.Add(`import { Transport, Stream } from 'lugma-web-helpers'`)

	for _, item := range mod.Structs {
		build.AddI("export interface %s {", item.ObjectName())
		for _, field := range item.Fields {
			build.Add(`%s: %s`, field.ObjectName(), ts.TSTypeOf(field.Type, mod.Path(), in))
		}
		build.AddD("}")
	}
	for _, item := range mod.Enums {
		build.AddI("export type %s =", item.ObjectName())
		simple := item.Simple()
		for idx, esac := range item.Cases {
			if simple {
				if idx != len(item.Cases)-1 {
					build.Add(`"%s" |`, esac.ObjectName())
				} else {
					build.Add(`"%s"`, esac.ObjectName())
				}
			} else {
				build.AddE(`{ %s: {`, esac.ObjectName())
				for _, field := range esac.Fields {
					build.AddK(`%s: %s;`, field.ObjectName(), ts.TSTypeOf(field.Type, mod.Path(), in))
				}
				build.AddK(`} }`)
				if idx != len(item.Cases)-1 {
					build.AddK(` |`)
				}
				build.AddNL()
				// build.Add(`"%s" |`, esac.ObjectName())
			}
		}
		build.Einzug--
	}
	for _, item := range mod.Flagsets {
		build.Add(`export type %s = string`, item.ObjectName())
	}
	for _, stream := range mod.Streams {
		build.AddI(`export interface %s extends Stream {`, stream.ObjectName())

		for _, ev := range stream.Events {
			build.AddE(`on%s(callback: (`, strcase.ToCamel(ev.ObjectName()))
			for idx, arg := range ev.Arguments {
				build.AddK(`%s: %s`, arg.ObjectName(), ts.TSTypeOf(arg.Type, mod.Path(), in))
				if idx != len(ev.Arguments)-1 {
					build.AddK(`, `)
				}
			}
			build.AddK(`) => void): number`)
			build.AddNL()
		}

		build.AddD(`}`)

		build.AddI(`export function open%sFromTransport<T>(transport: Transport<T>, extra: T | undefined): %s {`, stream.ObjectName(), stream.ObjectName())
		{
			build.AddI(`return Object.create(`)
			build.Add(`transport.openStream("%s", extra),`, stream.Path().String())
			build.AddI(`{`)

			for _, ev := range stream.Events {
				build.AddI(`on%s: {`, ev.ObjectName())
				build.AddE(`value: function(callback: (`)
				for idx, arg := range ev.Arguments {
					build.AddK(`%s: %s`, arg.ObjectName(), ts.TSTypeOf(arg.Type, mod.Path(), in))
					if idx != len(ev.Arguments)-1 {
						build.AddK(`, `)
					}
				}
				build.AddK(`) => void): number {`)
				build.AddNL()
				build.Einzug++

				build.Add(`return this.on("%s", callback)`, ev.ObjectName())

				build.AddD(`}`)
				build.AddD(`}`)
			}

			build.AddD(`}`)
			build.AddD(`)`)
		}
		build.AddD(`}`)
	}
	for _, protocol := range mod.Protocols {
		build.AddI(`export interface %s<T> {`, protocol.ObjectName())

		for _, fn := range protocol.Funcs {
			build.AddE(`%s(`, fn.ObjectName())
			for _, arg := range fn.Arguments {
				build.AddK(`%s: %s`, arg.ObjectName(), ts.TSTypeOf(arg.Type, mod.Path(), in))
				build.AddK(`, `)
			}
			build.AddK(`extra: T`)
			build.AddK(`)`)
			if fn.Returns != nil {
				build.AddK(`: Promise<%s>`, ts.TSTypeOf(fn.Returns, mod.Path(), in))
			} else {
				build.AddK(`: Promise<void>`)
			}
			build.AddNL()
		}
		build.AddD(`}`)

		build.AddI(`export function make%sFromTransport<T>(transport: Transport<T>): %s<T> {`, protocol.ObjectName(), protocol.ObjectName())
		build.AddI(`return {`)
		for _, fn := range protocol.Funcs {
			build.AddE(`async %s(`, fn.ObjectName())
			for _, arg := range fn.Arguments {
				build.AddK(`%s: %s`, arg.ObjectName(), ts.TSTypeOf(arg.Type, mod.Path(), in))
				build.AddK(`, `)
			}
			build.AddK(`extra: T`)
			build.AddK(`)`)

			if fn.Returns != nil {
				build.AddK(`: Promise<%s>`, ts.TSTypeOf(fn.Returns, mod.Path(), in))
			} else {
				build.AddK(`: Promise<void>`)
			}

			build.AddK(` {`)

			build.AddNL()
			build.Einzug++

			build.AddI(`return await transport.makeRequest(`)
			build.Add(`"%s",`, fn.Path())
			build.AddI(`{`)
			for _, arg := range fn.Arguments {
				build.Add(`%s: %s,`, arg.ObjectName(), arg.ObjectName())
			}
			build.AddD(`},`)
			build.Add(`extra,`)
			build.AddD(`)`)

			build.AddD(`},`)
		}

		build.AddD(`}`)
		build.AddD(`}`)
	}

	return build.String(), nil
}
