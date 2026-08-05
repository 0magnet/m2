// Package main tmpl.go
package main

import (
	"bytes"
	"fmt"
	htmpl "html/template"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	ttmpl "text/template"
	"time"

	p "github.com/0magnet/m2/pkg/product"
	"github.com/gofiber/fiber/v3"
)

/*
//go:embed htmpl/*
var templatesFS embed.FS

//go:embed content/*
var contentFS embed.FS
*/
/*
var (
	templatesFS = os.DirFS("htmpl")
	contentFS   = os.DirFS("content")
)
*/
/*
func mustReadEmbeddedFileToString(path string, fs embed.FS) string {
	return string(mustReadEmbeddedFileToBytes(path, fs))
}

func mustReadEmbeddedFileToBytes(path string, fs embed.FS) []byte {
	data, err := fs.ReadFile(path)
	if err != nil {
		panic(err)
	}
	return data
}
*/

func mustReadFileToString(path string) string {
	return string(mustReadFileToBytes(path))
}

// optionalReadFileToString returns "" when the file does not exist.
// Used for deployment-local assets (e.g. content/font.css) that are
// not part of the source repo.
func optionalReadFileToString(path string) string {
	data, err := os.ReadFile(path) //nolint
	if err != nil {
		return ""
	}
	return string(data)
}

// contentFile returns the deployment-local file when present, falling
// back to the committed <path>.example. Lets site operators override
// stock pages (about, policy, links) without touching tracked files.
func contentFile(path string) string {
	if s := optionalReadFileToString(path); s != "" {
		return s
	}
	return mustReadFileToString(path + ".example")
}

func mustReadFileToBytes(path string) []byte {
	data, err := os.ReadFile(path) //nolint
	if err != nil {
		panic(err)
	}
	return data
}

type htmlTemplate struct {
	Empty           func() string
	Head           func() string
	Logo           func() string
	Header         func() string
	Categories     func() string
	CatSubcats     func() string
	Footer         func() string
	MainPage       func() string
	AuxPage        func() string
	FrontPage      func() string
	CategoryPage   func() string
	CategoryPageMD func() string
	ProductPage    func() string
	ProductPageMD  func() string
	Schema         func() string
	Cart           func() string
	XMLSitemap     func() string
	Wasm           func() string
	AboutPage      func() string
	PolicyPage     func() string
	LinksPage      func() string
	CheckoutPage   func() string
	CompletePage   func() string
	CheckoutCSS    func() string
	StyleCSS       func() string
}

var h = htmlTemplate{
	Empty:           func() string { return mustReadFileToString("htmpl/empty.html") },
	Head:           func() string { return mustReadFileToString("htmpl/head.html") },
	Logo:           func() string { return mustReadFileToString("htmpl/logo.html") },
	Header:         func() string { return mustReadFileToString("htmpl/header.html") },
	Categories:     func() string { return mustReadFileToString("htmpl/categories.html") },
	CatSubcats:     func() string { return mustReadFileToString("htmpl/catsubcats.html") },
	Footer:         func() string { return mustReadFileToString("htmpl/footer.html") },
	MainPage:       func() string { return mustReadFileToString("htmpl/main.html") },
	AuxPage:        func() string { return mustReadFileToString("htmpl/aux.html") },
	FrontPage:      func() string { return mustReadFileToString("htmpl/front.html") },
	CategoryPage:   func() string { return mustReadFileToString("htmpl/category.html") },
	CategoryPageMD: func() string { return mustReadFileToString("htmpl/category.md") },
	ProductPage:    func() string { return mustReadFileToString("htmpl/product.html") },
	ProductPageMD:  func() string { return mustReadFileToString("htmpl/product.md") },
	Schema:         func() string { return mustReadFileToString("htmpl/schema.html") },
	Cart:           func() string { return mustReadFileToString("htmpl/cart.html") },
	XMLSitemap:     func() string { return mustReadFileToString("htmpl/sitemap.xml") },
	Wasm:           func() string { return mustReadFileToString("htmpl/wasm.html") },
	CompletePage:   func() string { return mustReadFileToString("htmpl/complete.html") },
	AboutPage:      func() string { return contentFile("content/about.html") },
	PolicyPage:     func() string { return contentFile("content/policy.html") },
	LinksPage:      func() string { return contentFile("content/links.html") },
	CheckoutPage:   func() string { return mustReadFileToString("content/checkout.html") },
	CheckoutCSS:    func() string { return mustReadFileToString("content/checkout.css") },
	StyleCSS: func() string {
		return optionalReadFileToString("content/font.css") + mustReadFileToString("content/style.css")
	},
}

var htmlPageTemplateData htmlTemplateData

var funcs = htmpl.FuncMap{
	"replace": replace, "mul": mul, "div": div, "safeHTML": safeHTML,
	"safeJS": safeJS, "stripProtocol": stripProtocol, "add": add, "sub": sub,
	"toFloat": toFloat, "equalsIgnoreCase": equalsIgnoreCase,
	"getsubcats": getsubcats, "escapesubcat": escapesubcat,
	"sortsubcats": sortsubcats, "repeat": repeat, "subcatlink": subcatlink,
}

