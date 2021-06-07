package formatter

import (
	"bytes"
	"fmt"
	"io"
	"regexp"
	"strings"
	"unicode/utf8"

	lineblocks "github.com/mdigger/goldmark-lineblocks"
	"github.com/yuin/goldmark/ast"
	east "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/util"
	"gopkg.in/yaml.v3"
)

// Format settings
var (
	SkipHTML          = false
	STXHeader         = true
	LineBreak         = false
	UseListMarker     = true
	FencedCodeBlock   = "```"
	ThematicBreak     = "----"
	EntityReplacement = map[string]string{
		"&ldquo;":  `"`,
		"&rdquo;":  `"`,
		"&laquo;":  `"`,
		"&raquo;":  `"`,
		"&lsquo;":  `'`,
		"&rsquo;":  `'`,
		"&ndash;":  `--`,
		"&mdash;":  `---`,
		"&hellip;": `...`,
	}
)

// Render write node as Markdown o writer.
func Render(w io.Writer, source []byte, node ast.Node) (err error) {
	defer func() {
		if p := recover(); p != nil && err == nil {
			if e, ok := p.(error); ok {
				err = e
			} else {
				err = fmt.Errorf("%v", p)
			}
		}
	}()

	// auxiliary feature for recording
	// when an error causes panic and automatically sets the error value
	write := func(str string, a ...interface{}) {
		if _, err = fmt.Fprintf(w, str, a...); err != nil {
			panic(err)
		}
	}

	// writeAttributes write markdown attributes to writer if exists
	writeAttributes := func(node ast.Node) {
		len := len(node.Attributes())
		if len == 0 {
			return
		}

		attrs := make([]string, 0, len)

		if value, ok := node.AttributeString("id"); ok {
			attrs = append(attrs, fmt.Sprintf("#%s", value))
		}

		if value, ok := node.AttributeString("class"); ok {
			for _, class := range bytes.Fields(value.([]byte)) {
				attrs = append(attrs, fmt.Sprintf(".%s", class))
			}
		}

		for _, attr := range node.Attributes() {
			switch util.BytesToReadOnlyString(attr.Name) {
			case "id", "class": // ignore
			default:
				attrs = append(attrs, fmt.Sprintf("%s=%q ", attr.Name, attr.Value))
			}
		}

		write("{%s}", strings.Join(attrs, " "))
	}

	return ast.Walk(node, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		switch n := node.(type) {

		case *ast.Document:
			// markdown metadata if defined
			if meta := n.Meta(); len(meta) > 0 {
				write("---\n")

				enc := yaml.NewEncoder(w)
				err = enc.Encode(meta)
				enc.Close()
				if err != nil {
					return ast.WalkStop, err
				}

				write("---\n")
			}

		case *ast.Heading:
			if entering {
				if !STXHeader || n.Level > 2 {
					write("%s ", "######"[:n.Level])
				}
			} else {
				if n.Attributes() != nil {
					write(" ")
					writeAttributes(n)
				}

				if STXHeader && n.Level < 3 {
					write("\n")

					lines := n.Lines()
					var length int
					if LineBreak {
						line := lines.At(lines.Len() - 1)
						length = utf8.RuneCount(line.Value(source))
					} else {
						for i := 0; i < lines.Len(); i++ {
							line := lines.At(i)
							length += utf8.RuneCount(
								util.TrimRightSpace(line.Value(source)))
						}
					}

					divider := "="
					if n.Level == 2 {
						divider = "-"
					}
					write(strings.Repeat(divider, length))
				}

				write("\n\n")
			}

		case *ast.Blockquote:
			if entering {
				var buf bytes.Buffer
				for child := n.FirstChild(); child != nil; child = child.NextSibling() {
					if err = Render(&buf, source, child); err != nil {
						return ast.WalkStop, err
					}
				}

				text := bytes.TrimSpace(buf.Bytes())
				lines := bytes.SplitAfter(text, []byte{'\n'})
				for _, line := range lines {
					write(">")
					if len(line) > 0 && line[0] != '>' && line[0] != '\n' {
						write(" ")
					}
					write("%s", line)
				}

				return ast.WalkSkipChildren, nil
			} else {
				if n.Attributes() != nil {
					write("\n")
					writeAttributes(n)
				}
				write("\n\n")
			}

		case *ast.CodeBlock:
			if entering {
				lines := n.Lines()
				for i := 0; i < lines.Len(); i++ {
					line := lines.At(i)
					write("    %s", line.Value(source))
				}

				write("\n")
				return ast.WalkSkipChildren, nil
			}

		case *ast.FencedCodeBlock:
			if entering {
				write(FencedCodeBlock)
				if n.Info != nil {
					write("%s", n.Info.Segment.Value(source))
				}
				write("\n")

				lines := n.Lines()
				for i := 0; i < lines.Len(); i++ {
					line := lines.At(i)
					write("%s", line.Value(source))
				}

				write(FencedCodeBlock)
				return ast.WalkSkipChildren, nil
			} else {
				if n.Attributes() != nil {
					write("\n")
					writeAttributes(n)
				}
				write("\n\n")
			}

		case *ast.HTMLBlock:
			if entering {
				lines := n.Lines()
				for i := 0; i < lines.Len(); i++ {
					line := lines.At(i)
					write("%s", line.Value(source))
				}

			} else {
				if n.HasClosure() {
					write("%s", n.ClosureLine.Value(source))
				}
				write("\n")
			}

		case *ast.List:
			if entering {
				start := n.Start
				if start == 0 {
					start = 1
				}
				indent := "  "
				if n.IsOrdered() {
					indent = "   "
				}

				var buf bytes.Buffer
				// all ListItems
				for nl := n.FirstChild(); nl != nil; nl = nl.NextSibling() {
					for chld := nl.FirstChild(); chld != nil; chld = chld.NextSibling() {
						if err = Render(&buf, source, chld); err != nil {
							return ast.WalkStop, err
						}
					}

					// print list item
					if n.IsOrdered() {
						write("%d", start)
						start++
					}
					switch {
					case UseListMarker:
						write("%c ", n.Marker)
					case n.IsOrdered():
						write(". ")
					default:
						write("- ")
					}

					text := bytes.TrimSpace(buf.Bytes())
					buf.Reset()

					lines := bytes.SplitAfter(text, []byte{'\n'})
					for i, line := range lines {
						if i > 0 && len(line) > 0 && line[0] != '\n' {
							write(indent)
						}
						write("%s", line)
					}

					write("\n")
					if !n.IsTight {
						write("\n")
					}
				}

				if n.IsTight {
					write("\n")
				}

				return ast.WalkSkipChildren, nil
			}

		case *ast.ListItem:
			// return ast.WalkSkipChildren, nil

		case *ast.Paragraph:
			if entering {
				if _, ok := n.PreviousSibling().(*ast.TextBlock); ok {
					write("\n")
				}
			} else {
				if n.Attributes() != nil {
					write("\n")
					writeAttributes(n)
				}
				write("\n\n")
			}

		case *ast.TextBlock:
			if !entering {
				if _, ok := n.NextSibling().(ast.Node); ok && n.FirstChild() != nil {
					write("\n")
				}
			}

		case *ast.ThematicBreak:
			if entering {
				write(ThematicBreak)
			} else {
				if n.Attributes() != nil {
					writeAttributes(n)
					write("\n")
				}
				write("\n\n")
			}

		case *ast.AutoLink:
			if entering {
				write("<%s>", n.Label(source))
			}

		case *ast.CodeSpan:
			write("`")

		case *ast.Emphasis:
			if n.Level == 1 {
				write("_")
			} else {
				write("**")
			}

		case *ast.Link:
			if entering {
				write("[")
			} else {
				write("](%s", n.Destination)
				if n.Title != nil {
					write(" %q", n.Title)
				}
				write(")")
				writeAttributes(n)
			}

		case *ast.Image:
			if entering {
				write("![")
			} else {
				write("](%s", n.Destination)
				if n.Title != nil {
					write(" %q", n.Title)
				}
				write(")")
				writeAttributes(n)
			}

		case *ast.RawHTML:
			if !SkipHTML && entering {
				lines := n.Segments
				for i := 0; i < lines.Len(); i++ {
					line := lines.At(i)
					write("%s", line.Value(source))
				}
			}

			return ast.WalkSkipChildren, nil

		case *ast.Text:
			if entering {
				write("%s", n.Segment.Value(source))
				if n.SoftLineBreak() {
					switch {
					case n.HardLineBreak():
						write("\\\n")
					case LineBreak:
						write("\n")
					default:
						write(" ")
					}
				}
			}

		case *ast.String:
			if entering {
				if n.IsCode() && len(EntityReplacement) > 0 {
					write("%s", reHTMLEntity.ReplaceAllFunc(n.Value, func(ent []byte) []byte {
						if val, ok := EntityReplacement[string(ent)]; ok {
							return []byte(val)
						}
						return ent
					}))
				} else {
					write("%s", n.Value)
				}
			}

		case *east.Strikethrough:
			write("~~")

		case *east.TaskCheckBox:
			if entering {
				if n.IsChecked {
					write("[x] ")
				} else {
					write("[ ] ")
				}
			}

		case *east.FootnoteLink:
			if entering {
				write("[^%d]", n.Index)
			}

		case *east.Footnote:
			if entering {
				write("[^%d]: ", n.Index)
				var buf bytes.Buffer
				for child := n.FirstChild(); child != nil; child = child.NextSibling() {
					if err = Render(&buf, source, child); err != nil {
						return ast.WalkStop, err
					}
				}

				text := bytes.TrimSpace(buf.Bytes())
				lines := bytes.SplitAfter(text, []byte{'\n'})
				for i, line := range lines {
					if i > 0 && len(line) > 0 && line[0] != '\n' {
						write("    ")
					}
					write("%s", line)
				}
				write("\n\n")

				return ast.WalkSkipChildren, nil
			}

		case *east.FootnoteBacklink:

		case *east.FootnoteList:
			if entering {
				write("\n")
			} else {
				write("\n")
				if n.Attributes() != nil {
					writeAttributes(n)
					write("\n")
				}
			}

		case *east.DefinitionList:
			if !entering {
				write("\n")
				if n.Attributes() != nil {
					writeAttributes(n)
					write("\n")
				}
			}

		case *east.DefinitionTerm:
			if !entering {
				write("\n")
			}

		case *east.DefinitionDescription:
			if entering {
				write(": ")

				var buf bytes.Buffer
				for child := n.FirstChild(); child != nil; child = child.NextSibling() {
					if err = Render(&buf, source, child); err != nil {
						return ast.WalkStop, err
					}
				}

				text := bytes.TrimSpace(buf.Bytes())
				lines := bytes.SplitAfter(text, []byte{'\n'})
				for i, line := range lines {
					if i > 0 && len(line) > 0 && line[0] != '\n' {
						write("  ")
					}
					write("%s", line)
				}
				write("\n")

				return ast.WalkSkipChildren, nil
			}

		case *east.Table:
			if entering {
				// collect all cells text
				var buf bytes.Buffer
				table := make([][]string, 0, n.ChildCount())
				columns := make([]int, len(n.Alignments))
				for row := n.FirstChild(); row != nil; row = row.NextSibling() {
					tableRow := make([]string, 0, len(n.Alignments))
					column := 0
					for cell := row.FirstChild(); cell != nil; cell = cell.NextSibling() {
						for child := cell.FirstChild(); child != nil; child = child.NextSibling() {
							if err = Render(&buf, source, child); err != nil {
								return ast.WalkStop, err
							}
						}
						text := buf.String()
						if l := utf8.RuneCountInString(text); l > columns[column] {
							columns[column] = l
						}
						tableRow = append(tableRow, text)
						buf.Reset()
						column++
					}

					table = append(table, tableRow)
				}

				for i, row := range table {
					for j, cell := range row {
						indent := strings.Repeat(" ",
							columns[j]-utf8.RuneCountInString(cell))

						switch n.Alignments[j] {
						case east.AlignRight:
							write("| %s%s ", indent, cell)
						case east.AlignCenter:
							write("| %s%s%s ", indent[:len(indent)/2], cell, indent[len(indent)/2:])
						default:
							write("| %s%s ", cell, indent)
						}
					}

					if i == 0 {
						write("|\n")
						// header divider
						for j, align := range n.Alignments {
							switch align {
							case east.AlignLeft:
								write("|:%s", strings.Repeat("-", columns[j]+1))
							case east.AlignRight:
								write("|%s:", strings.Repeat("-", columns[j]+1))
							case east.AlignCenter:
								write("|:%s:", strings.Repeat("-", columns[j]))
							default:
								write("|%s", strings.Repeat("-", columns[j]+2))
							}
						}
					}

					write("|\n")
				}

				return ast.WalkSkipChildren, nil
			} else {
				if n.Attributes() != nil {
					writeAttributes(n)
					write("\n")
				}
			}

			write("\n")

		case *east.TableHeader:

		case *east.TableRow:

		case *east.TableCell:

		case *lineblocks.LineBlock:
			if !entering {
				write("\n")
			}

		case *lineblocks.LineBlockItem:
			if entering {
				write("| ")
				for i := 0; i < n.Padding; i++ {
					write(" ")
				}
			} else {
				write("\n")
			}
		}

		return ast.WalkContinue, nil
	})
}

var reHTMLEntity = regexp.MustCompile(`&[[:alpha:]]{5,6};`)
