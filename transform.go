package sidenote

import (
	gast "github.com/yuin/goldmark/ast"
	extast "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)

type transformer struct {
	config Config
}

// Transform places a Sidenote beside each footnote reference.
//
// It runs after the footnote extension's own transformer, which has already
// numbered the notes, appended their backlinks, dropped the definitions that
// nothing refers to, sorted the list and appended it to the document. When
// there is no list the footnote extension is not enabled, and there is nothing
// to do.
func (t *transformer) Transform(doc *gast.Document, _ text.Reader, _ parser.Context) {
	list := footnoteList(doc)
	if list == nil {
		return
	}

	definitions := map[int]*extast.Footnote{}
	for def := list.FirstChild(); def != nil; def = def.NextSibling() {
		definition, ok := def.(*extast.Footnote)
		if !ok {
			continue
		}
		definitions[definition.Index] = definition
	}
	links := footnoteLinks(doc)

	// A sidenote holds the contents of the note it points at, so a note that
	// refers to itself, directly or through other notes, would be rendered
	// inside itself without end. Map out which definition refers to which so
	// that references closing such a loop can be left as they are.
	refs := map[*extast.Footnote][]*extast.Footnote{}
	for _, link := range links {
		container, target := containingFootnote(link), definitions[link.Index]
		if container != nil && target != nil {
			refs[container] = append(refs[container], target)
		}
	}

	ordinal := 0
	for _, link := range links {
		parent := link.Parent()
		if parent == nil {
			continue
		}
		definition := definitions[link.Index]
		if definition != nil && !closesLoop(refs, definition, containingFootnote(link)) {
			parent.InsertAfter(parent, link, &Sidenote{
				Index:    link.Index,
				Ordinal:  ordinal,
				Footnote: definition,
			})
			ordinal++
		}
		if !t.config.Footnotes {
			parent.RemoveChild(parent, link)
		}
	}

	if !t.config.Footnotes {
		// The notes are rendered where they are referenced. What is left of the
		// list is only a holder for their contents, and its backlinks point at
		// references that no longer exist.
		for _, backlink := range walk(list, extast.KindFootnoteBacklink) {
			backlink.Parent().RemoveChild(backlink.Parent(), backlink)
		}
		doc.RemoveChild(doc, list)
	}
}

// footnoteList returns the list of footnotes the footnote extension appended to
// the document, or nil when there is none.
func footnoteList(doc *gast.Document) *extast.FootnoteList {
	for c := doc.LastChild(); c != nil; c = c.PreviousSibling() {
		if list, ok := c.(*extast.FootnoteList); ok {
			return list
		}
	}
	return nil
}

// footnoteLinks returns every reference to a footnote, in the order they appear
// in the document.
func footnoteLinks(doc *gast.Document) []*extast.FootnoteLink {
	var links []*extast.FootnoteLink
	for _, n := range walk(doc, extast.KindFootnoteLink) {
		links = append(links, n.(*extast.FootnoteLink))
	}
	return links
}

func walk(root gast.Node, kind gast.NodeKind) []gast.Node {
	var found []gast.Node
	_ = gast.Walk(root, func(n gast.Node, entering bool) (gast.WalkStatus, error) {
		if entering && n.Kind() == kind {
			found = append(found, n)
		}
		return gast.WalkContinue, nil
	})
	return found
}

// containingFootnote returns the footnote definition that holds the given node,
// or nil when the node is in the document itself.
func containingFootnote(n gast.Node) *extast.Footnote {
	for p := n.Parent(); p != nil; p = p.Parent() {
		if footnote, ok := p.(*extast.Footnote); ok {
			return footnote
		}
	}
	return nil
}

// closesLoop reports whether showing target inside container would begin a
// chain of references that arrives back at container. A reference in the
// document itself can never do so.
func closesLoop(refs map[*extast.Footnote][]*extast.Footnote, target, container *extast.Footnote) bool {
	if container == nil {
		return false
	}
	seen := map[*extast.Footnote]bool{}
	var reaches func(*extast.Footnote) bool
	reaches = func(from *extast.Footnote) bool {
		if from == container {
			return true
		}
		if seen[from] {
			return false
		}
		seen[from] = true
		for _, next := range refs[from] {
			if reaches(next) {
				return true
			}
		}
		return false
	}
	return reaches(target)
}
