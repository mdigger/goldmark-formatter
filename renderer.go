package formatter

import (
	"bytes"
	"fmt"
	"io"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/yuin/goldmark/ast"
	east "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/renderer"
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
		"&mdash;":  `--`,
		"&hellip;": `...`,
	}
)

type render struct{}

// Markdown is a markdown format renderer.
var Markdown renderer.Renderer = new(render)

// AddOptions adds given option to this renderer.
func (r *render) AddOptions(opts ...renderer.Option) {}

// Write render node as Markdown.
func (r *render) Render(w io.Writer, source []byte, node ast.Node) error {
	return ast.Walk(node, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		switch n := node.(type) {

		case *ast.Document:
			// markdown metadata if defined
			if meta := n.Meta(); len(meta) > 0 {
				_, _ = io.WriteString(w, "---\n")

				enc := yaml.NewEncoder(w)
				err := enc.Encode(meta)
				enc.Close()
				if err != nil {
					return ast.WalkStop, err
				}

				_, _ = io.WriteString(w, "---\n")
			}

		case *ast.Heading:
			if entering {
				if !STXHeader || n.Level > 2 {
					_, _ = fmt.Fprintf(w, "%s ", "######"[:n.Level])
				}

			} else {
				if n.Attributes() != nil {
					_, _ = io.WriteString(w, " ")
					renderAttributes(w, n)
				}

				if STXHeader && n.Level < 3 {
					_, _ = io.WriteString(w, "\n")

					lines := n.Lines()
					var length int
					if LineBreak {
						line := lines.At(lines.Len() - 1)
						length = utf8.RuneCount(line.Value(source))
					} else {
						for i := 0; i < lines.Len(); i++ {
							line := lines.At(i)
							length += utf8.RuneCount(line.Value(source))
						}
					}

					divider := []byte("=")
					if n.Level == 2 {
						divider = []byte("-")
					}

					_, _ = w.Write(bytes.Repeat(divider, length))
				}

				_, _ = io.WriteString(w, "\n\n")
			}

		case *ast.Blockquote:
			if entering {
				if n.Attributes() != nil {
					renderAttributes(w, n)
					_, _ = io.WriteString(w, "\n")
				}

				var buf bytes.Buffer
				for child := n.FirstChild(); child != nil; child = child.NextSibling() {
					_ = r.Render(&buf, source, child)
				}

				text := bytes.TrimSpace(buf.Bytes())
				lines := bytes.SplitAfter(text, []byte{'\n'})
				for _, line := range lines {
					_, _ = io.WriteString(w, ">")
					if len(line) > 0 && line[0] != '>' && line[0] != '\n' {
						_, _ = io.WriteString(w, " ")
					}
					_, _ = w.Write(line)
				}

				_, _ = io.WriteString(w, "\n\n")
				return ast.WalkSkipChildren, nil
			}

		case *ast.CodeBlock:
			if entering {
				lines := n.Lines()
				for i := 0; i < lines.Len(); i++ {
					_, _ = io.WriteString(w, "    ")
					line := lines.At(i)
					_, _ = w.Write(line.Value(source))
				}

				_, _ = io.WriteString(w, "\n")
				return ast.WalkSkipChildren, nil
			}

		case *ast.FencedCodeBlock:
			if entering {
				if n.Attributes() != nil {
					renderAttributes(w, n)
					_, _ = io.WriteString(w, "\n")
				}

				_, _ = io.WriteString(w, FencedCodeBlock)
				if n.Info != nil {
					_, _ = w.Write(n.Info.Segment.Value(source))
				}
				_, _ = io.WriteString(w, "\n")

				lines := n.Lines()
				for i := 0; i < lines.Len(); i++ {
					line := lines.At(i)
					_, _ = w.Write(line.Value(source))
				}

				_, _ = io.WriteString(w, FencedCodeBlock+"\n\n")
				return ast.WalkSkipChildren, nil
			}

		case *ast.HTMLBlock:
			if entering {
				lines := n.Lines()
				for i := 0; i < lines.Len(); i++ {
					line := lines.At(i)
					_, _ = w.Write(line.Value(source))
				}

			} else {
				if n.HasClosure() {
					_, _ = w.Write(n.ClosureLine.Value(source))
				}

				_, _ = io.WriteString(w, "\n")
			}

		case *ast.List:
			if entering {
				if n.Attributes() != nil {
					renderAttributes(w, n)
					_, _ = io.WriteString(w, "\n")
				}

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
						_ = r.Render(&buf, source, chld)
					}

					// print list item
					if n.IsOrdered() {
						fmt.Fprintf(w, "%d", start)
						start++
					}
					switch {
					case UseListMarker:
						_, _ = fmt.Fprintf(w, "%c ", n.Marker)
					case n.IsOrdered():
						_, _ = io.WriteString(w, ". ")
					default:
						_, _ = io.WriteString(w, "- ")
					}

					text := bytes.TrimSpace(buf.Bytes())
					buf.Reset()

					lines := bytes.SplitAfter(text, []byte{'\n'})
					for i, line := range lines {
						if i > 0 && len(line) > 0 && line[0] != '\n' {
							_, _ = io.WriteString(w, indent)
						}
						_, _ = w.Write(line)
					}

					_, _ = io.WriteString(w, "\n")
					if !n.IsTight {
						_, _ = io.WriteString(w, "\n")
					}
				}

				if n.IsTight {
					_, _ = io.WriteString(w, "\n")
				}

				return ast.WalkSkipChildren, nil
			}

		case *ast.ListItem:
			// return ast.WalkSkipChildren, nil

		case *ast.Paragraph:
			if entering {
				if n.Attributes() != nil {
					renderAttributes(w, n)
					_, _ = io.WriteString(w, "\n")
				}

				if _, ok := n.PreviousSibling().(*ast.TextBlock); ok {
					_, _ = io.WriteString(w, "\n")
				}

			} else {
				_, _ = io.WriteString(w, "\n\n")
			}

		case *ast.TextBlock:
			if !entering {
				if _, ok := n.NextSibling().(ast.Node); ok && n.FirstChild() != nil {
					_, _ = io.WriteString(w, "\n")
				}
			}

		case *ast.ThematicBreak:
			if entering {
				if n.Attributes() != nil {
					renderAttributes(w, n)
					_, _ = io.WriteString(w, "\n")
				}

				_, _ = io.WriteString(w, ThematicBreak+"\n\n")
			}

		case *ast.AutoLink:
			if entering {
				_, _ = fmt.Fprintf(w, "<%s>", n.Label(source))
			}

		case *ast.CodeSpan:
			_, _ = io.WriteString(w, "`")

		case *ast.Emphasis:
			// io.WriteString(w, "**"[:n.Level])
			if n.Level == 1 {
				_, _ = io.WriteString(w, "_")
			} else {
				_, _ = io.WriteString(w, "**")
			}

		case *ast.Link:
			if entering {
				_, _ = io.WriteString(w, "[")
			} else {
				_, _ = io.WriteString(w, "](")
				_, _ = w.Write(n.Destination)
				if n.Title != nil {
					_, _ = fmt.Fprintf(w, " %q", n.Title)
				}
				_, _ = io.WriteString(w, ")")

				if n.Attributes() != nil {
					renderAttributes(w, n)
				}

			}

		case *ast.Image:
			if entering {
				_, _ = io.WriteString(w, "![")
			} else {
				_, _ = io.WriteString(w, "](")
				_, _ = w.Write(n.Destination)

				if n.Title != nil {
					fmt.Fprintf(w, " %q", n.Title)
				}
				_, _ = io.WriteString(w, ")")

				if n.Attributes() != nil {
					renderAttributes(w, n)
				}
			}

		case *ast.RawHTML:
			if !SkipHTML && entering {
				lines := n.Segments
				for i := 0; i < lines.Len(); i++ {
					line := lines.At(i)
					_, _ = w.Write(line.Value(source))
				}
			}

			return ast.WalkSkipChildren, nil

		case *ast.Text:
			if entering {
				_, _ = w.Write(n.Segment.Value(source))
				if n.SoftLineBreak() {
					switch {
					case n.HardLineBreak():
						_, _ = io.WriteString(w, "\\\n")
					case LineBreak:
						_, _ = io.WriteString(w, "\n")
					default:
						_, _ = io.WriteString(w, " ")
					}
				}
			}

		case *ast.String:
			if entering {
				if n.IsCode() && len(EntityReplacement) > 0 {
					_, _ = w.Write(reHTMLEntity.ReplaceAllFunc(n.Value, func(ent []byte) []byte {
						if val, ok := EntityReplacement[string(ent)]; ok {
							return []byte(val)
						}
						return ent
					}))

				} else {
					_, _ = w.Write(n.Value)
				}
			}

		case *east.Strikethrough:
			_, _ = io.WriteString(w, "~~")

		case *east.TaskCheckBox:
			if entering {
				if n.IsChecked {
					_, _ = io.WriteString(w, "[x] ")
				} else {
					_, _ = io.WriteString(w, "[ ] ")
				}
			}

		case *east.FootnoteLink:
			if entering {
				_, _ = fmt.Fprintf(w, "[^%d]", n.Index)
			}

		case *east.Footnote:
			if entering {
				_, _ = fmt.Fprintf(w, "[^%d]: ", n.Index)
				var buf bytes.Buffer
				for child := n.FirstChild(); child != nil; child = child.NextSibling() {
					_ = r.Render(&buf, source, child)
				}

				text := bytes.TrimSpace(buf.Bytes())
				lines := bytes.SplitAfter(text, []byte{'\n'})
				for i, line := range lines {
					if i > 0 && len(line) > 0 && line[0] != '\n' {
						_, _ = io.WriteString(w, "    ")
					}
					_, _ = w.Write(line)
				}
				_, _ = io.WriteString(w, "\n\n")

				return ast.WalkSkipChildren, nil
			}

		case *east.FootnoteBacklink:

		case *east.FootnoteList:
			if entering {
				_, _ = io.WriteString(w, "\n")

				if n.Attributes() != nil {
					renderAttributes(w, n)
					_, _ = io.WriteString(w, "\n")
				}
			}

		case *east.DefinitionList:
			if !entering {
				_, _ = io.WriteString(w, "\n")

				if n.Attributes() != nil {
					renderAttributes(w, n)
					_, _ = io.WriteString(w, "\n")
				}
			}

		case *east.DefinitionTerm:
			if !entering {
				_, _ = io.WriteString(w, "\n")
			}

		case *east.DefinitionDescription:
			if entering {
				_, _ = io.WriteString(w, ": ")

				var buf bytes.Buffer
				for child := n.FirstChild(); child != nil; child = child.NextSibling() {
					_ = r.Render(&buf, source, child)
				}

				text := bytes.TrimSpace(buf.Bytes())
				lines := bytes.SplitAfter(text, []byte{'\n'})
				for i, line := range lines {
					if i > 0 && len(line) > 0 && line[0] != '\n' {
						_, _ = io.WriteString(w, "  ")
					}
					_, _ = w.Write(line)
				}
				_, _ = io.WriteString(w, "\n")

				return ast.WalkSkipChildren, nil
			}

		case *east.Table:
			if entering {
				if n.Attributes() != nil {
					renderAttributes(w, n)
					_, _ = io.WriteString(w, "\n")
				}

				// collect all cells text
				var buf bytes.Buffer
				table := make([][]string, 0, n.ChildCount())
				columns := make([]int, len(n.Alignments))
				for row := n.FirstChild(); row != nil; row = row.NextSibling() {
					tableRow := make([]string, 0, len(n.Alignments))
					column := 0
					for cell := row.FirstChild(); cell != nil; cell = cell.NextSibling() {
						for child := cell.FirstChild(); child != nil; child = child.NextSibling() {
							_ = r.Render(&buf, source, child)
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
						_, _ = io.WriteString(w, "| ")
						indent := strings.Repeat(" ",
							columns[j]-utf8.RuneCountInString(cell))

						switch n.Alignments[j] {
						// case east.AlignLeft:
						case east.AlignRight:
							_, _ = io.WriteString(w, indent)
							_, _ = io.WriteString(w, cell)
						case east.AlignCenter:
							_, _ = io.WriteString(w, indent[:len(indent)/2])
							_, _ = io.WriteString(w, cell)
							_, _ = io.WriteString(w, indent[len(indent)/2:])
						default:
							_, _ = io.WriteString(w, cell)
							_, _ = io.WriteString(w, indent)
						}
						_, _ = io.WriteString(w, " ")
						// io.WriteString(w, cell)
						// TODO: add align
						// io.WriteString(w, strings.Repeat(" ",
						// 	columns[j]-utf8.RuneCountInString(cell)+1))
					}

					if i == 0 {
						_, _ = io.WriteString(w, "|\n")
						// header divider
						for j, align := range n.Alignments {
							_, _ = io.WriteString(w, "|")
							switch align {
							case east.AlignLeft:
								_, _ = io.WriteString(w, ":")
								_, _ = io.WriteString(w, strings.Repeat("-", columns[j]+1))
							case east.AlignRight:
								_, _ = io.WriteString(w, strings.Repeat("-", columns[j]+1))
								_, _ = io.WriteString(w, ":")
							case east.AlignCenter:
								_, _ = io.WriteString(w, ":")
								_, _ = io.WriteString(w, strings.Repeat("-", columns[j]))
								_, _ = io.WriteString(w, ":")
							default:
								_, _ = io.WriteString(w, strings.Repeat("-", columns[j]+2))
							}
							// _, _ = io.WriteString(w, " ")
						}
					}

					_, _ = io.WriteString(w, "|\n")
				}

				return ast.WalkSkipChildren, nil
			}

			_, _ = io.WriteString(w, "\n")

		case *east.TableHeader:
			// if entering {
			// 	io.WriteString(w, "|")
			// } else {
			// 	io.WriteString(w, "\n|")
			// 	for _, align := range n.Parent().(*east.Table).Alignments {
			// 		switch align {
			// 		case east.AlignLeft:
			// 			io.WriteString(w, " :---")
			// 		case east.AlignRight:
			// 			io.WriteString(w, " ---:")
			// 		case east.AlignCenter:
			// 			io.WriteString(w, " :---:")
			// 		default:
			// 			io.WriteString(w, " ---")
			// 		}
			// 		io.WriteString(w, " |")
			// 	}
			// 	io.WriteString(w, "\n")
			// }

		case *east.TableRow:
			// if entering {
			// 	io.WriteString(w, "|")
			// } else {
			// 	io.WriteString(w, "\n")
			// }

		case *east.TableCell:
			// if entering {
			// 	io.WriteString(w, " ")
			// } else {
			// 	io.WriteString(w, " |")
			// }
		}

		return ast.WalkContinue, nil
	})
}

var reHTMLEntity = regexp.MustCompile(`&[[:alpha:]]{5,6};`)

func renderAttributes(w io.Writer, node ast.Node) {
	var buf bytes.Buffer
	if value, ok := node.AttributeString("id"); ok {
		fmt.Fprintf(&buf, "#%s ", value)
	}

	if value, ok := node.AttributeString("class"); ok {
		for _, class := range bytes.Fields(value.([]byte)) {
			fmt.Fprintf(&buf, ".%s ", class)
		}
	}

	for _, attr := range node.Attributes() {
		switch util.BytesToReadOnlyString(attr.Name) {
		case "id", "class": // ignore
		default:
			fmt.Fprintf(&buf, "%s=%q ", attr.Name, attr.Value)
		}
	}

	fmt.Fprintf(w, "{%s}", util.TrimRightSpace(buf.Bytes()))
}
