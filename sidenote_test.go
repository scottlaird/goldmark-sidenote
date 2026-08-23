package sidenote

import (
	"bytes"
	"strings"
	"testing"

	"github.com/yuin/goldmark"
	gast "github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/testutil"
)

func markdown(t *testing.T, opts []Option, renderOpts ...renderer.Option) goldmark.Markdown {
	t.Helper()
	return goldmark.New(
		goldmark.WithRendererOptions(append([]renderer.Option{html.WithUnsafe()}, renderOpts...)...),
		goldmark.WithExtensions(extension.Footnote, New(opts...)),
	)
}

func convert(t *testing.T, md goldmark.Markdown, source string) string {
	t.Helper()
	var buf bytes.Buffer
	if err := md.Convert([]byte(source), &buf); err != nil {
		t.Fatalf("Convert(%q): %v", source, err)
	}
	return buf.String()
}

func TestSidenote(t *testing.T) {
	testutil.DoTestCaseFile(markdown(t, nil), "_test/sidenote.txt", t, testutil.ParseCliCaseArg()...)
}

func TestSidenoteWithFootnotes(t *testing.T) {
	md := markdown(t, []Option{WithFootnotes()})
	testutil.DoTestCaseFile(md, "_test/sidenote_footnotes.txt", t, testutil.ParseCliCaseArg()...)
}

// TestXHTML pins the markup against the output of pandoc-sidenote, which
// existing Tufte CSS stylesheets are written for.
func TestXHTML(t *testing.T) {
	md := markdown(t, nil, html.WithXHTML())
	got := convert(t, md, "Text.[^1]\n\n[^1]: Most players are able to down-convert video that\ndoesn't fit right.")
	want := `<p>Text.<span class="sidenote-wrapper"><label for="sn-0" class="margin-toggle sidenote-number"></label><input type="checkbox" id="sn-0" class="margin-toggle"/><span class="sidenote">Most players are able to down-convert video that
doesn't fit right.<br />
<br />
</span></span></p>
`
	if got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

// TestIDPrefix covers a prefix being applied to sidenote and footnote ids
// alike when both are configured, which is what several documents on one page
// need. The two prefixes are separate options and have to be set separately.
func TestIDPrefix(t *testing.T) {
	md := goldmark.New(
		goldmark.WithRendererOptions(html.WithUnsafe()),
		goldmark.WithExtensions(
			extension.NewFootnote(extension.WithFootnoteIDPrefix("article12-")),
			New(WithFootnotes(), WithIDPrefix("article12-")),
		),
	)
	got := convert(t, md, "Text.[^1]\n\n[^1]: A note.")
	for _, want := range []string{`id="article12-sn-0"`, `id="article12-fn:1"`} {
		if !strings.Contains(got, want) {
			t.Errorf("output does not contain %s:\n%s", want, got)
		}
	}
}

func TestIDPrefixFunction(t *testing.T) {
	md := markdown(t, []Option{WithIDPrefixFunction(func(gast.Node) []byte {
		return []byte("fn-")
	})})
	got := convert(t, md, "Text.[^1]\n\n[^1]: A note.")
	if want := `id="fn-sn-0"`; !strings.Contains(got, want) {
		t.Errorf("output does not contain %s:\n%s", want, got)
	}
}

// TestWithoutFootnoteExtension covers the extension being harmless when
// goldmark's footnote extension is not enabled: there are no footnotes to
// render as sidenotes.
func TestWithoutFootnoteExtension(t *testing.T) {
	md := goldmark.New(
		goldmark.WithRendererOptions(html.WithUnsafe()),
		goldmark.WithExtensions(New()),
	)
	got := convert(t, md, "Text.[^1]\n\n[^1]: A note.")
	want := "<p>Text.[^1]</p>\n<p>[^1]: A note.</p>\n"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}
