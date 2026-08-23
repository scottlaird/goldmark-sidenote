// Package sidenote renders the footnotes of a Markdown document as sidenotes:
// notes set beside the text that refers to them, in the margin, in the manner
// of Tufte CSS.
//
// The markup matches [pandoc-sidenote], so a stylesheet written for that will
// work unchanged.
//
// This extension renders the footnotes found by goldmark's own footnote
// extension, which has to be enabled alongside it:
//
//	goldmark.New(
//		goldmark.WithExtensions(extension.Footnote, sidenote.New()),
//	)
//
// Without extension.Footnote there are no footnotes to render and this
// extension does nothing.
//
// [pandoc-sidenote]: https://github.com/jez/pandoc-sidenote
package sidenote

import (
	"fmt"

	"github.com/yuin/goldmark"
	gast "github.com/yuin/goldmark/ast"
	extast "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/util"
)

// A Sidenote node represents a footnote rendered where it is referenced.
//
// A Sidenote does not own the contents it displays. They belong to the
// footnote it points at, which under WithFootnotes is rendered a second time
// in the footnote list at the end of the document.
type Sidenote struct {
	gast.BaseInline

	// Index is the number of the footnote that this sidenote displays.
	Index int

	// Ordinal is the position of this sidenote among the sidenotes of the
	// document, counting from zero. The id attribute is derived from it rather
	// than from Index, which is not unique when a footnote is referenced more
	// than once.
	Ordinal int

	// Footnote is the definition whose contents this sidenote displays.
	Footnote *extast.Footnote
}

// KindSidenote is the NodeKind of the Sidenote node.
var KindSidenote = gast.NewNodeKind("Sidenote")

// Kind implements Node.Kind.
func (n *Sidenote) Kind() gast.NodeKind { return KindSidenote }

// Dump implements Node.Dump.
func (n *Sidenote) Dump(source []byte, level int) {
	gast.DumpHelper(n, source, level, map[string]string{
		"Index":   fmt.Sprintf("%v", n.Index),
		"Ordinal": fmt.Sprintf("%v", n.Ordinal),
	}, nil)
}

// Config holds the configuration of the extension.
type Config struct {
	// Footnotes keeps the footnote list at the end of the document, so that
	// every note is rendered twice.
	Footnotes bool

	// IDPrefix is a prefix for the id attributes of sidenotes.
	IDPrefix []byte

	// IDPrefixFunction determines the prefix for the id attribute of a given
	// node. It is consulted only when IDPrefix is empty.
	IDPrefixFunction func(gast.Node) []byte
}

// An Option configures the extension.
type Option interface {
	apply(*Config)
}

type optionFunc func(*Config)

func (f optionFunc) apply(c *Config) { f(c) }

// WithFootnotes keeps the footnote list at the end of the document in addition
// to the sidenotes, leaving it to a stylesheet to decide which of the two is
// visible.
//
// A sidenote needs a margin wide enough to hold it, so a stylesheet will
// typically show sidenotes on large displays and fall back to footnotes on
// small ones. The contents of every note appear twice in the output.
func WithFootnotes() Option {
	return optionFunc(func(c *Config) { c.Footnotes = true })
}

// WithIDPrefix sets a prefix for the id attributes of sidenotes, which is
// useful when several documents are displayed inside one HTML page.
//
// The prefix given to extension.WithFootnoteIDPrefix does not carry over:
// goldmark's footnote extension passes its options straight to its own
// renderer, so they are not among the renderer options this extension can see.
// Set the same prefix in both places when using both.
func WithIDPrefix[T []byte | string](prefix T) Option {
	return optionFunc(func(c *Config) {
		c.IDPrefix = []byte(prefix)
	})
}

// WithIDPrefixFunction sets a function that determines the prefix for the id
// attribute of a given node. It behaves like WithIDPrefix otherwise.
func WithIDPrefixFunction(f func(gast.Node) []byte) Option {
	return optionFunc(func(c *Config) {
		c.IDPrefixFunction = f
	})
}

type extender struct {
	config Config
}

// New returns a goldmark extension that renders footnotes as sidenotes.
// goldmark's footnote extension has to be enabled alongside it.
func New(opts ...Option) goldmark.Extender {
	e := &extender{}
	for _, opt := range opts {
		opt.apply(&e.config)
	}
	return e
}

// Extend implements goldmark.Extender.
func (e *extender) Extend(m goldmark.Markdown) {
	m.Parser().AddOptions(parser.WithASTTransformers(
		// goldmark's footnote extension transforms the tree at priority 999.
		// Running after it means the notes have already been numbered, the
		// unreferenced ones dropped and the list appended to the document.
		util.Prioritized(&transformer{config: e.config}, 1000),
	))
	m.Renderer().AddOptions(renderer.WithNodeRenderers(
		util.Prioritized(&nodeRenderer{Config: html.NewConfig(), config: e.config}, 500),
	))
}
