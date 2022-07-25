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
	"strings"

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

func contains[T comparable](a []T, b T) bool {
	for _, item := range a {
		if item == b {
			return true
		}
	}
	return false
}

func (ts TypescriptBackend) GenerateCommand() *cli.Command {
	possible := []string{"server", "client"}
	return &cli.Command{
		Name:    "typescript",
		Aliases: []string{"ts"},
		Usage:   "Generate TypeScript modules for Lugma",
		Flags: append(backends.StandardFlags, []cli.Flag{
			&cli.StringSliceFlag{
				Name:  "types",
				Usage: "The types of code to generate",
				Value: cli.NewStringSlice(possible...),
			},
		}...),
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

				result, err := ts.GenerateTypes(mod, w.Context)
				if err != nil {
					return err
				}

				err = ioutil.WriteFile(path.Join(outdir, mod.Name+".types.ts"), []byte(result), fs.ModePerm)
				if err != nil {
					return err
				}
			}

			types := cCtx.StringSlice("types")
			if contains(types, "client") {
				for _, prod := range w.Module.Products {
					mod := w.KnownModules[prod.Name]

					result, err := ts.GenerateClient(mod, w.Context)
					if err != nil {
						return err
					}

					err = ioutil.WriteFile(path.Join(outdir, mod.Name+".client.ts"), []byte(result), fs.ModePerm)
					if err != nil {
						return err
					}
				}
			}
			if contains(types, "server") {
				for _, prod := range w.Module.Products {
					mod := w.KnownModules[prod.Name]

					result, err := ts.GenerateServer(mod, w.Context)
					if err != nil {
						return err
					}

					err = ioutil.WriteFile(path.Join(outdir, mod.Name+".server.ts"), []byte(result), fs.ModePerm)
					if err != nil {
						return err
					}
				}
			}

			return nil
		},
	}
}

func (ts TypescriptBackend) GenerateTypes(mod *typechecking.Module, in *typechecking.Context) (string, error) {
	build := backends.Filebuilder{}

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

	return build.String(), nil
}

func (ts TypescriptBackend) GenerateServer(mod *typechecking.Module, in *typechecking.Context) (string, error) {
	build := backends.Filebuilder{}

	build.Add(`import { Result, Transport, Stream } from 'lugma-server-helpers'`)
	build.Add(`import { %s } from './%s.types'`, strings.Join(allNames(mod), ", "), mod.Name)
	build.Add(`export * from './%s.types'`, mod.Name)

	for _, stream := range mod.Streams {
		build.AddI(`export interface %s<T> extends Stream<T> {`, stream.ObjectName())

		for _, ev := range stream.Signals {
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

		build.AddI(`export function wrap%sFromStream<T>(stream: Stream<T>): %s<T> {`, stream.ObjectName(), stream.ObjectName())
		{
			build.AddI(`return Object.create(`)
			build.Add(`stream,`)
			build.AddI(`{`)

			for _, ev := range stream.Signals {
				build.AddI(`on%s: {`, strcase.ToCamel(ev.ObjectName()))
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

		build.AddI(`export function bind%sToTransport<T>(transport: Transport<T>, slot: (stream: %s<T>) => void) {`, stream.ObjectName(), stream.ObjectName())
		build.Add(`transport.bindStream('%s', (stream: Stream<T>) => slot(wrap%sFromStream(stream)))`, stream.Path().String(), stream.ObjectName())
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

			ret := "void"
			if fn.Returns != nil {
				ret = ts.TSTypeOf(fn.Returns, mod.Path(), in)
			}
			fai := "void"
			if fn.Throws != nil {
				ret = ts.TSTypeOf(fn.Throws, mod.Path(), in)
			}

			build.AddK(`: Promise<Result<%s, %s>>`, ret, fai)

			build.AddNL()
		}
		build.AddD(`}`)

		build.AddI(`export function bind%sToTransport<T>(impl: %s<T>, transport: Transport<T>) {`, protocol.ObjectName(), protocol.ObjectName())
		for _, fn := range protocol.Funcs {
			build.AddE(`transport.bindMethod("%s", (content: any, extra: T | undefined) => impl.%s(`, fn.Path().String(), fn.ObjectName())
			for _, arg := range fn.Arguments {
				build.AddK(`content['%s'], `, arg.ObjectName())
			}
			build.AddK(`extra))`)
			build.AddNL()
		}
		// build.AddI(`return {`)
		// for _, fn := range protocol.Funcs {
		// 	build.AddE(`async %s(`, fn.ObjectName())
		// 	for _, arg := range fn.Arguments {
		// 		build.AddK(`%s: %s`, arg.ObjectName(), ts.TSTypeOf(arg.Type, mod.Path(), in))
		// 		build.AddK(`, `)
		// 	}
		// 	build.AddK(`extra: T`)
		// 	build.AddK(`)`)

		// 	if fn.Returns != nil {
		// 		build.AddK(`: Promise<%s>`, ts.TSTypeOf(fn.Returns, mod.Path(), in))
		// 	} else {
		// 		build.AddK(`: Promise<void>`)
		// 	}

		// 	build.AddK(` {`)

		// 	build.AddNL()
		// 	build.Einzug++

		// 	build.AddI(`return await transport.makeRequest(`)
		// 	build.Add(`"%s",`, fn.Path())
		// 	build.AddI(`{`)
		// 	for _, arg := range fn.Arguments {
		// 		build.Add(`%s: %s,`, arg.ObjectName(), arg.ObjectName())
		// 	}
		// 	build.AddD(`},`)
		// 	build.Add(`extra,`)
		// 	build.AddD(`)`)

		// 	build.AddD(`},`)
		// }

		// build.AddD(`}`)
		build.AddD(`}`)
	}

	return build.String(), nil
}

func allNames(mod *typechecking.Module) []string {
	var names []string
	for _, item := range mod.Structs {
		names = append(names, item.ObjectName())
	}
	for _, item := range mod.Enums {
		names = append(names, item.ObjectName())
	}
	for _, item := range mod.Flagsets {
		names = append(names, item.ObjectName())
	}
	return names
}

func (ts TypescriptBackend) GenerateClient(mod *typechecking.Module, in *typechecking.Context) (string, error) {
	build := backends.Filebuilder{}

	build.Add(`import { Transport, Stream } from 'lugma-web-helpers'`)
	build.Add(`import { %s } from './%s.types'`, strings.Join(allNames(mod), ", "), mod.Name)
	build.Add(`export * from './%s.types'`, mod.Name)

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
