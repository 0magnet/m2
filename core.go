// Package main core.go
package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	htmpl "html/template"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	ttmpl "text/template"
	"time"

	p "github.com/0magnet/m2/pkg/product"
	"github.com/bitfield/script"
	"github.com/briandowns/spinner"
	"github.com/gofiber/fiber/v3"
)

func watchCORE() {
	createCORE()
	buildCORE()

	files := map[string]struct {
		lastMod time.Time
		handler func()
	}{
		"ui/🌐.go": {
			handler: func() {
				createCORE()
				buildCORE()
			},
		},
		"htmpl/product.md": {
			handler: func() {
				createCORE()
				buildCORE()
			},
		},
	}

	go func() {
		for path := range files {
			fi, err := os.Stat(path)
			if err != nil {
				log.Printf("Cannot stat %s: %v", path, err)
				continue
			}
			files[path] = struct {
				lastMod time.Time
				handler func()
			}{
				lastMod: fi.ModTime(),
				handler: files[path].handler,
			}
		}

		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			for path, entry := range files {
				fi, err := os.Stat(path)
				if err != nil {
					log.Printf("Error stating %s: %v", path, err)
					continue
				}
				if fi.ModTime().After(entry.lastMod) {
					log.Println("📦 Detected change in", path)
					files[path] = struct {
						lastMod time.Time
						handler func()
					}{
						lastMod: fi.ModTime(),
						handler: entry.handler,
					}
					entry.handler()
				}
			}
		}
	}()
}

func handleCORE(r *fiber.App) {
	r.Use("/", func(c fiber.Ctx) error {
		coreUIPath := "ui/bin/web"
		trim := strings.Trim(c.Path(), "/")
		if trim == "" {
			trim = "index.html"
		}
		fullPath := filepath.Join(coreUIPath, trim)
		info, err := os.Stat(fullPath)
		if err == nil && info.IsDir() {
			fullPath = filepath.Join(fullPath, "index.html")
		}
		if _, err := os.Stat(fullPath); err != nil {
			c.Status(fiber.StatusNotFound)
			return c.SendFile(filepath.Join(coreUIPath, "404.html"))
		}
		if strings.HasSuffix(fullPath, ".wasm") {
			c.Set("Content-Type", "application/wasm")
		}
		return c.SendFile(fullPath)
	})
}

func buildCORE() {
	s := spinner.New(spinner.CharSets[14], 25*time.Millisecond)
	s.Suffix = " Building C.O.R.E UI..."
	s.Start()
	ldflags := `-X 'main.` + f.LDFlagsX + `'`
	if f.PagesUI {
		ldflags += ` -X 'main.uiMode=pages'`
	}
	_, err := script.Exec(`bash -c 'set -x ; go get -u ./... ; go mod tidy ; go mod vendor ; cd ui || exit 1 ; rm -rf bin ; time go run -x cogentcore.org/core@main build web -vv -ldflags="` + ldflags + `" || timeout 30 go run -x .'`).Stdout()
	s.Stop()
	if err != nil {
		log.Println(err)
	} else {
		log.Println("✅ Done building C.O.R.E UI")
	}
}

