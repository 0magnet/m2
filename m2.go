// Package main m2.go
package main

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"log"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	p "github.com/0magnet/m2/pkg/product"
	"github.com/bitfield/script"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/static"
)


func main() { Execute() }

var collapseNewlines = regexp.MustCompile(`\n{2,}`)

func methodColor(method string, colors fiber.Colors) string {
	switch method {
	case fiber.MethodGet:
		return colors.Cyan
	case fiber.MethodPost:
		return colors.Green
	case fiber.MethodPut:
		return colors.Yellow
	case fiber.MethodDelete:
		return colors.Red
	case fiber.MethodPatch:
		return colors.White
	case fiber.MethodHead:
		return colors.Magenta
	case fiber.MethodOptions:
		return colors.Blue
	default:
		return colors.Reset
	}
}

func statusColor(code int, colors fiber.Colors) string {
	switch {
	case code >= fiber.StatusOK && code < fiber.StatusMultipleChoices:
		return colors.Green
	case code >= fiber.StatusMultipleChoices && code < fiber.StatusBadRequest:
		return colors.Blue
	case code >= fiber.StatusBadRequest && code < fiber.StatusInternalServerError:
		return colors.Yellow
	default:
		return colors.Red
	}
}

func server() {
	wg := new(sync.WaitGroup)
	wg.Add(1)
	initTMPL()
	r := fiber.New(fiber.Config{
		ErrorHandler: func(c fiber.Ctx, err error) error {
			code := fiber.StatusInternalServerError
			var e *fiber.Error
			if errors.As(err, &e) {
				code = e.Code
			}
			c.Set(fiber.HeaderContentType, fiber.MIMETextPlainCharsetUTF8)
			return c.Status(code).SendString(err.Error())
		},
	})

	r.Use(func(c fiber.Ctx) error {
		start := time.Now()
		err := c.Next()
		status := c.Response().StatusCode()
		lat := time.Since(start)
		colors := c.App().Config().ColorScheme
		ip := fmt.Sprintf("%*s", 15, c.IP())
		ipsStr := strings.Join(c.IPs(), ", ")
		ips := fmt.Sprintf("%*s", 15, ipsStr)
		method := fmt.Sprintf("%-*s", 6, c.Method())
		statCol := statusColor(status, colors) + fmt.Sprintf("%3d", status) + colors.Reset
		methCol := methodColor(c.Method(), colors) + method + colors.Reset
		fmt.Printf("%s | %s | %12s | %s | %s | %s | %s\n", time.Now().Format("2006-01-02 15:04:05"), statCol, lat, ip, ips, methCol, c.Path())
		return err
	})
	serveSourceCode(r)
	serveWASM(r)
	r.Get("/logo", logo)
	r.Get("/logo/:width", logo)
	r.Get("/logo/:width/:height", logo)
	r.Get("/logo.png", sendFile)
	r.Get("/logo.html", sendFile)
	r.Get("/mobilelogo.html", sendFile)
	r.Get("/logolarge.html", sendFile)
	r.Get("/favicon.ico", sendImage)
	r.Get("/robots.txt", robots)
	if f.Siteimagesrc == "" {
		r.Use("/i", static.New("./img"))
		r.Use("/img", static.New("./img"))
	}
	r.Use("/font", static.New("./font"))
	r.Get("/stl/:filename", func(c fiber.Ctx) error {
		name := c.Params("filename")
		if strings.ContainsAny(name, "/\\..") || strings.Contains(name, "..") {
			return c.SendStatus(fiber.StatusBadRequest)
		}
		return c.SendFile("./img/stl/" + name)
	})
	r.Get("/stl/base64/:filename", stlbase64)
	r.Get("/site.webmanifest", func(c fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"name":       f.Sitelongname,
			"short_name": f.Sitename,
			"icons": []fiber.Map{
				{"src": f.Siteimagesrc + "/i/android-chrome-192x192.png", "sizes": "192x192", "type": "image/png"},
				{"src": f.Siteimagesrc + "/i/android-chrome-512x512.png", "sizes": "512x512", "type": "image/png"},
			},
			"theme_color":      "#ffffff",
			"background_color": "#ffffff",
			"display":          "standalone",
		})
	})
	r.Get("/sitemap", sitemap)
	r.Get("/sitemap.xml", sitemap)
	r.Get("/", homepage)
	r.Get("/p/:partno", productpage)
	r.Get("/post/:partno", handlecat)
	r.Get("/p", handlecat)
	r.Get("/cat", handlecat)
	r.Get("/cat/:cat", handlecat)
	r.Get("/cat/:cat/:subcat", handlecat)
	r.Get("/style.css", style)
	for _, register := range extraRoutes {
		register(r)
	}
	handleOrder(r)
	if !f.NoCore {
		handleCORE(r)
	}
	go func() {
		err := r.Listen(fmt.Sprintf(":%d", f.WebPort))
		if err != nil {
			log.Println("Error serving http: ", err)
		}
		wg.Done()
	}()
	if !f.NoCore {
		watchCORE()
	}
	compileWASM()
	wg.Wait()
}

