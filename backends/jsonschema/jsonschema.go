package jsonschema

import (
	"encoding/json"
	"io/fs"
	"io/ioutil"
	"lugmac/backends"
	"lugmac/typechecking"

	"github.com/urfave/cli/v2"
)

type JSONSchemaBackend struct {
	URLBase string
}

var _ backends.Backend = JSONSchemaBackend{}

type AnyDict map[string]interface{}

func (j JSONSchemaBackend) JSONTypeOf(lugma typechecking.Type, module string, in *typechecking.Context) (child AnyDict) {
	switch k := lugma.(type) {
	case typechecking.PrimitiveType:
		switch k {
		case typechecking.UInt8, typechecking.UInt16, typechecking.UInt32, typechecking.Int8, typechecking.Int16, typechecking.Int32:
			return AnyDict{"type": "number"}
		case typechecking.Int64, typechecking.UInt64, typechecking.String, typechecking.Bytes:
			return AnyDict{"type": "string"}
		case typechecking.Bool:
			return AnyDict{"type": "bool"}
		default:
			panic("unhandled primitive " + k.String())
		}
	case typechecking.ArrayType:
		return AnyDict{"type": "array", "items": j.JSONTypeOf(k.Element, module, in)}
	case typechecking.DictionaryType:
		return AnyDict{"type": "object", "additionalProperties": j.JSONTypeOf(k.Element, module, in)}
	case typechecking.OptionalType:
		v := j.JSONTypeOf(k.Element, module, in)
		switch a := v["type"].(type) {
		case string:
			v["type"] = []string{a, "null"}
		default:
			// nothing
		}
		return v
	case typechecking.Struct, typechecking.Enum:
		elementPath := lugma.Path()
		return AnyDict{"$ref": j.URLBase + elementPath.ModulePath + elementPath.InModulePath}
	default:
		panic("unhandled " + k.String())
	}
}

func init() {
	backends.RegisterBackend(JSONSchemaBackend{})
}

func (j JSONSchemaBackend) GenerateCommand() *cli.Command {
	return &cli.Command{
		Name:    "json-schema",
		Aliases: []string{"jsons"},
		Usage:   "Generate JSON Schema files for Lugma",
		Flags:   backends.StandardFlags,
		Action: func(cCtx *cli.Context) error {
			output := cCtx.String("output")

			ctx := typechecking.NewContext()
			err := ctx.MakeModule(cCtx.Args().First())
			if err != nil {
				return err
			}

			var result string

			result, err = j.Generate(cCtx.Args().First(), ctx)
			if err != nil {
				return err
			}

			if output == "" {
				println(result)
			} else {
				ioutil.WriteFile(output, []byte(result), fs.ModePerm)
			}

			return nil
		},
	}
}

func (j JSONSchemaBackend) Generate(module string, in *typechecking.Context) (string, error) {
	mod := in.KnownModules[module]

	schemas := map[string]AnyDict{}

	for _, strct := range mod.Structs {
		props := map[string]AnyDict{}
		required := []string{}

		for _, item := range strct.Fields {
			props[item.Name] = j.JSONTypeOf(item.Type, module, in)
			switch props[item.Name]["type"].(type) {
			case []string:
				// do nothing
			default:
				required = append(required, item.Name)
			}
		}

		loc := j.URLBase + strct.Path().ModulePath + strct.Path().InModulePath
		schemas[loc] = AnyDict{
			"$id":        strct.Name,
			"type":       "object",
			"properties": props,
			"required":   required,
		}
	}
	for _, enum := range mod.Enums {
		oneOfs := []interface{}{}

		simple := enum.Simple()
		for _, esac := range enum.Cases {
			if simple {
				oneOfs = append(oneOfs, AnyDict{"type": "string", "const": esac.Name})
			} else {
				props := map[string]AnyDict{}
				required := []string{}

				for _, item := range esac.Fields {
					props[item.Name] = j.JSONTypeOf(item.Type, module, in)
					switch props[item.Name]["type"].(type) {
					case []string:
						// do nothing
					default:
						required = append(required, item.Name)
					}
				}

				oneOfs = append(oneOfs, AnyDict{
					"type":       "object",
					"properties": props,
					"required":   required,
				})
			}
		}

		loc := j.URLBase + enum.Path().ModulePath + enum.Path().InModulePath
		schemas[loc] = AnyDict{
			"$id":   loc,
			"oneOf": oneOfs,
		}
	}
	for _, protocol := range mod.Protocols {
		for _, fn := range protocol.Funcs {
			props := map[string]AnyDict{}
			required := []string{}

			for _, item := range fn.Arguments {
				props[item.Name] = j.JSONTypeOf(item.Type, module, in)
				switch props[item.Name]["type"].(type) {
				case []string:
					// do nothing
				default:
					required = append(required, item.Name)
				}
			}

			loc := j.URLBase + fn.Path().ModulePath + fn.Path().InModulePath
			schemas[loc] = AnyDict{
				"$id":        loc,
				"type":       "object",
				"properties": props,
				"required":   required,
			}
		}
		for _, ev := range protocol.Events {
			props := map[string]AnyDict{}
			required := []string{}

			for _, item := range ev.Arguments {
				props[item.Name] = j.JSONTypeOf(item.Type, module, in)
				switch props[item.Name]["type"].(type) {
				case []string:
					// do nothing
				default:
					required = append(required, item.Name)
				}
			}

			loc := j.URLBase + ev.Path().ModulePath + ev.Path().InModulePath
			schemas[loc] = AnyDict{
				"$id":        loc,
				"type":       "object",
				"properties": props,
				"required":   required,
			}
		}
	}
	for _, flagset := range mod.Flagsets {
		loc := j.URLBase + flagset.Path().ModulePath + flagset.Path().InModulePath
		pattern := `^[TF]*$`
		if flagset.Optional {
			pattern = `^[TF_]*$`
		}
		schemas[loc] = AnyDict{
			"$id":     loc,
			"type":    "string",
			"pattern": pattern,
		}
	}

	data, err := json.MarshalIndent(AnyDict{
		"$id":     j.URLBase + module,
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"$defs":   schemas,
	}, "", "\t")
	if err != nil {
		return "", err
	}

	return string(data), nil
}
