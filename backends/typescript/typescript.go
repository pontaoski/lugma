package typescript

import (
	"fmt"
	"lugmac/backends"
	"lugmac/typechecking"
)

type TypescriptBackend struct {
}

var _ backends.Backend = TypescriptBackend{}

func (ts TypescriptBackend) TSTypeOf(lugma typechecking.Type, module string, in *typechecking.Context) string {
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
	case typechecking.Struct, typechecking.Enum:
		if k.Path().ModulePath == module {
			return k.String()
		}
		return fmt.Sprintf("TODO")
	default:
		panic("unhandled " + k.String())
	}
}

func (ts TypescriptBackend) Generate(module string, in *typechecking.Context) error {
	build := backends.Filebuilder{}
	mod := in.KnownModules[module]

	build.Add(`import { Transport, Stream } from 'lugma-web-helpers'`)

	for _, item := range mod.Structs {
		build.AddI("export interface %s {", item.Name)
		for _, field := range item.Fields {
			build.Add(`%s: %s`, field.Name, ts.TSTypeOf(field.Type, module, in))
		}
		build.AddD("}")
	}
	for _, item := range mod.Enums {
		build.AddI("export type %s =", item.Name)
		simple := item.Simple()
		for idx, esac := range item.Cases {
			if simple {
				build.Add(`"%s" |`, esac.Name)
			} else {
				build.AddE(`{ %s: {`, esac.Name)
				for _, field := range esac.Fields {
					build.AddK(`%s: %s;`, field.Name, ts.TSTypeOf(field.Type, module, in))
				}
				build.AddK(`} }`)
				if idx != len(item.Cases)-1 {
					build.AddK(` |`)
				}
				build.AddNL()
				// build.Add(`"%s" |`, esac.Name)
			}
		}
		build.Einzug--
	}
	for _, item := range mod.Flagsets {
		build.Add(`export type %s = string`, item.Name)
	}
	for _, protocol := range mod.Protocols {
		build.AddI(`export interface %sRequests<T> {`, protocol.Name)
		if len(protocol.Events) > 0 {
			build.Add(`SubscribeToEvents(extra: T | undefined): Promise<%sStream>`, protocol.Name)
		}
		for _, fn := range protocol.Funcs {
			build.AddE(`%s(`, fn.Name)
			for _, arg := range fn.Arguments {
				build.AddK(`%s: %s`, arg.Name, ts.TSTypeOf(arg.Type, module, in))
				build.AddK(`, `)
			}
			build.AddK(`extra: T`)
			build.AddK(`)`)
			if fn.Returns != nil {
				build.AddK(`: %s`, ts.TSTypeOf(fn.Returns, module, in))
			} else {
				build.AddK(`: Promise<void>`)
			}
			build.AddNL()
		}
		build.AddD(`}`)

		if len(protocol.Events) > 0 {
			build.AddI(`export interface %sStream extends Stream {`, protocol.Name)
			for _, ev := range protocol.Events {
				build.AddE(`on%s(callback: (`, ev.Name)
				for idx, arg := range ev.Arguments {
					build.AddK(`%s: %s`, arg.Name, ts.TSTypeOf(arg.Type, module, in))
					if idx != len(ev.Arguments)-1 {
						build.AddK(`, `)
					}
				}
				build.AddK(`) => void): number`)
				build.AddNL()
			}
			build.AddD(`}`)
		}

		build.AddI(`export function make%sFromTransport<T>(transport: Transport<T>): ChatRequests<T> {`, protocol.Name)
		build.AddI(`return {`)
		for _, fn := range protocol.Funcs {
			build.AddE(`async %s(`, fn.Name)
			for _, arg := range fn.Arguments {
				build.AddK(`%s: %s`, arg.Name, ts.TSTypeOf(arg.Type, module, in))
				build.AddK(`, `)
			}
			build.AddK(`extra: T`)
			build.AddK(`)`)

			if fn.Returns != nil {
				build.AddK(`: %s`, ts.TSTypeOf(fn.Returns, module, in))
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
				build.Add(`%s: %s,`, arg.Name, arg.Name)
			}
			build.AddD(`},`)
			build.Add(`extra,`)
			build.AddD(`)`)

			build.AddD(`},`)
		}

		if len(protocol.Events) > 0 {
			build.AddI(`async SubscribeToEvents(extra: T | undefined): Promise<%sStream> {`, protocol.Name)
			build.AddI(`return Object.create(`)
			build.Add(`transport.openStream("%s", extra),`, protocol.Path().String())
			build.AddI(`{`)

			for _, ev := range protocol.Events {
				build.AddI(`on%s: {`, ev.Name)
				build.AddE(`value: function(callback: (`)
				for idx, arg := range ev.Arguments {
					build.AddK(`%s: %s`, arg.Name, ts.TSTypeOf(arg.Type, module, in))
					if idx != len(ev.Arguments)-1 {
						build.AddK(`, `)
					}
				}
				build.AddK(`) => void): number {`)
				build.AddNL()
				build.Einzug++

				build.Add(`return this.on("%s", callback)`, ev.Name)

				build.AddD(`}`)
				build.AddD(`}`)
			}

			build.AddD(`}`)
			build.AddD(`)`)
			build.AddD(`}`)
		}

		build.AddD(`}`)
		build.AddD(`}`)
	}

	print(build.String())

	return nil
}