func createCORE() {
	log.Println("Creating Content Files")
	if _, err := script.Exec(`bash -c 'rm -rf ui/content ; cp -r ui/content-bak ui/content'`).Stdout(); err != nil {
		log.Fatalf(err.Error())
	}
	log.Println("Populating Content")
	prodPageMDTmpl, err := ttmpl.New("index").Funcs(htmpl.FuncMap{
		"replace": replace, "mul": mul, "div": div,
		"safeHTML": safeHTML, "safeJS": safeJS, "stripProtocol": stripProtocol,
		"add": add, "sub": sub, "toFloat": toFloat, "equalsIgnoreCase": equalsIgnoreCase,
		"getsubcats": getsubcats, "escapesubcat": escapesubcat,
		"sortsubcats": sortsubcats, "repeat": repeat,
	}).Parse(h.ProductPageMD())
	if err != nil {
		log.Println("Error parsing product page markdown template:", err)
		log.Fatalf(err.Error())
	}

	catPageMDTmpl, err := ttmpl.New("index").Funcs(htmpl.FuncMap{
		"replace": replace, "mul": mul, "div": div,
		"safeHTML": safeHTML, "safeJS": safeJS, "stripProtocol": stripProtocol,
		"add": add, "sub": sub, "toFloat": toFloat, "equalsIgnoreCase": equalsIgnoreCase,
		"getsubcats": getsubcats, "escapesubcat": escapesubcat,
		"sortsubcats": sortsubcats, "repeat": repeat,
	}).Parse(h.CategoryPageMD())
	if err != nil {
		log.Println("Error parsing category page markdown template:", err)
		log.Fatalf(err.Error())
	}

	var enabled []p.Product
	catSet := make(map[string]struct{})
	for _, prod := range allproducts {
		if strings.EqualFold(prod.Enable, "TRUE") {
			enabled = append(enabled, prod)
			catSet[prod.Category] = struct{}{}
		}
	}

	var cats []string
	for c := range catSet {
		cats = append(cats, c)
	}
	sort.Slice(cats, func(i, j int) bool {
		return strings.ToLower(cats[i]) < strings.ToLower(cats[j])
	})

	for _, cat := range cats {
		var prods []p.Product
		for _, prod := range enabled {
			if strings.EqualFold(prod.Category, cat) {
				prods = append(prods, prod)
			}
		}

		var buf bytes.Buffer
		if err := catPageMDTmpl.Execute(&buf, map[string]interface{}{
			"Products": prods,
			"Category": cat,
			"Domain":   f.Sitedomain,
			"Page": map[string]interface{}{
				"ImgSRC": f.Siteimagesrc,
			},
		}); err != nil {
			log.Fatalf("execute category MD (%q): %v", cat, err)
		}

		lines := strings.Split(buf.String(), "\n")
		var cleaned []string
		for _, line := range lines {
			if strings.TrimSpace(line) != "" {
				cleaned = append(cleaned, line)
			}
		}
		md := strings.Join(cleaned, "\n") + "\n"

		filename := "ui/content/" + escapesubcat(cat) + ".md"
		if _, err := script.Echo(md).WriteFile(filename); err != nil {
			log.Fatalf("write category MD (%q): %v", filename, err)
		}
	}

	for _, product := range allproducts {
		if product.Enable != "TRUE" {
			continue
		}

		var buf bytes.Buffer
		err := prodPageMDTmpl.Execute(&buf, map[string]interface{}{"Prod": product, "Domain": f.Sitedomain})
		if err != nil {
			log.Fatalf(err.Error())
		}
		lines := strings.Split(buf.String(), "\n")
		var cleanedLines []string
		for _, line := range lines {
			if strings.TrimSpace(line) != "" {
				cleanedLines = append(cleanedLines, line)
			}
		}
		cleaned := strings.Join(cleanedLines, "\n") + "\n"
		filename := "ui/content/" + escapesubcat(product.Partno) + ".md"
		_, err = script.Echo(cleaned).WriteFile(filename)
		if err != nil {
			log.Fatalf(err.Error())
		}
	}
	log.Println("Writing products.json")

	data := readproductscsv(f.ProductsCSV)
	if len(data) == 0 {
		log.Fatalf("CSV file %s is empty or unreadable", f.ProductsCSV)
	}
	scanner := bufio.NewScanner(bytes.NewReader(data))
	var jsonData []map[string]string
	var headers []string
	lineNum := 0
	for scanner.Scan() {
		line := scanner.Text()
		f := strings.Split(line, ",")
		if lineNum == 0 {
			headers = f
			lineNum++
			continue
		}
		if len(f) < 4 {
			continue
		}
		if f[3] == "TRUE" {
			entry := make(map[string]string)
			for i := range f {
				if i < len(headers) {
					entry[headers[i]] = f[i]
				}
			}
			jsonData = append(jsonData, entry)
		}
		lineNum++
	}

	if err := scanner.Err(); err != nil {
		log.Fatalf("Error scanning CSV: %v", err)
	}

	// Convert to JSON
	output, err := json.MarshalIndent(jsonData, "", "  ")
	if err != nil {
		log.Fatalf("Error encoding JSON: %v", err)
	}

	_, err = script.Echo(string(output)).WriteFile("ui/products.json")
	if err != nil {
		log.Fatalf(err.Error())
	}
}
