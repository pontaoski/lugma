package extension

import (
	"bytes"
	"fmt"
	"sync"
	"unicode"

	"github.com/rivo/uniseg"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

type SymbolLinkExtender struct {
	Resolver SymbolResolver
}

// Extend implements goldmark.Extender
func (self *SymbolLinkExtender) Extend(m goldmark.Markdown) {
	m.Parser().AddOptions(parser.WithInlineParsers(util.Prioritized(&SymbolLinkParser{}, 999)))
	m.Renderer().AddOptions(renderer.WithNodeRenderers(util.Prioritized(&SymbolLinkRenderer{
		Resolver: self.Resolver,
	}, 999)))
}

var _ goldmark.Extender = &SymbolLinkExtender{}

type SymbolLinkParser struct {
}

func span(tag []byte) int {
	if idx := bytes.IndexFunc(tag, unicode.IsSpace); idx >= 0 {
		tag = tag[:idx]
	}
	end := len(tag)

	gr := uniseg.NewGraphemes(string(tag))
	for gr.Next() {
		if bytes.IndexFunc(gr.Bytes(), unicode.IsSpace) < 0 {
			end, _ = gr.Positions()
		}
	}

	return end
}

// Parse implements parser.InlineParser
func (self *SymbolLinkParser) Parse(parent ast.Node, block text.Reader, pc parser.Context) ast.Node {
	line, seg := block.PeekLine()

	if len(line) == 0 || line[0] != '@' {
		return nil
	}

	end := span(line)
	if end < 0 {
		return nil
	}

	seg = seg.WithStop(seg.Start + end + 1)

	n := SymbolLinkNode{
		Symbol: block.Value(seg.WithStart(seg.Start + 1)),
	}
	n.AppendChild(&n, ast.NewTextSegment(seg.WithStart(seg.Start+1)))
	block.Advance(seg.Len())
	return &n
}

// Trigger implements parser.InlineParser
func (self *SymbolLinkParser) Trigger() []byte {
	return []byte("@")
}

var _ parser.InlineParser = &SymbolLinkParser{}

var _ ast.Node = &SymbolLinkNode{}

type SymbolLinkNode struct {
	ast.BaseInline

	Symbol []byte
}

// Dump implements ast.Node
func (n *SymbolLinkNode) Dump(source []byte, level int) {
	ast.DumpHelper(n, source, level, map[string]string{
		"Symbol": string(n.Symbol),
	}, nil)
}

var SymbolLinkKind = ast.NewNodeKind("SymbolLink")

// Kind implements ast.Node
func (*SymbolLinkNode) Kind() ast.NodeKind {
	return SymbolLinkKind
}

type SymbolResolver interface {
	ResolveSymbol(*SymbolLinkNode) (destination []byte, err error)
}

type SymbolLinkRenderer struct {
	Resolver SymbolResolver

	once    sync.Once
	hasDest map[*SymbolLinkNode]struct{}
}

// RegisterFuncs registers rendering functions from this renderer onto the
// provided registerer.
func (r *SymbolLinkRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(SymbolLinkKind, r.Render)
}

func (r *SymbolLinkRenderer) init() {
	r.once.Do(func() {
		r.hasDest = make(map[*SymbolLinkNode]struct{})
	})
}

func (r *SymbolLinkRenderer) Render(w util.BufWriter, src []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	r.init()

	n, ok := node.(*SymbolLinkNode)
	if !ok {
		return ast.WalkStop, fmt.Errorf("unexpected node %T, expected *SymbolLinkNode", node)
	}

	if entering {
		if err := r.enter(w, n); err != nil {
			return ast.WalkStop, err
		}
	} else {
		r.exit(w, n)
	}

	return ast.WalkContinue, nil
}

func (r *SymbolLinkRenderer) enter(w util.BufWriter, n *SymbolLinkNode) error {

	var dest []byte
	if res := r.Resolver; res != nil {
		var err error
		dest, err = res.ResolveSymbol(n)
		if err != nil {
			return fmt.Errorf("resolve hashtag %q: %w", n.Symbol, err)
		}
	}

	if len(dest) == 0 {
		w.WriteString(`<span class="bad-symbol">`)
		return nil
	}

	r.hasDest[n] = struct{}{}
	w.WriteString(`<a class="symbol-link" href="`)
	w.Write(util.URLEscape(dest, true /* resolve references */))
	w.WriteString(`">`)
	return nil
}

func (r *SymbolLinkRenderer) exit(w util.BufWriter, n *SymbolLinkNode) {
	if _, ok := r.hasDest[n]; ok {
		delete(r.hasDest, n)
		w.WriteString("</a>")
	} else {
		w.WriteString("</span>")
	}
}
