package formatter_test

import (
	"log"
	"os"

	formatter "github.com/mdigger/goldmark-formatter"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/renderer/html"
)

func Example() {
	source := []byte("# Title\nParagraph *em **bold*** [link](/).")
	md := goldmark.New(
		goldmark.WithRenderer(
			formatter.New(html.WithHardWraps())), // markdown output
	)
	if err := md.Convert(source, os.Stdout); err != nil {
		log.Fatal(err)
	}
	// Output:
	// Title
	// =====
	//
	// Paragraph *em **bold*** [link](/).
}
