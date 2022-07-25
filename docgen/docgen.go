package docgen

import (
	"bytes"
	"fmt"
	"html/template"
	"lugmac/ast"
	"lugmac/ast/extension"
	"lugmac/backends"
	"lugmac/modules"
	"lugmac/typechecking"
	"os"
	"path"
	"path/filepath"
	"strings"

	gmast "github.com/yuin/goldmark/ast"

	"github.com/urfave/cli/v2"
	"github.com/yuin/goldmark"
)

type resolver struct {
	basePath    string
	currentlyIn typechecking.Object
}

func findChild(in typechecking.Object, strs []string) typechecking.Object {
	if len(strs) == 0 {
		return in
	}

	if child := in.Child(strs[0]); child != nil {
		return findChild(child, strs[1:])
	}
	if obj, found := in.Env().Search(strs[0]); found {
		return findChild(obj, strs[1:])
	}

	return nil
}

func findUp(o typechecking.Object, s []string) typechecking.Object {
	if child := findChild(o, s); child != nil {
		return child
	}
	if o.Parent() != nil {
		return findUp(o.Parent(), s)
	}
	return nil
}

func resolveURL(of typechecking.Object, relativeToBase string) string {
	if strings.HasPrefix(of.Path().String(), relativeToBase+"/") {
		return "./" + strings.TrimPrefix(of.Path().String(), relativeToBase+"/") + "/"
	} else {
		rel, err := filepath.Rel(relativeToBase, of.Path().String()+"/")
		if err != nil {
			return of.Path().String() + "/"
		} else {
			return rel + "/"
		}
	}
}

// ResolveSymbol implements extension.SymbolResolver
func (r *resolver) ResolveSymbol(sym *extension.SymbolLinkNode) (destination []byte, err error) {
	obj := findUp(r.currentlyIn, strings.Split(string(sym.Symbol), "."))

	if obj != nil {
		return []byte(resolveURL(obj, r.basePath)), nil
	}

	return []byte(""), nil
}

var _ extension.SymbolResolver = &resolver{}

// only a flat first-level view, not a tree
func renderOnPageStructureTo(sb *strings.Builder, nodes []gmast.Node, structure Item) {
	sb.WriteString(`<h2>Topics</h2>`)

	if len(nodes) > 0 {
		var res = &resolver{}
		var gm = goldmark.New(
			goldmark.WithExtensions(
				&extension.SymbolLinkExtender{Resolver: res},
			),
			goldmark.WithParser(
				goldmark.DefaultParser(),
			),
		)
		docs := DocumentationItemFor(structure.Object)

		for _, node := range nodes {
			if v, ok := node.(*StructuralList); ok {
				for _, item := range v.SymbolLinks {
					obj := structure.Object
					child := obj.Child(string(item.Symbol))
					if child != nil {
						url := resolveURL(child, structure.Object.Path().String())
						sig := HTMLSignatureFor(child, structure.Object)
						sb.WriteString(fmt.Sprintf(`<h4><a class="code" href="%s">%s</a></h4>`, url, sig))
						sb.WriteString(fmt.Sprintf(`<p class="pl-6">%s</p>`, SummaryFor(child)))
					} else {
						sb.WriteString(`bad link`)
					}
				}
			} else {
				gm.Renderer().Render(sb, docs.Source, node)
			}
		}
	} else {
		for _, section := range structure.Children {
			sb.WriteString(fmt.Sprintf(`<h3>%s</h3>`, section.Title))
			for _, item := range section.Items {
				url := resolveURL(item.Object, structure.Object.Path().String())
				sig := HTMLSignatureFor(item.Object, structure.Object)
				sb.WriteString(fmt.Sprintf(`<h4><a class="code" href="%s">%s</a></h4>`, url, sig))
				sb.WriteString(fmt.Sprintf(`<p class="pl-6">%s</p>`, SummaryFor(item.Object)))
			}
		}
	}
}

func breadcrumbsFor(item typechecking.Object) string {
	var s []string
	var root typechecking.Object

	for i := item; i != nil; i = i.Parent() {
		s = append(s, fmt.Sprintf(`<a href="%s">%s</a>`, resolveURL(i, item.Path().String()), i.ObjectName()))
		root = i
	}

	for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
		s[i], s[j] = s[j], s[i]
	}

	if strings.HasSuffix(root.Path().String(), "/"+root.ObjectName()) {
		parents := strings.Split(strings.TrimSuffix(root.Path().String(), "/"+root.ObjectName()), "/")
		for i := range parents {
			parents[i] = fmt.Sprintf(`<a>%s</a>`, parents[i])
		}
		s = append(parents, s...)
	}

	return strings.Join(s, ` <span class="px-2">/</span> `)
}

