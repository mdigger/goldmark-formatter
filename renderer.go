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
)

type render struct {
	config *renderer.Config
}

// NewRenderer returns a new Renderer with given options.
func NewRenderer(opts ...renderer.Option) renderer.Renderer {
	r := &render{
		config: renderer.NewConfig(),
	}
	r.AddOptions(opts...)
	return r
}

// AddOptions adds given option to this renderer.
func (r *render) AddOptions(opts ...renderer.Option) {
	for _, opt := range opts {
		opt.SetConfig(r.config)
	}
}

// HardWraps is an option name used in WithHardWraps.
const optHardWraps renderer.OptionName = "HardWraps"

func (r *render) hardWrap() bool {
	val, ok := r.config.Options[optHardWraps]
	return ok && val.(bool)
}

// Write render node as Markdown.
func (r *render) Render(w io.Writer, source []byte, node ast.Node) error {
	return ast.Walk(node, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		switch n := node.(type) {
		case *ast.Document:
			// nothing todo
		case *ast.Heading:
			if entering {
				if n.Level > 2 {
					fmt.Fprintf(w, "%s ", "######"[:n.Level])
				}
			} else {
				if n.Attributes() != nil {
					io.WriteString(w, " ")
					renderAttributes(w, n)
				}
				if n.Level < 3 {
					io.WriteString(w, "\n")
					lines := n.Lines()
					var length int
					if r.hardWrap() {
						line := lines.At(lines.Len() - 1) // last line
						length = utf8.RuneCount(line.Value(source))
					} else {
						for i := 0; i < lines.Len(); i++ {
							line := lines.At(i)
							length += utf8.RuneCount(line.Value(source))
						}
					}
					var divider = []byte("=")
					if n.Level == 2 {
						divider = []byte("-")
					}
					w.Write(bytes.Repeat(divider, length))
				}
				io.WriteString(w, "\n\n")
			}
		case *ast.Blockquote:
			if entering {
				if n.Attributes() != nil {
					renderAttributes(w, n)
					io.WriteString(w, "\n")
				}
				var buf bytes.Buffer
				for child := n.FirstChild(); child != nil; child = child.NextSibling() {
					r.Render(&buf, source, child)
				}
				text := bytes.TrimSpace(buf.Bytes())
				lines := bytes.SplitAfter(text, []byte{'\n'})
				for _, line := range lines {
					io.WriteString(w, ">")
					if len(line) > 0 && line[0] != '>' && line[0] != '\n' {
						io.WriteString(w, " ")
					}
					w.Write(line)
				}
				io.WriteString(w, "\n\n")
				return ast.WalkSkipChildren, nil
			}
		case *ast.CodeBlock:
			if entering {
				lines := n.Lines()
				for i := 0; i < lines.Len(); i++ {
					io.WriteString(w, "    ")
					line := lines.At(i)
					w.Write(line.Value(source))
				}
				io.WriteString(w, "\n")
				return ast.WalkSkipChildren, nil
			}
		case *ast.FencedCodeBlock:
			if entering {
				if n.Attributes() != nil {
					renderAttributes(w, n)
					io.WriteString(w, "\n")
				}
				io.WriteString(w, "```")
				if n.Info != nil {
					io.WriteString(w, " ")
					w.Write(n.Info.Segment.Value(source))
				}
				io.WriteString(w, "\n")
				lines := n.Lines()
				for i := 0; i < lines.Len(); i++ {
					line := lines.At(i)
					w.Write(line.Value(source))
				}
				io.WriteString(w, "```\n\n")
				return ast.WalkSkipChildren, nil
			}
		case *ast.HTMLBlock:
			if entering {
				lines := n.Lines()
				for i := 0; i < lines.Len(); i++ {
					line := lines.At(i)
					w.Write(line.Value(source))
				}
			} else {
				if n.HasClosure() {
					w.Write(n.ClosureLine.Value(source))
				}
				io.WriteString(w, "\n")
			}
		case *ast.List:
			if entering {
				if n.Attributes() != nil {
					renderAttributes(w, n)
					io.WriteString(w, "\n")
				}
				var (
					start  = n.Start
					indent = "  "
				)
				if start == 0 {
					start = 1
				}
				if n.IsOrdered() {
					indent = "   "
				}
				var buf bytes.Buffer
				// all ListItems
				for nl := n.FirstChild(); nl != nil; nl = nl.NextSibling() {
					for chld := nl.FirstChild(); chld != nil; chld = chld.NextSibling() {
						r.Render(&buf, source, chld)
					}
					// print list item
					if n.IsOrdered() {
						fmt.Fprintf(w, "%d", start)
						start++
					}
					fmt.Fprintf(w, "%c ", n.Marker)
					text := bytes.TrimSpace(buf.Bytes())
					buf.Reset()
					lines := bytes.SplitAfter(text, []byte{'\n'})
					for i, line := range lines {
						if i > 0 && len(line) > 0 && line[0] != '\n' {
							io.WriteString(w, indent)
						}
						w.Write(line)
					}
					io.WriteString(w, "\n")
					if !n.IsTight {
						io.WriteString(w, "\n")
					}
				}
				if n.IsTight {
					io.WriteString(w, "\n")
				}
				return ast.WalkSkipChildren, nil
			}
		case *ast.ListItem:
			// return ast.WalkSkipChildren, nil
		case *ast.Paragraph:
			if entering {
				if n.Attributes() != nil {
					renderAttributes(w, n)
					io.WriteString(w, "\n")
				}
				if _, ok := n.PreviousSibling().(*ast.TextBlock); ok {
					io.WriteString(w, "\n")
				}
			} else {
				io.WriteString(w, "\n\n")
			}
		case *ast.TextBlock:
			if !entering {
				if _, ok := n.NextSibling().(ast.Node); ok && n.FirstChild() != nil {
					io.WriteString(w, "\n")
				}
			}
		case *ast.ThematicBreak:
			if entering {
				if n.Attributes() != nil {
					renderAttributes(w, n)
					io.WriteString(w, "\n")
				}
				io.WriteString(w, "* * *\n\n")
			}
		case *ast.AutoLink:
			if entering {
				fmt.Fprintf(w, "<%s>", n.Label(source))
			}
		case *ast.CodeSpan:
			io.WriteString(w, "`")
		case *ast.Emphasis:
			io.WriteString(w, "**"[:n.Level])
			// if n.Level == 1 {
			// 	io.WriteString(w, "_")
			// } else {
			// 	io.WriteString(w, "**")
			// }
		case *ast.Link:
			if entering {
				io.WriteString(w, "[")
			} else {
				io.WriteString(w, "](")
				w.Write(n.Destination)
				if n.Title != nil {
					fmt.Fprintf(w, " %q", n.Title)
				}
				io.WriteString(w, ")")
				if n.Attributes() != nil {
					renderAttributes(w, n)
				}

			}
		case *ast.Image:
			if entering {
				io.WriteString(w, "![")
			} else {
				io.WriteString(w, "](")
				w.Write(n.Destination)
				if n.Title != nil {
					fmt.Fprintf(w, " %q", n.Title)
				}
				io.WriteString(w, ")")
				if n.Attributes() != nil {
					renderAttributes(w, n)
				}
			}
		case *ast.RawHTML:
			if entering {
				lines := n.Segments
				for i := 0; i < lines.Len(); i++ {
					line := lines.At(i)
					w.Write(line.Value(source))
				}
				return ast.WalkSkipChildren, nil
			}
		case *ast.Text:
			if entering {
				w.Write(n.Segment.Value(source))
				if n.SoftLineBreak() {
					if n.HardLineBreak() || r.hardWrap() {
						io.WriteString(w, "\\\n")
					} else {
						io.WriteString(w, " ")
					}
				}
			}
		case *ast.String:
			if entering {
				if n.IsCode() {
					w.Write(reHTMLEntity.ReplaceAllFunc(n.Value, htmlEntitiReplace))
				} else {
					w.Write(n.Value)
				}
			}

		case *east.Strikethrough:
			io.WriteString(w, "~~")
		case *east.TaskCheckBox:
			if entering {
				if n.IsChecked {
					io.WriteString(w, "[x] ")
				} else {
					io.WriteString(w, "[ ] ")
				}
			}
		case *east.FootnoteLink:
			if entering {
				fmt.Fprintf(w, "[^%d]", n.Index)
			}
		case *east.Footnote:
			if entering {
				fmt.Fprintf(w, "[^%d]: ", n.Index)
				var buf bytes.Buffer
				for child := n.FirstChild(); child != nil; child = child.NextSibling() {
					r.Render(&buf, source, child)
				}
				text := bytes.TrimSpace(buf.Bytes())
				lines := bytes.SplitAfter(text, []byte{'\n'})
				for i, line := range lines {
					if i > 0 && len(line) > 0 && line[0] != '\n' {
						io.WriteString(w, "    ")
					}
					w.Write(line)
				}
				io.WriteString(w, "\n\n")
				return ast.WalkSkipChildren, nil
			}
		case *east.FootnoteBackLink:
		case *east.FootnoteList:
			if entering {
				io.WriteString(w, "\n")
				if n.Attributes() != nil {
					renderAttributes(w, n)
					io.WriteString(w, "\n")
				}
			}
		case *east.DefinitionList:
			if !entering {
				io.WriteString(w, "\n")
				if n.Attributes() != nil {
					renderAttributes(w, n)
					io.WriteString(w, "\n")
				}
			}
		case *east.DefinitionTerm:
			if !entering {
				io.WriteString(w, "\n")
			}
		case *east.DefinitionDescription:
			if entering {
				io.WriteString(w, ": ")
				var buf bytes.Buffer
				for child := n.FirstChild(); child != nil; child = child.NextSibling() {
					r.Render(&buf, source, child)
				}
				// text := bytes.TrimSuffix(buf.Bytes(), []byte{'\n'})
				text := bytes.TrimSpace(buf.Bytes())
				lines := bytes.SplitAfter(text, []byte{'\n'})
				for i, line := range lines {
					if i > 0 && len(line) > 0 && line[0] != '\n' {
						io.WriteString(w, "  ")
					}
					w.Write(line)
				}
				io.WriteString(w, "\n")
				return ast.WalkSkipChildren, nil
			}
		case *east.Table:
			if entering {
				if n.Attributes() != nil {
					renderAttributes(w, n)
					io.WriteString(w, "\n")
				}
				// collect all cells text
				var buf bytes.Buffer
				var table = make([][]string, 0, n.ChildCount())
				var columns = make([]int, len(n.Alignments))
				for row := n.FirstChild(); row != nil; row = row.NextSibling() {
					var tableRow = make([]string, 0, len(n.Alignments))
					var column = 0
					for cell := row.FirstChild(); cell != nil; cell = cell.NextSibling() {
						for child := cell.FirstChild(); child != nil; child = child.NextSibling() {
							r.Render(&buf, source, child)
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
						io.WriteString(w, "| ")
						indent := strings.Repeat(" ",
							columns[j]-utf8.RuneCountInString(cell))
						switch n.Alignments[j] {
						// case east.AlignLeft:
						case east.AlignRight:
							io.WriteString(w, indent)
							io.WriteString(w, cell)
						case east.AlignCenter:
							io.WriteString(w, indent[:len(indent)/2])
							io.WriteString(w, cell)
							io.WriteString(w, indent[len(indent)/2:])
						default:
							io.WriteString(w, cell)
							io.WriteString(w, indent)
						}
						io.WriteString(w, " ")
						// io.WriteString(w, cell)
						// TODO: add align
						// io.WriteString(w, strings.Repeat(" ",
						// 	columns[j]-utf8.RuneCountInString(cell)+1))
					}
					if i == 0 {
						io.WriteString(w, "|\n")
						// header divider
						for j, align := range n.Alignments {
							io.WriteString(w, "|")
							switch align {
							case east.AlignLeft:
								io.WriteString(w, ":")
								io.WriteString(w, strings.Repeat("-", columns[j]+1))
							case east.AlignRight:
								io.WriteString(w, strings.Repeat("-", columns[j]+1))
								io.WriteString(w, ":")
							case east.AlignCenter:
								io.WriteString(w, ":")
								io.WriteString(w, strings.Repeat("-", columns[j]))
								io.WriteString(w, ":")
							default:
								io.WriteString(w, strings.Repeat("-", columns[j]+2))
							}
							// io.WriteString(w, " ")
						}
					}
					io.WriteString(w, "|\n")
				}
				return ast.WalkSkipChildren, nil
			}
			io.WriteString(w, "\n")

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

func htmlEntitiReplace(val []byte) []byte {
	switch string(val) {
	case "&ldquo;", "&rdquo;", "&laquo;", "&raquo;":
		return []byte{'"'}
	case "&lsquo;", "&rsquo;":
		return []byte{'\''}
	case "&ndash;", "&mdash;":
		return []byte("--")
	case "&hellip;":
		return []byte("...")
	default:
		return val
	}
}

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