func sitemap(c fiber.Ctx) error {
	c.Type("xml", "utf-8")
	return c.SendString(generateSitemapXML())
}

// extraRoutes collects route registrars from optional drop-in files
// (see other.go.example). A drop-in appends its registrar from init();
// deleting the file removes its routes with no other code changes.
var extraRoutes []func(*fiber.App)

func logo(c fiber.Ctx) error {
	tmpl, err := auxTmpl()
	if err != nil {
		msg := fmt.Sprintf("Error parsing html template: %v", err)
		log.Println(msg)
		return c.Status(fiber.StatusInternalServerError).SendString(msg)
	}
	tmpl0, err := tmpl.Clone()
	if err != nil {
		msg := fmt.Sprintf("Error cloning template: %v", err)
		log.Println(msg)
		return c.Status(fiber.StatusInternalServerError).SendString(msg)
	}
	_, err = tmpl0.New("main").Parse(h.Logo())
	if err != nil {
		msg := fmt.Sprintf("Error parsing product page template: %v", err)
		log.Println(msg)
		return c.Status(fiber.StatusInternalServerError).SendString(msg)
	}
	tmpl = tmpl0
	c.Set("Content-Type", "text/html;charset=utf-8")

	img2txtFlags := ""
	if w, err := strconv.Atoi(c.Params("width")); err == nil {
		img2txtFlags = fmt.Sprintf("--width=%d ",w)
	}
	if h, err := strconv.Atoi(c.Params("height")); err == nil {
		img2txtFlags = fmt.Sprintf("--height=%d ",h)
	}

	logoHTMLslice, err := script.Exec(fmt.Sprintf("bash -c 'img2txt %s logo.jpg | ansifilter -H'", img2txtFlags)).Slice()
	if err != nil {
		log.Println("error: ", err)
		_, err = c.Status(fiber.StatusInternalServerError).Write([]byte(err.Error()+"/n"+strings.Join(logoHTMLslice,"\n")))
		return err
	}
	if len(logoHTMLslice) > 2 {
	    logoHTMLslice = logoHTMLslice[:len(logoHTMLslice)-3]
	}
	if len(logoHTMLslice) > 18 {
	    logoHTMLslice = logoHTMLslice[19:]
	}


	var result bytes.Buffer
	h1 := pageMeta(c, htmlTemplateData{})
	h1.Page = "logo"
	h1.Title = "logo"
	tmplData := map[string]interface{}{
		"Content": strings.Join(logoHTMLslice, "\n"),
	}
	err = tmpl.Execute(&result, tmplData)
	if err != nil {
		log.Println("error: ", err)
		_, err = c.Status(fiber.StatusInternalServerError).Write(result.Bytes())
		return err
	}
	_, err = c.Status(fiber.StatusOK).Write(collapseNewlines.ReplaceAll(result.Bytes(), []byte("\n")))
	return err
}