func mainTmpl() (tmpl *htmpl.Template, err error) {
	tmpl = htmpl.New("index").Funcs(funcs)
	if _, err := tmpl.Parse(h.MainPage()); err != nil {
		log.Println("Error parsing index template:", err)
		return tmpl, err
	}

	partials := []struct {
		Name    string
		Content string
	}{
		{"head", h.Head()},
		{"schema", h.Schema()},
		{"header", h.Header()},
		{"catsubcats", h.CatSubcats()},
		{"categories", h.Categories()},
		{"footer", h.Footer()},
		{"cart", h.Cart()},
		{"wasm", h.Wasm()},
	}

	for _, p := range partials {
		if _, err := tmpl.New(p.Name).Parse(p.Content); err != nil {
			log.Printf("Error parsing %s template: %v", p.Name, err)
			return tmpl, err
		}
	}
	return tmpl, err
}

func auxTmpl() (tmpl *htmpl.Template, err error) {
	tmpl = htmpl.New("index").Funcs(funcs)
	if _, err := tmpl.Parse(h.AuxPage()); err != nil {
		log.Println("Error parsing index template:", err)
		return tmpl, err
	}

	partials := []struct {
		Name    string
		Content string
	}{
		{"head", h.Head()},
		{"schema", h.Empty()},
		{"wasm", h.Empty()},
	}

	for _, p := range partials {
		if _, err := tmpl.New(p.Name).Parse(p.Content); err != nil {
			log.Printf("Error parsing %s template: %v", p.Name, err)
			return tmpl, err
		}
	}
	return tmpl, err
}

func pageMeta(c fiber.Ctx, base htmlTemplateData) htmlTemplateData {
	h := base
	host := string(c.Request().Host())
	/*
		proto := "http"
		if c.Secure() {
			proto += "s"
		}
	*/
	proto := "https"
	h.Canonical = proto + "://" + host + c.OriginalURL()
	h.BaseURL = proto + "://" + host
	h.RequestHost = host
	h.Protocol = proto
	h.CatsCounts, h.Cats, h.SubCatsCounts, h.SubCatsByCat = getcategories(allproducts)
	h.LenAllProducts = len(allproducts)
	h.Time = time.Now().Format(time.RFC3339Nano)
	h.Year = fmt.Sprintf("%v", time.Now().Year())
	h.MetaDesc = f.Sitemeta
	h.KeyWords = strings.Replace(f.Sitelongname, " ", ", ", -1)
	return h
}

func initTMPL() {
	htmlPageTemplateData = htmlTemplateData{
		NoCore:             f.NoCore,
		TestMode:           f.Teststripekey,
		Title:              f.Sitelongname,
		StripePK:           f.StripePK,
		SiteName:           f.Sitedomain,
		SiteTagLine:        f.Sitetagline,
		SiteName1:          htmpl.HTML(checkerBoard(f.Sitedomain)), //nolint
		SiteLongName:       f.Sitelongname,
		SiteASCIILogo:      htmpl.HTML(f.SiteASCIILogo), //nolint
		SitePrettyName:     f.Siteprettyname,
		SitePrettyNameCap:  f.Siteprettynamecap,
		SitePrettyNameCaps: f.Siteprettynamecaps,
		TelegramContact:    f.Tgcontact,
		TelegramChannel:    f.Tgchannel,
		WasmExecPath:       f.WasmExecPath,
		WasmExecRel:        f.WasmExecPath,
		Cats:               getcats(),
		LenAllProducts:     len(allproducts),
		ImgSRC: func() (ret string) {
			ret = f.Siteimagesrc
			if ret == "" {
				ret = "/i"
			}
			return ret
		}(),
		Page: "front",
		Time: time.Now().Format(time.RFC3339Nano),
		Year: fmt.Sprintf("%v", time.Now().Year()),
	}
	htmlPageTemplateData.CatsCounts, htmlPageTemplateData.Cats, htmlPageTemplateData.SubCatsCounts, htmlPageTemplateData.SubCatsByCat = getcategories(allproducts)
	htmlPageTemplateData.WasmBinary = wasmBinary()

}

func wasmBinary() (ret []string) {
	if len(f.WasmSRC) == 0 {
		return ret
	}
	if f.UseTinygo {
		for _, wasmSRC := range f.WasmSRC {
			outputFile := strings.TrimSuffix(filepath.Base(wasmSRC), filepath.Ext(wasmSRC)) + "-tiny.wasm"
			ret = append(ret, outputFile)
		}
		return ret
	}
	for _, wasmSRC := range f.WasmSRC {
		outputFile := strings.TrimSuffix(filepath.Base(wasmSRC), filepath.Ext(wasmSRC)) + ".wasm"
		ret = append(ret, outputFile)
	}
	return ret
}

type xmlTemplateData struct {
	BaseURL      string
	Cats         []string
	SubCatsByCat map[string][]string
	Products     p.Products
	Update       string
}

