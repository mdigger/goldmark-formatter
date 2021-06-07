package formatter

import (
	"io"

	lineblocks "github.com/mdigger/goldmark-lineblocks"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/text"
)

var DefaultMarkdown = goldmark.New(
	goldmark.WithExtensions(
		extension.GFM,
		extension.DefinitionList,
		extension.Footnote,
		lineblocks.Extension,
	),
	goldmark.WithParserOptions(
		parser.WithAttribute(),
	),
)

// MarkdownRender is a markdown format renderer.
var MarkdownRender renderer.Renderer = new(render)

// Format write reformatted markdown source to w.
func Format(source []byte, w io.Writer, opts ...parser.ParseOption) error {
	doc := DefaultMarkdown.Parser().Parse(
		text.NewReader(source), opts...)
	return MarkdownRender.Render(w, source, doc)
}
