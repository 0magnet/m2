// Package main 🌐.go
package main

import (
	"embed"
	"encoding/json"
	"io/fs"
	"log"
	"reflect"
	"time"

	p "github.com/0magnet/m2/pkg/product"
	"github.com/0magnet/calvin"

	"github.com/bitfield/script"
	"cogentcore.org/core/content"
	"cogentcore.org/core/core"
	"cogentcore.org/core/htmlcore"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/text/fonts"
	"cogentcore.org/core/text/rich"
	"cogentcore.org/core/text/text"
	"cogentcore.org/core/tree"
	"cogentcore.org/core/yaegicore/coresymbols"
)

//go:embed mononoki/*.ttf
var mononoki embed.FS

//go:embed icon.svg
var mySVG embed.FS

var origin = Origin()
var host = Host()
var siteName = host

// uiMode is set via -ldflags "-X 'main.uiMode=pages'" to use the
// programmatic Pages UI instead of the content system.
var uiMode string

var b = core.NewBody(calvin.BlackboardBold(siteName))

func main() {
	var err error
	if siteName == "" {
		siteName, err = script.Echo(coreToml).Match("NamePrefix").Replace(`NamePrefix = "`, "").Replace(`"`, "").String()
		if err != nil {
			log.Fatal(err)
		}
	}
	b = core.NewBody(calvin.BlackboardBold(siteName))
	data, err := fs.ReadFile(mySVG, "icon.svg")
	if err != nil {
		panic("failed to read icon.svg: " + err.Error())
	}
	core.AppIcon = string(data)
	fonts.AddEmbedded(mononoki)

	core.TheApp.SetSceneInit(func(sc *core.Scene) {
		sc.SetWidgetInit(func(w core.Widget) {
			w.AsWidget().Styler(func(s *styles.Style) {
				s.Font.Family = rich.Custom
				s.Font.CustomFont = "mononoki"
				s.Text.LineHeight = 1
				s.Text.WhiteSpace = text.WhiteSpacePreWrap
			})
		})
	})
	b.Styler(func(s *styles.Style) {
		s.Font.Family = rich.Custom
		s.Font.CustomFont = "mononoki"
		s.Text.LineHeight = 1
		s.Text.WhiteSpace = text.WhiteSpacePreWrap
	})

	var products p.Products
	if err := json.Unmarshal([]byte(productsJSON), &products); err != nil {
		log.Fatalf("Error unmarshaling JSON: %v", err)
	}
	prodByID := make(map[string]p.Product, len(products))
	for _, pr := range products {
		prodByID[pr.Partno] = pr
	}

	if uiMode == "pages" {
		cart := setupPagesUI(b, products, prodByID)
		SetupCartUI(b, nil, prodByID, cart)
	} else {
		ct := content.NewContent(b).SetSource(econtent)
		ctx := ct.Context

		b.AddTopBar(func(bar *core.Frame) {
			tb := core.NewToolbar(bar)
			tb.Maker(ct.MakeToolbar)
			tb.Maker(func(p *tree.Plan) {
				tree.Add(p, func(w *core.Text) {
					w.Updater(func() {
						w.SetText(time.Now().Format("Mon Jan 2 2006 15:04:05"))
					})
					go func() {
						ticker := time.NewTicker(time.Second)
						defer ticker.Stop()
						for range ticker.C {
							if !w.IsVisible() {
								continue
							}
							w.AsyncLock()
							w.UpdateRender()
							w.AsyncUnlock()
						}
					}()
				})
				tree.Add(p, func(w *core.Button) {
					ctx.LinkButton(w, "about")
					w.SetText("About").SetIcon(icons.QuestionMarkFill)
				})
				tree.Add(p, func(w *core.Button) {
					ctx.LinkButton(w, "https://"+siteName+"/")
					w.SetText("Interface").SetIcon(icons.HtmlFill)
				})
			})
			tb.AddOverflowMenu(func(m *core.Scene) {
				core.NewFuncButton(m).SetFunc(core.SettingsWindow)
			})
			tb.Styler(func(s *styles.Style) {
				s.Font.Family = rich.Monospace
				s.Font.CustomFont = "mononoki"
				s.Text.LineHeight = 1
				s.Text.WhiteSpace = text.WhiteSpacePreWrap
			})
		})

		if coresymbols.Symbols["."] == nil {
			coresymbols.Symbols["."] = make(map[string]reflect.Value)
		}
		coresymbols.Symbols["."]["econtent"] = reflect.ValueOf(econtent)

		ctx.ElementHandlers["home-page"] = homePage
		ctx.ElementHandlers["about"] = func(ctx *htmlcore.Context) bool {
			f := core.NewFrame(ctx.BlockParent)
			tree.AddChild(f, func(w *core.Text) {
				w.SetType(core.TextTitleLarge).SetText("")
			})
			return true
		}

		SetupCartUI(b, ctx, prodByID, nil)
	}

	b.RunMainWindow()
	log.Println("exiting")
}

func inlineParentNode(hctx *htmlcore.Context) tree.Node {
	switch v := any(hctx.InlineParent).(type) {
	case core.Widget:
		return v.AsTree()
	case func() core.Widget:
		return v().AsTree()
	default:
		return hctx.BlockParent.AsTree()
	}
}
