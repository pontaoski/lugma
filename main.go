package main

import (
	"log"
	"lugmac/backends"
	"lugmac/docgen"
	"lugmac/modules"
	"os"

	"github.com/urfave/cli/v2"

	_ "lugmac/backends/jsonschema"
	_ "lugmac/backends/typescript"
)

func main() {
	gen := &cli.Command{
		Name:  "generate",
		Usage: "Generate code from Lugma IDL definitions",
	}
	for _, backend := range backends.Backends {
		gen.Subcommands = append(gen.Subcommands, backend.GenerateCommand())
	}
	app := &cli.App{
		Usage: "The command for everything Lugma",
		Commands: []*cli.Command{
			gen,
			docgen.Command,
			{
				Name:  "verify",
				Usage: "Verify a Lugma file",
				Action: func(cCtx *cli.Context) error {
					mod, err := modules.LoadWorkspaceFrom(".")
					if err != nil {
						return err
					}

					err = mod.GenerateModules()
					if err != nil {
						return err
					}

					return nil
				},
			},
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatalf("%s", err)
	}
}
