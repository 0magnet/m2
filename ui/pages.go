// Package main pages.go — programmatic UI mode (no content system)
package main

import (
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	p "github.com/0magnet/m2/pkg/product"
	"cogentcore.org/core/colors"
	"cogentcore.org/core/core"
	"cogentcore.org/core/cursors"
	"cogentcore.org/core/events"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/abilities"
	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/text/rich"
	"cogentcore.org/core/text/text"
	"cogentcore.org/core/tree"
	"golang.org/x/image/draw"
	goimage "image"
	_ "image/jpeg"
	_ "image/png"
)

var pageContent *core.Frame

type catInfo struct {
	Name     string
	Products []p.Product
}

func setupPagesUI(root *core.Body, products p.Products, prodByID map[string]p.Product) *Cart {
	cart := NewCart()
	if items, ok := LoadCartState(); ok {
		cart.mu.Lock()
		cart.Items = items
		cart.mu.Unlock()
	}

	// Organize products by category
	catMap := make(map[string]*catInfo)
	for _, pr := range products {
		ci, ok := catMap[pr.Category]
		if !ok {
			ci = &catInfo{Name: pr.Category}
			catMap[pr.Category] = ci
		}
		ci.Products = append(ci.Products, pr)
	}
	var categories []*catInfo
	for _, ci := range catMap {
		categories = append(categories, ci)
	}
	sort.Slice(categories, func(i, j int) bool {
		return strings.ToLower(categories[i].Name) < strings.ToLower(categories[j].Name)
	})

	// Toolbar with Home, About, clock, Interface
	root.AddTopBar(func(bar *core.Frame) {
		tb := core.NewToolbar(bar)
		tb.Maker(func(p *tree.Plan) {
			tree.Add(p, func(w *core.Button) {
				w.SetText("Home").SetIcon(icons.Home)
				w.OnClick(func(e events.Event) { showPagesHome() })
			})
			tree.Add(p, func(w *core.Button) {
				w.SetText("About").SetIcon(icons.Info)
				w.OnClick(func(e events.Event) { showPagesAbout() })
			})
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
				w.SetText("Interface").SetIcon(icons.HtmlFill)
				w.OnClick(func(e events.Event) {
					core.TheApp.OpenURL("https://" + siteName + "/")
				})
			})
		})
		tb.Styler(func(s *styles.Style) {
			s.Font.Family = rich.Monospace
			s.Font.CustomFont = "mononoki"
			s.Text.LineHeight = 1
			s.Text.WhiteSpace = text.WhiteSpacePreWrap
		})
	})

	// Main layout: nav sidebar + content area using Splits (same as content system)
	sp := core.NewSplits(root)
	sp.SetSplits(0.2, 0.8)

	// Nav sidebar
	nav := core.NewFrame(sp)
	nav.Styler(func(s *styles.Style) {
		s.Direction = styles.Column
		s.Overflow.Y = styles.OverflowAuto
		s.Grow.Set(1, 1)
		s.Padding.Set(units.Dp(4))
		s.Gap.Set(units.Dp(0))
	})

	// Content area
	pageContent = core.NewFrame(sp)
	pageContent.Styler(func(s *styles.Style) {
		s.Direction = styles.Column
		s.Overflow.Y = styles.OverflowAuto
		s.Grow.Set(1, 1)
		s.Padding.Set(units.Dp(8))
	})

	// Nav: Categories with collapsible product lists
	for _, ci := range categories {
		cat := ci // capture for closure
		col := core.NewCollapser(nav)

		// Category name in summary — clicking it shows the category page
		catName := cat.Name
		catProds := cat.Products
		catText := core.NewText(col.Summary).SetText(fmt.Sprintf("%s (%d)", cat.Name, len(cat.Products)))
		catText.OnClick(func(e events.Event) {
			showPagesCategory(catName, catProds, cart)
		})
		catText.Styler(func(s *styles.Style) {
			s.SetAbilities(true, abilities.Hoverable)
			s.Cursor = cursors.Pointer
		})

		details := core.NewFrame(col.Details)
		details.Styler(func(s *styles.Style) {
			s.Direction = styles.Column
			s.Gap.Set(units.Dp(0))
			s.Padding.Left = units.Em(0.5)
		})

		// Individual products as compact text links
		for _, pr := range cat.Products {
			prod := pr // capture
			link := core.NewText(details).SetText(prod.Name)
			link.Styler(func(s *styles.Style) {
				s.SetAbilities(true, abilities.Hoverable, abilities.Clickable)
				s.Cursor = cursors.Pointer
				s.Font.Size = units.Dp(13)
				s.Color = colors.Scheme.Primary.Base
				s.Padding.Set(units.Dp(2), units.Dp(4))
				s.Text.WhiteSpace = text.WrapNever
				s.Max.X = units.Pw(100)
				s.Overflow.X = styles.OverflowHidden
			})
			link.OnClick(func(e events.Event) { showPagesProduct(prod, cart) })
		}
	}

	// Show home initially
	showPagesHome()

	return cart
}