func robots(c fiber.Ctx) error {
	c.Set("Content-Type", "text/plain;charset=utf-8")
	_, err := c.Status(fiber.StatusOK).Write([]byte(fmt.Sprintf("User-agent: *\n\nSitemap: https://%s/sitemap.xml", c.Hostname())))
	return err
}

func style(c fiber.Ctx) error {
	c.Set("Content-Type", "text/css;charset=utf-8")
	_, err := c.Status(fiber.StatusOK).Write([]byte(h.StyleCSS()))
	return err
}

func serveWASM(r *fiber.App) {
	if f.WasmExecPath != "" {
		_, err := script.File(f.WasmExecPath).Bytes()
		if err != nil {
			log.Printf("Error reading %s: %v\n", f.WasmExecPath, err)
		} else { //the wasm exec must be present or none of the webassembly stuff will work ; provided by the golang installaton
			r.Get(f.WasmExecPathTinyGo, func(c fiber.Ctx) error {
				wasmExecData, err := script.File(f.WasmExecPathTinyGo).Bytes()
				if err != nil {
					log.Printf("Error reading %s: %v\n", f.WasmExecPathTinyGo, err)
					return c.SendStatus(fiber.StatusNotFound)
				}
				c.Set("Content-Type", "application/js")
				_, err = c.Status(fiber.StatusOK).Write(wasmExecData)
				return err
			})

			r.Get(f.WasmExecPathGo, func(c fiber.Ctx) error {
				wasmExecData, err := script.File(f.WasmExecPathGo).Bytes()
				if err != nil {
					log.Printf("Error reading %s: %v\n", f.WasmExecPathGo, err)
					return c.SendStatus(fiber.StatusNotFound)
				}
				c.Set("Content-Type", "application/js")
				_, err = c.Status(fiber.StatusOK).Write(wasmExecData)
				return err
			})

			suffix := ".wasm"
			if f.UseTinygo {
				suffix = "-tiny.wasm"
			}
			for _, wasmSRC := range f.WasmSRC {
				outputFile := strings.TrimSuffix(filepath.Base(wasmSRC), filepath.Ext(wasmSRC)) + suffix
				r.Get("/"+outputFile, func(c fiber.Ctx) error {
					data, err := script.File(outputFile).Bytes()
					if err != nil {
						script.File(outputFile).Stdout() //nolint
						return c.SendStatus(fiber.StatusInternalServerError)
					}
					c.Set("Content-Type", "application/wasm")
					return c.Status(fiber.StatusOK).Send(data)
				})
			}
		}
	}
}

func sendFile(c fiber.Ctx) error {
	return c.SendFile("." + c.Path())
}
func sendImage(c fiber.Ctx) error {
	c.Set("Content-Type", "image/jpeg")
	return c.SendFile("./img" + c.Path())
}

func stlbase64(c fiber.Ctx) error {
	name := c.Params("filename")
	if strings.ContainsAny(name, "/\\..") || strings.Contains(name, "..") {
		return c.SendStatus(fiber.StatusBadRequest)
	}
	stlfile, err := script.File("img/stl/" + name).Bytes()
	if err != nil {
		return c.SendStatus(fiber.StatusNotFound)
	}
	_, err = c.Status(fiber.StatusOK).Write([]byte("data:model/stl;base64," + base64.StdEncoding.EncodeToString(stlfile)))
	return err
}

type item struct {
	ID     string
	Amount int64
}

