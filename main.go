package main

import (
	"io/ioutil"
	"lugmac/ast"
	"os"

	"github.com/alecthomas/repr"
	sitter "github.com/smacker/go-tree-sitter"
)

func main() {
	parser := sitter.NewParser()
	parser.SetLanguage(Lang)

	file, err := ioutil.ReadFile(os.Args[1])
	if err != nil {
		panic(err)
	}

	tree := parser.Parse(nil, file)

	fileAST := ast.FileFromNode(tree.RootNode(), file)
	repr.Println(fileAST)
}
