package formatter_test

import (
	"log"
	"os"

	formatter "github.com/mdigger/goldmark-formatter"
	"github.com/yuin/goldmark"
)

func Example() {
	source := []byte("# Title\nParagraph *em **bold*** [link](/).")
	md := goldmark.New(
		goldmark.WithRenderer(formatter.Markdown), // markdown output
	)
	if err := md.Convert(source, os.Stdout); err != nil {
		log.Fatal(err)
	}
	// Output:
	// Title
	// =====
	//
	// Paragraph _em **bold**_ [link](/).
}