func cathtmlfunc(c fiber.Ctx) error {
	tmpl, err := mainTmpl()
	if err != nil {
		msg := fmt.Sprintf("Error parsing html template: %v", err)
		log.Println(msg)
		return c.Status(fiber.StatusInternalServerError).SendString(msg)
	}
	tmpl0, err := tmpl.Clone()
	if err != nil {
		msg := fmt.Sprintf("Error cloning html template: %v", err)
		log.Println(msg)
		return c.Status(fiber.StatusInternalServerError).SendString(msg)
	}
	_, err = tmpl0.New("main").Parse(h.CategoryPage())
	if err != nil {
		msg := fmt.Sprintf("Error parsing Category page template: %v", err)
		log.Println(msg)
		return c.Status(fiber.StatusInternalServerError).SendString(msg)
	}
	tmpl = tmpl0
	var tmplData map[string]interface{}
	var result bytes.Buffer
	var categoryproducts p.Products
	c.Set("Content-Type", "text/html;charset=utf-8")
	h1 := pageMeta(c, htmlPageTemplateData)
	h1.Title = fmt.Sprintf("%s | %s", func() string {
		var str string
		if c.Params("partno") != "" {
			return "No product matching partno.: " + c.Params("partno") + " | Showing All Products"
		}
		if c.Params("cat") == "" {
			return "All Products"
		}
		str = fmt.Sprintf("Category: %s", c.Params("cat"))
		if c.Params("subcat") != "" {
			str += fmt.Sprintf("; Subcategory: %s", c.Params("subcat"))
		}
		return str
	}(), h1.Title)
	h1.Page = "category"
	if c.Params("cat") == "" && c.Params("subcat") == "" {
		tmplData = map[string]interface{}{
			"Products":    allproducts,
			"Page":        h1,
			"Category":    c.Params("cat"),
			"Subcategory": c.Params("subcat"),
			"Prods":       allproducts,
			"Product":     c.Params("partno"),
		}
	} else {

		for _, prod := range allproducts {
			if strings.EqualFold(prod.Category, c.Params("cat")) && (c.Params("subcat") == "" || strings.EqualFold(escapesubcat(prod.Subcategory), c.Params("subcat"))) {
				categoryproducts = append(categoryproducts, prod)
			}
		}
		tmplData = map[string]interface{}{
			"Products":    categoryproducts,
			"Page":        h1,
			"Category":    c.Params("cat"),
			"Subcategory": c.Params("subcat"),
			"Prods":       allproducts,
		}
	}
	err = tmpl.Execute(&result, tmplData)
	if err != nil {
		msg := fmt.Sprintf("Error execute html template: %v", err)
		log.Println(msg)
		return c.Status(fiber.StatusInternalServerError).SendString(msg)
	}
	_, err = c.Status(fiber.StatusOK).Write(collapseNewlines.ReplaceAll(result.Bytes(), []byte("\n")))
	return err
}