func generateSitemapXML() string {
	xmlSitemapTemplateData := xmlTemplateData{
		BaseURL:  "https://" + f.Sitedomain,
		Products: allproducts,
		Update:   time.Now().Format("2006-01-02"),
	}
	_, xmlSitemapTemplateData.Cats, _, xmlSitemapTemplateData.SubCatsByCat = getcategories(allproducts)
	var err1 error
	xtmpl, err1 := ttmpl.New("index").Funcs(ttmpl.FuncMap{"getsubcats": getsubcats}).Parse(h.XMLSitemap())
	if err1 != nil {
		log.Println("Error parsing index template:", err1)
	}
	var result bytes.Buffer
	err1 = xtmpl.Execute(&result, xmlSitemapTemplateData)
	if err1 != nil {
		log.Println("error: ", err1)
	}
	return result.String()
}

func toFloat(s string) float64 {
	if s == "" {
		return 0.0
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0.0
	}
	return f
}

func checkerBoard(input string) string {
	var result strings.Builder
	for i, char := range input {
		// Wrap every other letter with the specified HTML
		if i%2 == 0 {
			result.WriteString(fmt.Sprintf("<span class='nv'>%c</span>", char))
		} else {
			result.WriteRune(char)
		}
	}
	return result.String()
}

type htmlTemplateData struct {
	Title              string
	MetaDesc           string
	Canonical          string
	BaseURL            string
	ImgSRC             string // url where images are hosted
	OrdersURL          string // url where checkout is served from
	SiteName           string
	SiteTagLine        string
	SiteName1          htmpl.HTML //checkerboard - alternate swap text & bg color
	SiteLongName       string
	SitePrettyName     string //𝕄𝕒𝕘𝕟𝕖𝕥𝕠𝕤𝕡𝕙𝕖𝕣𝕖.𝕟𝕖𝕥
	SitePrettyNameCap  string //𝕄𝕒𝕘𝕟𝕖𝕥𝕠𝕤𝕡𝕙𝕖𝕣𝕖.𝕟𝕖𝕥
	SitePrettyNameCaps string //𝕄𝔸𝔾ℕ𝔼𝕋𝕆𝕊ℙℍ𝔼ℝ𝔼.ℕ𝔼𝕋
	SiteASCIILogo      htmpl.HTML
	TelegramContact    string
	TelegramChannel    string
	Protocol           string
	RequestHost        string
	KeyWords           string
	Style              htmpl.HTML
	Heading            htmpl.HTML
	StripePK           string
	Cats               []string
	CatsCounts         map[string]int
	SubCatsCounts      map[string]map[string]int
	SubCatsByCat       map[string][]string
	LenAllProducts     int
	Mobile             bool
	Gocanvas           htmpl.HTML
	WasmBinary         []string
	WasmExecPath       string
	WasmExecRel        string
	StyleFontFace      htmpl.CSS
	Message            htmpl.HTML
	Page               string
	Year               string
	Time               string
	AboutHTML          htmpl.HTML
	LinksHTML          htmpl.HTML
	PolicyHTML         htmpl.HTML
	TestMode           bool
	NoCore             bool
}

func equalsIgnoreCase(a, b string) bool {
	return strings.EqualFold(strings.Join(strings.Fields(a), ""), strings.Join(strings.Fields(b), ""))
}

func replace(s, o, n string) string {
	return strings.ReplaceAll(s, o, n)
}
func mul(a, b float64) float64 {
	return a * b
}
func div(a, b float64) float64 {
	return a / b
}
func add(a, b int) int {
	return a + b
}
func sub(a, b int) int {
	return a - b
}
func safeHTML(s string) htmpl.HTML {
	return htmpl.HTML(s) //nolint
}
func safeJS(s string) htmpl.JS {
	return htmpl.JS(s) //nolint
}
func stripProtocol(s string) string {
	return strings.Replace(strings.Replace(s, "https://", "", -1), "http://", "", -1)
}
func repeat(s string, count int) string {
	var result string
	for i := 0; i < count; i++ {
		result += s
	}
	return result
}
func sortsubcats(subcats []string, counts map[string]map[string]int) []string {
	sort.Slice(subcats, func(i, j int) bool {
		catI, catJ := subcats[i], subcats[j]
		countI, countJ := counts[catI]["count"], counts[catJ]["count"]
		return countI > countJ
	})
	return subcats
}

func subcatlink(subcategory string) string {
	s := subcategory
	s = strings.ReplaceAll(s, "¼", "quarter-")
	s = strings.ReplaceAll(s, "½", "half-")
	s = strings.ReplaceAll(s, "1/16", "sixteenth-")
	s = strings.ReplaceAll(s, "%", "-pct")
	s = strings.ReplaceAll(s, "  ", " ")
	s = strings.ReplaceAll(s, "watt1", "watt-1")
	s = strings.ReplaceAll(s, "watt5", "watt-5")
	s = strings.ReplaceAll(s, " ", "-")
	s = strings.ReplaceAll(s, "--", "-")
	return s
}
