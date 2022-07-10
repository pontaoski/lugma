package main

import (
	"lugmac/backends/typescript"
	"lugmac/typechecking"
	"os"
)

func main() {
	ctx := typechecking.NewContext()
	err := ctx.MakeModule(os.Args[1])
	if err != nil {
		panic(err)
	}

	j := typescript.TypescriptBackend{}
	err = j.Generate(os.Args[1], ctx)
	if err != nil {
		panic(err)
	}
}