func getcats() (cats []string) {
	var catsMap = make(map[string]int)
	for _, prod := range allproducts {
		catsMap[prod.Category]++
	}
	for cat := range catsMap {
		cats = append(cats, cat)
	}
	return cats
}
func contains(slice []string, str string) bool {
	for _, s := range slice {
		if s == str {
			return true
		}
	}
	return false
}
func getcategories(allproducts p.Products) (map[string]int, []string, map[string]map[string]int, map[string][]string) {
	categoryCounts := make(map[string]int)
	subcategoryCounts := make(map[string]map[string]int)
	subcategoriesByCategory := make(map[string][]string)

	for _, prod := range allproducts {
		if prod.Category != "" {
			categoryCounts[prod.Category]++
			if prod.Subcategory != "" {
				if subcategoryCounts[prod.Category] == nil {
					subcategoryCounts[prod.Category] = make(map[string]int)
				}
				subcategoryCounts[prod.Category][prod.Subcategory]++
				if !contains(subcategoriesByCategory[prod.Category], prod.Subcategory) {
					subcategoriesByCategory[prod.Category] = append(subcategoriesByCategory[prod.Category], prod.Subcategory)
				}
			}
		}
	}

	var sortableCategories []struct {
		Name  string
		Count int
	}
	for cat, count := range categoryCounts {
		sortableCategories = append(sortableCategories, struct {
			Name  string
			Count int
		}{Name: cat, Count: count})
	}
	sort.Slice(sortableCategories, func(i, j int) bool {
		return sortableCategories[i].Count > sortableCategories[j].Count
	})
	var sortedCategories []string
	for _, cat := range sortableCategories {
		sortedCategories = append(sortedCategories, cat.Name)
		var sortableSubcategories []struct {
			Name  string
			Count int
		}
		for subcat, count := range subcategoryCounts[cat.Name] {
			sortableSubcategories = append(sortableSubcategories, struct {
				Name  string
				Count int
			}{Name: subcat, Count: count})
		}
		sort.Slice(sortableSubcategories, func(i, j int) bool {
			return sortableSubcategories[i].Count > sortableSubcategories[j].Count
		})
		var sortedSubcategories []string
		for _, subcat := range sortableSubcategories {
			sortedSubcategories = append(sortedSubcategories, subcat.Name)
		}
		subcategoriesByCategory[cat.Name] = sortedSubcategories
	}
	return categoryCounts, sortedCategories, subcategoryCounts, subcategoriesByCategory
}

func getsubcats(cat string) (subcats []string) {
	var subcatsMap = make(map[string]int)
	for _, prod := range allproducts {
		if cat == "" || strings.EqualFold(cat, prod.Category) {
			if prod.Subcategory != "" {
				subcatsMap[escapesubcat(prod.Subcategory)]++
			}
		}
	}
	for subcat := range subcatsMap {
		subcats = append(subcats, subcat)
	}
	return subcats
}
func escapesubcat(sc string) (esc string) {
	esc = strings.Replace(sc, "¼", "quarter-", -1)
	esc = strings.Replace(esc, "½", "half-", -1)
	esc = strings.Replace(esc, "1/16", "sixteenth-", -1)
	esc = strings.Replace(esc, "%", "-pct", -1)
	esc = strings.Replace(esc, "  ", " ", -1)
	esc = strings.Replace(esc, " ", "-", -1)
	esc = strings.Replace(esc, "--", "-", -1)
	esc = strings.Replace(esc, "watt1", "watt-1", -1)
	esc = strings.Replace(esc, "watt5", "watt-5", -1)
	return esc
}

func handlecat(c fiber.Ctx) error {
	if c.Params("cat") == "" && c.Params("subcat") == "" {
		return cathtmlfunc(c)
	}
	var catexists bool
	var subcatexists bool
	catexists = false
	for _, cat := range getcats() {
		if strings.EqualFold(cat, c.Params("cat")) {
			catexists = true
			break
		}
	}
	subcatexists = false
	if c.Params("subcat") != "" {
		for _, subcat := range getsubcats("") {
			if strings.EqualFold(escapesubcat(subcat), c.Params("subcat")) {
				subcatexists = true
				break
			}
		}
	}
	if c.Params("subcat") != "" && !subcatexists {
		log.Printf("subcategory %s does not match any existing subcategory\n", c.Params("subcat"))
		return c.Redirect().To("/cat/" + c.Params("cat"))
	}
	if !catexists {
		log.Printf("category %s does not match any existing category\n", c.Params("cat"))
		return c.Redirect().To("/cat")
	}
	if catexists || (catexists && subcatexists) {
		return cathtmlfunc(c)
	}
	return c.SendStatus(fiber.StatusNotFound)
}