func showPagesHome() {
	pageContent.DeleteChildren()
	initTitle()

	core.NewText(pageContent).SetType(core.TextHeadlineLarge).SetText(title).Styler(func(s *styles.Style) {
		s.Font.Family = rich.Custom
		s.Font.CustomFont = "mononoki"
		s.Text.WhiteSpace = text.WhiteSpacePre
		s.Text.LineHeight = 1
		s.Color = colors.Scheme.Primary.Base
	})

	// SVG + globe animation
	frame := core.NewFrame(pageContent)
	frame.Styler(func(s *styles.Style) {
		s.Display = styles.Custom
		s.Grow.Set(1, 1)
	})
	frame.OnClick(func(e events.Event) {
		pause = !pause
		if pause {
			core.MessageSnackbar(b, "animation paused")
		} else {
			core.MessageSnackbar(b, "animation resumed")
		}
	})
	mySvg = core.NewSVG(frame)
	mySvg.OpenFS(mySVG, "icon.svg") //nolint
	mySvg.Styler(func(s *styles.Style) { s.Grow.Set(1, 1) })
	c := core.NewCanvas(frame)
	c.Styler(func(s *styles.Style) {
		s.RenderBox = false
		s.Grow.Set(1, 1)
	})
	globeAnimation(c)

	core.NewText(pageContent).SetType(core.TextTitleLarge).SetText(tagLine)
	pageContent.Update()
}