func renderObject(outdir string, workspace *typechecking.Workspace, item typechecking.Object, docs *ast.ItemDocumentation) error {
	var res = &resolver{}
	var gm = goldmark.New(
		goldmark.WithExtensions(
			&extension.SymbolLinkExtender{Resolver: res},
		),
		goldmark.WithParser(
			goldmark.DefaultParser(),
		),
	)

	args := TemplateArguments{}

	var tableOfContents strings.Builder

	DefaultStructureForWorkspace(workspace).Children[0].renderTableOfContentsTo(&tableOfContents, item)

	args.TableOfContents = template.HTML(tableOfContents.String())
	args.Breadcrumbs = template.HTML(breadcrumbsFor(item))

	if docs != nil {
		oldCur := res.currentlyIn
		defer func() { res.currentlyIn = oldCur }()

		oldBas := res.basePath
		defer func() { res.basePath = oldBas }()

		res.currentlyIn = item
		res.basePath = item.Path().String()

		var mainBuilder strings.Builder

		rend := func(t gmast.Node) error {
			return gm.Renderer().Render(&mainBuilder, docs.Source, t)
		}

		mainBuilder.WriteString(fmt.Sprintf("<h1>%s</h1>", item.ObjectName()))

		err := rend(docs.Summary)
		if err != nil {
			return err
		}

		mainBuilder.WriteString(fmt.Sprintf(`<pre><code>%s</code></pre>`, HTMLSignatureFor(item, item)))

		mainBuilder.WriteString(`<hr>`)

		if len(docs.Discussion) > 0 {
			mainBuilder.WriteString(fmt.Sprintf("<h2>Discussion</h2>"))
		}

		for _, disc := range docs.Discussion {
			err = rend(disc)
			if err != nil {
				return err
			}
		}

		// structural objects
		if IsStructuralObject(item) {
			nodes, item := StructureFor(item)
			renderOnPageStructureTo(&mainBuilder, nodes, item)
		}

		// function-likes
		if len(docs.Parameters) > 0 {
			mainBuilder.WriteString(fmt.Sprintf("<h2>Parameters</h2>"))
			mainBuilder.WriteString(fmt.Sprintf("<dl>"))
			for parm, item := range docs.Parameters {
				mainBuilder.WriteString(fmt.Sprintf("<dt>%s</dt>", parm))
				mainBuilder.WriteString("<dd>")
				err = rend(item)
				if err != nil {
					return err
				}
				mainBuilder.WriteString("</dd>")
			}
			mainBuilder.WriteString(fmt.Sprintf("</dl>"))
		}
		if docs.Returns != nil {
			mainBuilder.WriteString(fmt.Sprintf("<h2>Return Value</h2>"))

			err = rend(docs.Returns)
			if err != nil {
				return err
			}
		}
		if docs.Throws != nil {
			mainBuilder.WriteString(fmt.Sprintf("<h2>Throws</h2>"))

			err = rend(docs.Throws)
			if err != nil {
				return err
			}
		}

		args.Main = template.HTML(mainBuilder.String())
	} else {
		var mainBuilder strings.Builder

		mainBuilder.WriteString(fmt.Sprintf("<h1>%s</h1>", item.ObjectName()))

		mainBuilder.WriteString(fmt.Sprintf(`<pre><code>%s</code></pre>`, HTMLSignatureFor(item, item)))

		// structural objects
		if IsStructuralObject(item) {
			nodes, item := StructureFor(item)
			renderOnPageStructureTo(&mainBuilder, nodes, item)
		}

		args.Main = template.HTML(mainBuilder.String())
	}

	err := os.MkdirAll(path.Join(outdir, item.Path().String()), 0750)
	if err != nil {
		return err
	}

	var out bytes.Buffer

	err = Template.Execute(&out, args)
	if err != nil {
		return err
	}

	err = os.WriteFile(path.Join(outdir, item.Path().String(), "index.html"), out.Bytes(), 0660)
	if err != nil {
		return err
	}

	return nil
}

var Command = &cli.Command{
	Name:  "document",
	Usage: "Generate documentation from Lugma IDL definitions",
	Flags: backends.StandardFlags,
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
		err = os.WriteFile(path.Join(outdir, "main.css"), []byte(css), 0660)
		if err != nil {
			return err
		}
		err = os.WriteFile(path.Join(outdir, "main.js"), []byte(js), 0660)
		if err != nil {
			return err
		}

		for _, prod := range w.Module.Products {
			mod := w.KnownModules[prod.Name]

			for _, flagset := range mod.Flagsets {
				err = renderObject(outdir, mod.InWorkspace, flagset, flagset.Documentation)
				if err != nil {
					return err
				}

				for _, flag := range flagset.Flags {
					err = renderObject(outdir, mod.InWorkspace, flag, flag.Documentation)
					if err != nil {
						return err
					}
				}
			}
			for _, enum := range mod.Enums {
				err = renderObject(outdir, mod.InWorkspace, enum, enum.Documentation)
				if err != nil {
					return err
				}

				for _, cas := range enum.Cases {
					err = renderObject(outdir, mod.InWorkspace, cas, cas.Documentation)
					if err != nil {
						return err
					}
				}
			}
			for _, strct := range mod.Structs {
				err = renderObject(outdir, mod.InWorkspace, strct, strct.Documentation)
				if err != nil {
					return err
				}

				for _, field := range strct.Fields {
					err = renderObject(outdir, mod.InWorkspace, field, field.Documentation)
					if err != nil {
						return err
					}
				}
			}
			for _, protocol := range mod.Protocols {
				err = renderObject(outdir, mod.InWorkspace, protocol, protocol.Documentation)
				if err != nil {
					return err
				}

				for _, fn := range protocol.Funcs {
					err = renderObject(outdir, mod.InWorkspace, fn, fn.Documentation)
					if err != nil {
						return err
					}
				}
			}
			for _, stream := range mod.Streams {
				err = renderObject(outdir, mod.InWorkspace, stream, stream.Documentation)
				for _, signal := range stream.Signals {
					err = renderObject(outdir, mod.InWorkspace, signal, signal.Documentation)
					if err != nil {
						return err
					}
				}
				for _, event := range stream.Events {
					err = renderObject(outdir, mod.InWorkspace, event, event.Documentation)
					if err != nil {
						return err
					}
				}
			}
		}

		return nil
	},
}
