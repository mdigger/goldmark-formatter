package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"log"
	"os"

	formatter "github.com/mdigger/goldmark-formatter"
	"gopkg.in/yaml.v3"
)

func main() {
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(),
			"Usage of %s [flags] < input > output:\n", os.Args[0])
		flag.PrintDefaults()
	}
	var skipMetadata bool
	flag.BoolVar(&skipMetadata, "skipMetadata", false, "remove metadata front matter")
	flag.BoolVar(&formatter.SkipHTML, "skipHTML", false, "remove HTML blocks")
	flag.BoolVar(&formatter.LineBreak, "wrapLines", false, "hard wrap lines")
	flag.BoolVar(&formatter.STXHeader, "stxHeaders", false, "use STX headers")
	flag.Parse()

	// load markdown
	source, err := io.ReadAll(os.Stdin)
	if err != nil {
		log.Fatal(err)
	}
	out := os.Stdout

	// decode metadata if exists
	if bytes.HasPrefix(source, []byte("---\n")) {
		// search the end of metadata
		var (
			start = 4
			end   int
		)
	research:
		for _, marker := range []string{"\n---", "\n..."} {
			end = bytes.Index(source[start:], []byte(marker))
			if end != -1 {
				break
			}
		}
		// check find metadata
		if end != -1 {
			// parse yaml front matter
			var meta yaml.Node
			err = yaml.Unmarshal(source[4:start+end], &meta)
			if err != nil || len(meta.Content) != 1 {
				start += end + 4
				goto research
			}
			// skip metadata from source
			source = source[start+end+4:]
			if !skipMetadata {
				// rewrite metadata
				_, _ = io.WriteString(out, "---\n")
				enc := yaml.NewEncoder(out)
				err = enc.Encode(meta.Content[0])
				enc.Close()
				if err != nil {
					log.Fatal(err)
				}
				_, _ = io.WriteString(out, "---\n")
			}
		}
	}

	// parse markdown and write reformatted source
	err = formatter.Format(source, out)
	if err != nil {
		log.Fatal(err)
	}
}