func showPagesCategory(catName string, prods []p.Product, cart *Cart) {
	pageContent.DeleteChildren()

	core.NewText(pageContent).SetType(core.TextTitleLarge).SetText(catName)

	for _, pr := range prods {
		prod := pr // capture
		row := core.NewFrame(pageContent)
		row.Styler(func(s *styles.Style) {
			s.Direction = styles.Row
			s.Gap.Set(units.Dp(12))
			s.Align.Items = styles.Center
			s.Padding.Set(units.Dp(6))
			s.Border.Width.Bottom = units.Dp(1)
			s.Border.Color.Bottom = colors.Scheme.OutlineVariant
		})

		// Thumbnail
		if prod.Image1 != "" {
			thumb := core.NewImage(row)
			thumb.Styler(func(s *styles.Style) {
				s.Min.Set(units.Dp(96), units.Dp(96))
				s.Max.Set(units.Dp(96), units.Dp(96))
				s.ObjectFit = styles.FitContain
			})
			loadProductImage(thumb, productImageURL(prod))
		}

		// Info column: name, price, stock
		info := core.NewFrame(row)
		info.Styler(func(s *styles.Style) {
			s.Direction = styles.Column
			s.Grow.X = 1
			s.Gap.Set(units.Dp(2))
		})

		nameLink := core.NewText(info).SetText(prod.Name)
		nameLink.Styler(func(s *styles.Style) {
			s.SetAbilities(true, abilities.Hoverable, abilities.Clickable)
			s.Cursor = cursors.Pointer
			s.Color = colors.Scheme.Primary.Base
			s.Text.WhiteSpace = text.WrapNever
		})
		nameLink.OnClick(func(e events.Event) { showPagesProduct(prod, cart) })

		detailRow := core.NewFrame(info)
		detailRow.Styler(func(s *styles.Style) {
			s.Direction = styles.Row
			s.Gap.Set(units.Dp(16))
		})
		core.NewText(detailRow).SetText("$" + prod.Price).Styler(func(s *styles.Style) {
			s.Font.Weight = rich.Bold
		})
		core.NewText(detailRow).SetText("In stock: " + prod.Quantity)

		// Add button — fixed width column on the right
		btnFrame := core.NewFrame(row)
		btnFrame.Styler(func(s *styles.Style) {
			s.Min.X = units.Dp(100)
			s.Align.Items = styles.Center
			s.Justify.Content = styles.Center
		})
		if prod.Quantity != "0" {
			core.NewButton(btnFrame).SetText("Add").SetIcon(icons.Add).SetType(core.ButtonTonal).
				OnClick(func(e events.Event) {
					cart.Add(prod)
					core.MessageSnackbar(pageContent, fmt.Sprintf("Added %s to cart", prod.Name))
				})
		} else {
			core.NewText(btnFrame).SetText("out of stock").Styler(func(s *styles.Style) {
				s.Color = colors.Scheme.Error.Base
			})
		}
	}

	pageContent.Update()
}

// productImageURL builds the image URL for a product.
func productImageURL(prod p.Product) string {
	return "https://" + siteName + "/i/" + prod.Category + "/" + prod.Image1
}

// loadProductImage fetches an image from a URL and sets it on the widget asynchronously.
func loadProductImage(img *core.Image, url string) {
	go func() {
		resp, err := http.Get(url) //nolint:gosec
		if err != nil {
			return
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return
		}
		src, _, err := goimage.Decode(resp.Body)
		if err != nil {
			return
		}
		// Scale down to reasonable size for display
		bounds := src.Bounds()
		maxW := 400
		if bounds.Dx() > maxW {
			ratio := float64(maxW) / float64(bounds.Dx())
			newH := int(float64(bounds.Dy()) * ratio)
			dst := goimage.NewRGBA(goimage.Rect(0, 0, maxW, newH))
			draw.CatmullRom.Scale(dst, dst.Bounds(), src, bounds, draw.Over, nil)
			src = dst
		}
		img.AsyncLock()
		img.SetImage(src)
		img.Update()
		img.AsyncUnlock()
	}()
}

