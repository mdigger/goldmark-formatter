# goldmark-formatter

[![GoDoc](https://godoc.org/github.com/mdigger/goldmark-formatter?status.svg)](https://godoc.org/github.com/mdigger/goldmark-formatter)

This [goldmark](https://github.com/yuin/goldmark) renderer extension adds support for formatting markdown.

```golang
source := []byte("# Title\nParagraph *em **bold*** [link](/).")
md := goldmark.New(
    goldmark.WithRenderer(formatter.NewRenderer()), // markdown output
    goldmark.WithRendererOptions(
        html.WithHardWraps(),
    ),
)
if err := md.Convert(source, os.Stdout); err != nil {
    log.Fatal(err)
}
```

```markdown
Title
=====

Paragraph *em **bold*** [link](/).
```
