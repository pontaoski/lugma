package modules

import (
	"fmt"
	"io/fs"
	"io/ioutil"
	"lugmac/ast"
	"lugmac/typechecking"
	"os"
	"path"
	"path/filepath"

	lugma "lugmac/parser"

	sitter "github.com/smacker/go-tree-sitter"
	"gopkg.in/yaml.v3"
)

type Workspace struct {
	Dir          string
	Module       *ModuleDefinition
	KnownModules map[string]*typechecking.Module
}

type ModuleDefinition struct {
	Name     string              `yaml:"name"`
	Version  string              `yaml:"version"`
	Products []ProductDefinition `yaml:"products"`
}

type ProductDefinition struct {
	Type    string   `yaml:"type"`
	Name    string   `yaml:"name"`
	Depends []string `yaml:"depends"`
}

func LoadModuleDefinitionFrom(dir string) (*ModuleDefinition, error) {
	file := path.Join(dir, "lugma.yaml")
	data, err := os.ReadFile(file)
	if err != nil {
		return nil, fmt.Errorf("failed to load lugma.yaml: %w", err)
	}

	m := ModuleDefinition{}

	err = yaml.Unmarshal(data, &m)
	if err != nil {
		return nil, fmt.Errorf("failed to parse lugma.yaml: %w", err)
	}

	return &m, nil
}

func LoadWorkspaceFrom(dir string) (*Workspace, error) {
	mod, err := LoadModuleDefinitionFrom(dir)
	if err != nil {
		return nil, err
	}

	return &Workspace{dir, mod, map[string]*typechecking.Module{}}, nil
}

func (m *Workspace) ModuleFor(context *typechecking.Context, path string, from string) (*typechecking.Module, error) {
	v, ok := m.KnownModules[path]
	if !ok {
		return nil, fmt.Errorf("idk where %s is", path)
	}

	return v, nil
}

func (m *Workspace) GenerateModules() error {
	ctx := typechecking.NewContext(m)

	parser := sitter.NewParser()
	parser.SetLanguage(lugma.GetLanguage())

	for _, product := range m.Module.Products {
		var files []string
		filepath.WalkDir(path.Join(m.Dir, "Sources", product.Name), func(path string, d fs.DirEntry, err error) error {
			if filepath.Ext(path) == ".lugma" && !d.IsDir() {
				files = append(files, path)
			}
			return nil
		})

		var astFiles []*ast.File

		for _, path := range files {
			file, err := ioutil.ReadFile(path)
			if err != nil {
				return fmt.Errorf("failed to load module at %s: %w", path, err)
			}

			tree := parser.Parse(nil, file)

			fileAST := ast.FileFromNode(tree.RootNode(), file)
			astFiles = append(astFiles, &fileAST)
		}

		module, err := ctx.MultiFileModule(astFiles, m.Module.Name+"/"+product.Name)
		if err != nil {
			return err
		}

		m.KnownModules[product.Name] = module
	}

	return nil
}
