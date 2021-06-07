package formatter_test

import (
	"log"
	"os"

	formatter "github.com/mdigger/goldmark-formatter"
)

func Example() {
	source := []byte("# Title\nParagraph *em **bold*** [link](/).")
	err := formatter.Format(source, os.Stdout)
	if err != nil {
		log.Fatal(err)
	}
	// Output:
	// Title
	// =====
	//
	// Paragraph _em **bold**_ [link](/).
}
