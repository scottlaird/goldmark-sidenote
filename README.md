goldmark-sidenote
=================

A [goldmark](https://github.com/yuin/goldmark) extension that renders footnotes
as **sidenotes**: notes set beside the text that refers to them, in the margin,
in the manner of [Tufte CSS](https://edwardtufte.github.io/tufte-css/#sidenotes).

The markup matches [pandoc-sidenote](https://github.com/jez/pandoc-sidenote), so
a stylesheet written for that works unchanged.

Installation
------------

```
go get github.com/scottlaird/goldmark-sidenote
```

Usage
-----

This extension renders the footnotes found by goldmark's own footnote
extension, which has to be enabled alongside it:

```go
md := goldmark.New(
	goldmark.WithExtensions(
		extension.Footnote,
		sidenote.New(),
	),
)
```

Without `extension.Footnote` there are no footnotes to render and this
extension does nothing.

Given this Markdown:

```markdown
Video is hard.[^1]

[^1]: Most players can down-convert video that doesn't fit.
```

it produces:

```html
<p>Video is hard.<span class="sidenote-wrapper"><label for="sn-0" class="margin-toggle sidenote-number"></label><input type="checkbox" id="sn-0" class="margin-toggle"><span class="sidenote">Most players can down-convert video that doesn't fit.<br>
<br>
</span></span></p>
```

Under `html.WithXHTML()` the tags are closed as `<br />` and `/>`, matching
pandoc-sidenote byte for byte.

### Numbers and ids

The number in front of a sidenote is not in the HTML. The `label` carries no
text of its own, and the stylesheet fills it from a CSS counter.

The `sn-0` in the markup is an HTML id and nothing more. It appears exactly
twice, as `id` on the checkbox and as `for` on the label, and nothing ever
links to it. Its only job is to bind the two together, which is what makes the
note expand when the number is tapped: a label, a hidden checkbox and a sibling
selector, with no JavaScript.

That id counts sidenotes, not footnotes. It runs from zero and advances once
per sidenote, so a note referenced twice yields `sn-0` and `sn-1` even though
both are footnote 1. Footnote ids are goldmark's own `fn:1` and `fnref:1`, and
those *are* link targets, for the jump to the note and the way back.

Options
-------

| Option | Description |
| ------ | ----------- |
| `sidenote.WithFootnotes()` | Keep the footnote list at the end of the document as well, so that every note is rendered twice. |
| `sidenote.WithIDPrefix(prefix)` | A prefix for the id attributes of sidenotes. |
| `sidenote.WithIDPrefixFunction(f)` | Determines the id prefix for a given node. |

### Sidenotes and footnotes together

A sidenote needs a margin wide enough to hold it, which a phone does not have.
`sidenote.WithFootnotes()` renders every note both ways and leaves it to a
stylesheet to decide which one is visible:

```go
sidenote.New(sidenote.WithFootnotes())
```

```css
@media (max-width: 760px) { .sidenote-wrapper { display: none } }
@media (min-width: 761px) { .footnotes, .footnote-ref { display: none } }
```

The contents of every note appear twice in the output in this mode, so any id
inside a note is written twice with it. In practice this only arises when one
note refers to another, whose reference carries an id.

Watch out for a note that is referenced more than once. Footnote numbering
counts notes, so both references read 1; a CSS counter on `.sidenote-number`
counts sidenotes, so they read 1 and 2. Only one of the two is visible at a
time, so nobody sees a contradiction, but a reader on a wide screen and a
reader on a phone are looking at the same document numbered differently.

A stylesheet cannot paper over this. The footnote number is text inside the
reference, and CSS has no way to read another element's text; a counter over
the references would count occurrences and land on 1 and 2 as well. Referencing
a note once is the only thing that avoids it.

### Id prefixes

`sidenote.WithIDPrefix` and `extension.WithFootnoteIDPrefix` are separate
options and have to be set separately, even though they serve the same purpose:

```go
goldmark.WithExtensions(
	extension.NewFootnote(extension.WithFootnoteIDPrefix("article12-")),
	sidenote.New(sidenote.WithFootnotes(), sidenote.WithIDPrefix("article12-")),
)
```

goldmark's footnote extension passes its options straight to its own renderer
rather than through the shared renderer options, so there is no way for this
extension to read the prefix that was configured there.

A prefix only matters when several documents are rendered into one HTML page.
Each would otherwise emit `id="sn-0"`, and a `label` binds to the first
matching id in the page, so tapping a sidenote in the second document would
expand the first document's note. One document to a page needs no prefix.

Limitations
-----------

A sidenote sits inside the paragraph that refers to it, so it cannot hold block
elements. The paragraphs of a note are flattened into one run of inline content
separated by line breaks, as pandoc-sidenote does. A note containing a list or a
code block will still render, but the result is not valid HTML inside a
paragraph.

A note referenced more than once produces one sidenote per reference, each with
its own id. A note that refers back to itself, directly or through other notes,
is rendered without the reference that closes the loop; a sidenote holds the
contents of the note it points at, so such a chain would otherwise never end.

Pandoc's inline footnote syntax, `^[a note written where it is referenced]`, is
not supported here. It has to be part of goldmark's footnote extension itself,
because an inline footnote has to share the numbering and the list that
extension keeps in unexported parser state.

License
-------

MIT. See [LICENSE](LICENSE).

The output markup follows [pandoc-sidenote](https://github.com/jez/pandoc-sidenote)
by Jake Zimmerman; no code is taken from it.
