package ast

import (
	"lugmac/ast/extension"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
)

var gm = goldmark.New(
	goldmark.WithExtensions(
		&extension.SymbolLinkExtender{},
	),
	goldmark.WithParser(
		goldmark.DefaultParser(),
	),
)

type ItemDocumentation struct {
	Summary    ast.Node
	Discussion []ast.Node
	Parameters map[string]ast.Node
	Returns    ast.Node
	Throws     ast.Node

	CustomStructure    []ast.Node
	HasCustomStructure bool

	Source []byte
}

func transmute(from ast.Node, into ast.Node) ast.Node {
	for i := from.FirstChild(); i != nil; i = i.NextSibling() {
		into.AppendChild(into, i)
	}
	return into
}

func FromDocumentationComment(comment string, isMethod bool) *ItemDocumentation {
	source := []byte(comment)

	document := gm.Parser().Parse(text.NewReader(source))

	doc := ItemDocumentation{}
	doc.Parameters = map[string]ast.Node{}
	doc.Source = source

	summaryGot := false

	for i := document.FirstChild(); i != nil; i = i.NextSibling() {
		if !summaryGot && i.Kind() == ast.KindParagraph {
			doc.Summary = i
			summaryGot = true
		} else if i.Kind() == ast.KindList && i == document.LastChild() && isMethod {
			for ii := i.FirstChild(); ii != nil; ii = ii.NextSibling() {
				if ii.FirstChild().Kind() != ast.KindParagraph {
					continue
				}
				if string(ii.FirstChild().Text(source)) == "Parameters:" {
					theList := ii.FirstChild().NextSibling()

					for iii := theList.FirstChild(); iii != nil; iii = iii.NextSibling() {
						txt := string(iii.Text(source))
						splitted := strings.SplitN(txt, ":", 2)
						if len(splitted) != 2 {
							continue
						}

						prefix := splitted[0] + ":"

						if strings.TrimSpace(string(iii.FirstChild().FirstChild().Text(source))) == prefix {
							iii.FirstChild().RemoveChild(iii.FirstChild(), iii.FirstChild().FirstChild())
						} else {
							frag := iii.FirstChild().Lines().At(0)
							frag.Start = frag.Start + len(prefix)
							iii.FirstChild().FirstChild().(*ast.Text).Segment = frag
						}

						doc.Parameters[splitted[0]] = transmute(iii, ast.NewTextBlock())
					}
				} else if strings.HasPrefix(string(ii.FirstChild().Text(source)), "Returns:") {
					if strings.TrimSpace(string(ii.FirstChild().FirstChild().Text(source))) == "Returns:" {
						ii.FirstChild().RemoveChild(ii.FirstChild(), ii.FirstChild().FirstChild())
					} else {
						frag := ii.FirstChild().Lines().At(0)
						frag.Start = frag.Start + len("Returns:")
						fragment := ii.FirstChild().FirstChild().(*ast.Text)
						fragment.Segment = frag
					}

					doc.Returns = transmute(ii, ast.NewTextBlock())
				} else if strings.HasPrefix(string(ii.FirstChild().Text(source)), "Throws:") {
					if strings.TrimSpace(string(ii.FirstChild().FirstChild().Text(source))) == "Throws:" {
						ii.FirstChild().RemoveChild(ii.FirstChild(), ii.FirstChild().FirstChild())
					} else {
						frag := ii.FirstChild().Lines().At(0)
						frag.Start = frag.Start + len("Throws:")
						fragment := ii.FirstChild().FirstChild().(*ast.Text)
						fragment.Segment = frag
					}

					doc.Throws = transmute(ii, ast.NewTextBlock())
				}
			}
		} else if h, ok := i.(*ast.Heading); ok && h.Level == 1 && string(h.Text(source)) == "Topics" {
			doc.HasCustomStructure = true
		} else if doc.HasCustomStructure {
			if i.Kind() == ast.KindHeading {
				i.(*ast.Heading).Level += 1
			}
			doc.CustomStructure = append(doc.CustomStructure, i)
		} else if i.Kind() == ast.KindHeading {
			i.(*ast.Heading).Level += 1
			doc.Discussion = append(doc.Discussion, i)
		} else {
			doc.Discussion = append(doc.Discussion, i)
		}
	}

	return &doc
}