func homepage(c fiber.Ctx) error {
	tmpl, err := mainTmpl()
	if err != nil {
		msg := fmt.Sprintf("Could not parsing html template: %v", err)
		log.Println(msg)
		return c.Status(fiber.StatusInternalServerError).SendString(msg)
	}
	tmpl0, err := tmpl.Clone()
	if err != nil {
		msg := fmt.Sprintf("Error cloning template: %v", err)
		log.Println(msg)
		return c.Status(fiber.StatusInternalServerError).SendString(msg)
	}
	_, err = tmpl0.New("main").Parse(h.FrontPage())
	if err != nil {
		msg := fmt.Sprintf("Error parsing Front Page template: %v", err)
		log.Println(msg)
		return c.Status(fiber.StatusInternalServerError).SendString(msg)
	}
	_, err = tmpl0.New("about").Parse(h.AboutPage())
	if err != nil {
		msg := fmt.Sprintf("Error parsing About Page template: %v", err)
		log.Println(msg)
		return c.Status(fiber.StatusInternalServerError).SendString(msg)
	}
	_, err = tmpl0.New("policy").Parse(h.PolicyPage())
	if err != nil {
		msg := fmt.Sprintf("Error parsing Policy Page template: %v", err)
		log.Println(msg)
		return c.Status(fiber.StatusInternalServerError).SendString(msg)
	}
	_, err = tmpl0.New("links").Parse(h.LinksPage())
	if err != nil {
		msg := fmt.Sprintf("Error parsing Links Page template: %v", err)
		log.Println(msg)
		return c.Status(fiber.StatusInternalServerError).SendString(msg)
	}
	tmpl = tmpl0
	log.Println(c.Get("User-Agent"))
	c.Set("Content-Type", "text/html;charset=utf-8")
	h1 := pageMeta(c, htmlPageTemplateData)
	tmplData := map[string]interface{}{
		"Page":  h1,
		"Prods": allproducts,
	}
	var result bytes.Buffer
	err = tmpl.Execute(&result, tmplData)
	if err != nil {
		msg := fmt.Sprintf("Error executing template: %v", err)
		log.Println(msg)
		return c.Status(fiber.StatusInternalServerError).SendString(msg)
	}
	_, err = c.Status(fiber.StatusOK).Write(collapseNewlines.ReplaceAll(result.Bytes(), []byte("\n")))
	return err
}

func productpage(c fiber.Ctx) error {
	tmpl, err := mainTmpl()
	if err != nil {
		msg := fmt.Sprintf("Error parsing html template: %v", err)
		log.Println(msg)
		return c.Status(fiber.StatusInternalServerError).SendString(msg)
	}
	tmpl0, err := tmpl.Clone()
	if err != nil {
		msg := fmt.Sprintf("Error cloning template: %v", err)
		log.Println(msg)
		return c.Status(fiber.StatusInternalServerError).SendString(msg)
	}
	_, err = tmpl0.New("main").Parse(h.ProductPage())
	if err != nil {
		msg := fmt.Sprintf("Error parsing product page template: %v", err)
		log.Println(msg)
		return c.Status(fiber.StatusInternalServerError).SendString(msg)
	}
	tmpl = tmpl0
	c.Set("Content-Type", "text/html;charset=utf-8")
	for _, prod := range allproducts {
		if strings.EqualFold(prod.Partno, c.Params("partno")) {
			var result bytes.Buffer
			h1 := pageMeta(c, htmlPageTemplateData)
			h1.Page = "product"
			h1.Title = fmt.Sprintf("%s | %s", prod.Name, h1.Title)
			tmplData := map[string]interface{}{
				"Prod":  prod,
				"Page":  h1,
				"Prods": allproducts,
			}
			err := tmpl.Execute(&result, tmplData)
			if err != nil {
				log.Println("error: ", err)
				_, err = c.Status(fiber.StatusInternalServerError).Write(result.Bytes())
				return err
			}
			_, err = c.Status(fiber.StatusOK).Write(collapseNewlines.ReplaceAll(result.Bytes(), []byte("\n")))
			return err
		}
	}
	log.Printf("product %s does not match any existing product\n", c.Params("partno"))
	return c.Status(fiber.StatusNotFound).Redirect().To("/cat")
}
