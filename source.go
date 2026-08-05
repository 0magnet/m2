// Package main source.go
package main

import (
	"bytes"
	"embed"
	"fmt"
	"io/fs"
	"os"
	"strings"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
	"github.com/gofiber/fiber/v3"
)

//go:embed *.go
var quine embed.FS

var sourceWasm = os.DirFS("wasm")
var sourcesWasm []fs.FS
var sourceCore = os.DirFS("ui")
var sourceHtml = os.DirFS("htmpl")
var sourceContent = os.DirFS("content")

func serveSourceCode(r *fiber.App) {
	for _, wasmSRC := range f.WasmSRC {
		sourcesWasm = append(sourcesWasm, os.DirFS(wasmSRC))
	}
	r.Get("/sourcecode", func(c fiber.Ctx) error {
		ret := `<!doctype html>
<html lang='en'>
<head>
<link rel="stylesheet" href="/style.css" type="text/css">
</head>
<body class='grid-container' style='background-color:black;color:white;'>
<a href='/sourcecode/go'>GO</a><br><br>

<a href='/sourcecode/html'>HTML</a><br><br>

<a href='/sourcecode/content'>Content</a><br><br>

<a href='/sourcecode/core'>C.O.R.E.</a><br><br>

<a href='/sourcecode/wasm'>WASM</a><br><br>

</body>
</html>
`
		c.Set("Content-Type", "text/html;charset=utf-8")
		_, err := c.Status(fiber.StatusOK).Write([]byte(ret))
		return err
	})

	r.Get("/sourcecode/html", sourcecodehtml)
	r.Get("/sourcecode/content", sourcecodecontent)
	r.Get("/sourcecode/go", sourcecodego)
	r.Get("/sourcecode/core", sourcecodecore)
	//	r.Get("/sourcecodewasm", sourcecodewasm)
	r.Get("/sourcecode/wasm", func(c fiber.Ctx) error {
		ret := `<!doctype html>
<html lang='en'>
<head>
<link rel="stylesheet" href="/style.css" type="text/css">
</head>
<body class='grid-container' style='background-color:black;color:white;'>
`
		for _, wasmSRC := range f.WasmSRC {
			pathNameSlc := strings.Split(wasmSRC, "/")
			pathName := pathNameSlc[len(pathNameSlc)-1]
			ret += `<a href='/sourcecode/wasm/` + pathName + `'>` + pathName + `</a><br>
			`
		}
		ret += `</body></html>
		`
		c.Set("Content-Type", "text/html;charset=utf-8")
		_, err := c.Status(fiber.StatusOK).Write([]byte(ret))
		return err
	})

	for i, wasmSRC := range f.WasmSRC {
		pathNameSlc := strings.Split(wasmSRC, "/")
		pathName := pathNameSlc[len(pathNameSlc)-1]
		r.Get("/sourcecode/wasm/"+pathName, func(c fiber.Ctx) error {
			return sourcecode(c, sourcesWasm[i], "dracula", "go")
		})
	}
}

func sourcecodehtml(c fiber.Ctx) error {
	return sourcecode(c, sourceHtml, "monokai", "html")
}
func sourcecodecontent(c fiber.Ctx) error {
	return sourcecode(c, sourceContent, "monokai", "html")
}
func sourcecodego(c fiber.Ctx) error {
	return sourcecode(c, quine, "monokai", "go")
}

func sourcecodewasm(c fiber.Ctx) error {
	return sourcecode(c, sourceWasm, "dracula", "go")
}

func sourcecodecore(c fiber.Ctx) error {
	return sourcecode(c, sourceCore, "solarized-dark256", "go")
}

func sourcecode(c fiber.Ctx, fsys fs.FS, styleName string, lang string) error {
	c.Set("Content-Type", "text/html;charset=utf-8")
	var buf bytes.Buffer
	var builder strings.Builder

	fs.WalkDir(fsys, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && strings.HasSuffix(path, "."+lang) {
			content, err := fs.ReadFile(fsys, path)
			if err != nil {
				return err
			}
			builder.WriteString(fmt.Sprintf("// ===== %s =====\n", path))
			builder.Write(content)
			builder.WriteString("\n\n")
		}
		return nil
	})

	// Pick lexer & style
	lexer := lexers.Get(lang)
	if lexer == nil {
		lexer = lexers.Fallback
	}
	lexer = chroma.Coalesce(lexer)

	style := styles.Get(styleName)
	if style == nil {
		style = styles.Fallback
	}

	// Formatter with line numbers & CSS classes
	formatter := html.New(
		html.WithLineNumbers(true),
		html.WithClasses(true),
	)

	iterator, err := lexer.Tokenise(nil, builder.String())
	if err != nil {
		return err
	}

	// Optional: include CSS in output
	var css bytes.Buffer
	_ = formatter.WriteCSS(&css, style)
	buf.WriteString("<style>")
	buf.Write(css.Bytes())
	buf.WriteString("</style>")

	if err := formatter.Format(&buf, style, iterator); err != nil {
		return err
	}

	_, err = c.Status(fiber.StatusOK).Write(buf.Bytes())
	return err
}
