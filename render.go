package sidenote

import (
	"strconv"

	gast "github.com/yuin/goldmark/ast"
	extast "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/util"
)

// Class names of the Tufte CSS sidenote markup, as emitted by pandoc-sidenote.
// The label carries no text of its own: the number in front of a sidenote is
// drawn by the stylesheet from a CSS counter.
const (
	wrapperClass = "sidenote-wrapper"
	numberClass  = "margin-toggle sidenote-number"
	toggleClass  = "margin-toggle"
	noteClass    = "sidenote"
)

type nodeRenderer struct {
	html.Config

	config Config

	// renderer is the renderer this node renderer was registered with, used to
	// render the contents of a footnote again as a sidenote.
	renderer renderer.Renderer
}

// RegisterFuncs implements renderer.NodeRenderer.
func (r *nodeRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	// A sidenote repeats contents that WithFootnotes also renders in the
	// footnote list. An AST node has a single parent and goldmark cannot copy a
	// subtree, so those nodes are rendered twice instead, which needs a handle
	// on the renderer driving the walk. goldmark passes itself as the
	// registerer, so it is the same renderer that will render this document.
	if rd, ok := reg.(renderer.Renderer); ok {
		r.renderer = rd
	}
	reg.Register(KindSidenote, r.renderSidenote)
}

// SetOption implements renderer.SetOptioner. goldmark offers every renderer
// option to each node renderer; html.Config ignores the ones it does not know,
// and picks up html.WithXHTML, which decides how the tags here are closed.
func (r *nodeRenderer) SetOption(name renderer.OptionName, value any) {
	r.Config.SetOption(name, value)
}

func (r *nodeRenderer) renderSidenote(
	w util.BufWriter, source []byte, node gast.Node, entering bool) (gast.WalkStatus, error) {
	if !entering {
		return gast.WalkContinue, nil
	}
	n := node.(*Sidenote)

	_, _ = w.WriteString(`<span class="` + wrapperClass + `"><label for="`)
	r.writeID(w, n)
	_, _ = w.WriteString(`" class="` + numberClass + `"></label><input type="checkbox" id="`)
	r.writeID(w, n)
	_, _ = w.WriteString(`" class="` + toggleClass)
	if r.Config.XHTML {
		_, _ = w.WriteString(`"/>`)
	} else {
		_, _ = w.WriteString(`">`)
	}
	_, _ = w.WriteString(`<span class="` + noteClass + `">`)
	if err := r.renderContents(w, source, n.Footnote); err != nil {
		return gast.WalkStop, err
	}
	_, _ = w.WriteString(`</span></span>`)
	return gast.WalkContinue, nil
}

func (r *nodeRenderer) writeID(w util.BufWriter, n *Sidenote) {
	_, _ = w.Write(r.idPrefix(n))
	_, _ = w.WriteString("sn-")
	_, _ = w.WriteString(strconv.Itoa(n.Ordinal))
}

func (r *nodeRenderer) idPrefix(node gast.Node) []byte {
	if r.config.IDPrefix != nil {
		return r.config.IDPrefix
	}
	if r.config.IDPrefixFunction != nil {
		return r.config.IDPrefixFunction(node)
	}
	return nil
}

// renderContents writes the contents of a footnote as inline content.
//
// A sidenote sits inside the paragraph that refers to it, so block elements
// cannot be nested in it: paragraphs are flattened and separated by line
// breaks, as pandoc-sidenote does.
//
// The nodes are written through the renderer that is driving the current walk
// rather than by walking them here, because under WithFootnotes the very same
// nodes are also rendered in the footnote list, and an AST node has only one
// parent. Backlinks are skipped: they point at a reference that a sidenote,
// being where the reference was, does not need.
func (r *nodeRenderer) renderContents(w util.BufWriter, source []byte, footnote *extast.Footnote) error {
	if footnote == nil || r.renderer == nil {
		return nil
	}
	breaks := "<br>\n<br>\n"
	if r.Config.XHTML {
		breaks = "<br />\n<br />\n"
	}
	for block := footnote.FirstChild(); block != nil; block = block.NextSibling() {
		if block.Kind() == gast.KindParagraph || block.Kind() == gast.KindTextBlock {
			for inline := block.FirstChild(); inline != nil; inline = inline.NextSibling() {
				if inline.Kind() == extast.KindFootnoteBacklink {
					continue
				}
				if err := r.renderer.Render(w, source, inline); err != nil {
					return err
				}
			}
		} else if err := r.renderer.Render(w, source, block); err != nil {
			return err
		}
		_, _ = w.WriteString(breaks)
	}
	return nil
}