func showPagesProduct(prod p.Product, cart *Cart) {
	pageContent.DeleteChildren()

	core.NewText(pageContent).SetType(core.TextTitleLarge).SetText(prod.Name)

	// Product image
	if prod.Image1 != "" {
		img := core.NewImage(pageContent)
		img.Styler(func(s *styles.Style) {
			s.Max.X = units.Dp(400)
			s.Max.Y = units.Dp(400)
		})
		loadProductImage(img, productImageURL(prod))
	}

	// Build spec rows
	type spec struct{ Label, Value string }
	var specs []spec

	addSpec := func(label, value string) {
		if value != "" && value != "0" && value != "0.0" {
			specs = append(specs, spec{label, value})
		}
	}

	specs = append(specs, spec{"Price", "$" + prod.Price})
	specs = append(specs, spec{"In Stock", prod.Quantity})
	specs = append(specs, spec{"Category", prod.Category})
	if prod.Subcategory != "" {
		specs = append(specs, spec{"Subcategory", prod.Subcategory})
	}
	if prod.Description1 != "" && !strings.EqualFold(prod.Description1, prod.Name) {
		specs = append(specs, spec{"Description", prod.Description1})
	}
	addSpec("Brand", prod.Mfgname)
	addSpec("MPN", prod.Mfgpartno)
	addSpec("Voltage", prod.VoltsRating)
	if prod.Value != "" && prod.Value != "0" && prod.Value != "0.0" {
		specs = append(specs, spec{"Value", prod.Value + prod.ValUnit})
	}
	addSpec("Amperage", prod.AmpsRating)
	if prod.Tolerance != "" && prod.Tolerance != "0" {
		if f, err := strconv.ParseFloat(prod.Tolerance, 64); err == nil {
			specs = append(specs, spec{"Tolerance", fmt.Sprintf("%.2f%%", f*100)})
		}
	}
	addSpec("Type", prod.Typ)
	addSpec("Package", prod.Packagetype)
	addSpec("Technology", prod.Technology)
	addSpec("Materials", prod.Materials)
	addSpec("Watts", prod.WattsRating)
	addSpec("Year", prod.Year)
	if prod.CableLengthInches != "" && prod.CableLengthInches != "0" && prod.CableLengthInches != "0.0" {
		specs = append(specs, spec{"Cable Length", prod.CableLengthInches + " inches"})
	}
	if prod.WeightOz != "" && prod.WeightOz != "0" && prod.WeightOz != "0.0" {
		specs = append(specs, spec{"Weight", prod.WeightOz + " oz"})
	}
	if prod.TempRating != "" && prod.TempRating != "0" && prod.TempRating != "0.0" {
		specs = append(specs, spec{"Temp Rating", prod.TempRating + prod.TempUnit})
	}
	addSpec("Condition", prod.Condition)
	addSpec("Datasheet", prod.Datasheet)
	addSpec("Docs", prod.Docs)
	addSpec("Note", prod.Note)
	addSpec("Warning", prod.Warning)
	if prod.Description2 != "" {
		specs = append(specs, spec{"Additional Info", prod.Description2})
	}

	for _, sp := range specs {
		row := core.NewFrame(pageContent)
		row.Styler(func(s *styles.Style) {
			s.Direction = styles.Row
			s.Gap.Set(units.Dp(12))
			s.Text.WhiteSpace = text.WrapNever
			s.Overflow.X = styles.OverflowHidden
		})
		lbl := core.NewText(row).SetText(sp.Label + ":")
		lbl.Styler(func(s *styles.Style) {
			s.Min.X = units.Em(10)
			s.Max.X = units.Em(10)
			s.Font.Weight = rich.Bold
			s.Text.WhiteSpace = text.WrapNever
		})
		core.NewText(row).SetText(sp.Value).Styler(func(s *styles.Style) {
			s.Text.WhiteSpace = text.WrapNever
		})
	}

	if prod.Quantity != "0" {
		core.NewButton(pageContent).SetText("Add to Cart").SetIcon(icons.Add).
			OnClick(func(e events.Event) {
				cart.Add(prod)
				core.MessageSnackbar(pageContent, fmt.Sprintf("Added %s to cart", prod.Name))
			})
	}

	pageContent.Update()
}

func showPagesAbout() {
	pageContent.DeleteChildren()
	core.NewText(pageContent).SetType(core.TextTitleLarge).SetText("About")
	core.NewText(pageContent).SetText(`Magnetosphere is an electronic surplus webstore

Our sincerest thanks to:
  BG Micro - bgmicro.com
  Tanner Electronics - tannerelectronics.com
  the Bunker of Doom - bunkerofdoom.com

In memory of Lewis Cearly - Nortex Electronics
In memory of Billy Gage - BG Micro

This website is made with golang & golang webassembly - Now with CogentCore`).Styler(func(s *styles.Style) {
		s.Text.WhiteSpace = text.WhiteSpacePre
	})
	pageContent.Update()
}
