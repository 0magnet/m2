//Package main cart_ui.go
package main

import (
	"fmt"
	"strings"

	p "github.com/0magnet/m2/pkg/product"
	"cogentcore.org/core/colors"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/htmlcore"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/text/rich"
	"cogentcore.org/core/text/text"
)

// cartRow backs the table rows
type cartRow struct {
	Item      string `edit:"-"`
	UnitPrice string `edit:"-"`
	Qty       int
	LineTotal string `edit:"-"`
	Remove    bool
}

func SetupCartUI(root *core.Body, ctx *htmlcore.Context, prodByID map[string]p.Product, existingCart *Cart) {
	cart := existingCart
	if cart == nil {
		cart = NewCart()
		// hydrate from storage on web (no-op elsewhere)
		if items, ok := LoadCartState(); ok {
			cart.mu.Lock()
			cart.Items = items
			cart.mu.Unlock()
		}
	}

	// ===== Main section (full width with the ONLY collapser) =====
	mainSection := core.NewFrame(root)
	mainSection.Styler(func(s *styles.Style) {
		s.Direction = styles.Column
		s.Gap.Set(units.Dp(12), units.Dp(12))
		s.Grow.Set(1, 0)
		s.Padding.Set(units.Dp(4), units.Dp(4))
	})

	// The single collapser
	col := core.NewCollapser(mainSection)
	// Live-updating summary on THIS collapser
	summary := core.NewText(col.Summary).SetText("Cart (0) — $0.00")
	summary.Updater(func() {
		summary.SetText(fmt.Sprintf("Cart (%d) — %s", cart.Count(), centsToMoney(cart.TotalCents())))
	})

	// Details root we control (full width)
	detailsRoot := core.NewFrame(col.Details)
	detailsRoot.Styler(func(s *styles.Style) {
		s.Direction = styles.Column
		s.Gap.Set(units.Dp(12), units.Dp(12))
		s.Grow.Set(1, 0)
	})

	// persistent backing for the table
	var tableRows []cartRow
	var rowIDs []string // parallel slice to map rows back to cart IDs

	// Shipping form state
	type shipFormVals struct {
		Name, Address, City, State, Zip, Country, Phone string
		PriceStr                                        string
	}
	sf := &shipFormVals{Country: "US"}

	// helper: find current shipping line
	getShipping := func() (id string, item *CartItem, ok bool) {
		for sid, it := range cart.Items {
			if strings.HasPrefix(sid, shippingPrefix) {
				return sid, it, true
			}
		}
		return "", nil, false
	}

	// helper: open shipping dialog
	openShippingDialog := func(anchor core.Widget) {
		if sid, it, ok := getShipping(); ok {
			parts := strings.SplitN(sid, "|", 8)
			if len(parts) == 8 {
				sf.Name = parts[1]
				sf.Address = parts[2]
				sf.City = parts[3]
				sf.State = parts[4]
				sf.Zip = parts[5]
				sf.Country = parts[6]
				sf.Phone = parts[7]
			}
			if it != nil {
				c := priceToCents(it.Product.Price)
				sf.PriceStr = centsToMoney(c)[1:]
			}
		}

		d := core.NewBody("Shipping")
		d.Styler(func(s *styles.Style) {
			s.Gap.Set(units.Dp(10), units.Dp(8))
			s.Padding.Set(units.Dp(8), units.Dp(8))
		})
		core.NewText(d).SetType(core.TextSupporting).SetText("Enter your shipping details:")

		tfName := core.NewTextField(d).SetPlaceholder("Full name").SetText(sf.Name)
		tfAddr := core.NewTextField(d).SetPlaceholder("Address").SetText(sf.Address)
		tfCity := core.NewTextField(d).SetPlaceholder("City").SetText(sf.City)
		tfState := core.NewTextField(d).SetPlaceholder("State/Province").SetText(sf.State)
		tfZip := core.NewTextField(d).SetPlaceholder("ZIP/Postal").SetText(sf.Zip)
		tfCountry := core.NewTextField(d).SetPlaceholder("Country").SetText(sf.Country)
		tfPhone := core.NewTextField(d).SetPlaceholder("Phone").SetText(sf.Phone)
		tfPrice := core.NewTextField(d).SetPlaceholder("Shipping price (e.g. 7.50)").SetText(sf.PriceStr)

		d.AddBottomBar(func(bar *core.Frame) {
			d.AddCancel(bar)
			d.AddOK(bar).OnClick(func(e events.Event) {
				sf.Name = tfName.Text()
				sf.Address = tfAddr.Text()
				sf.City = tfCity.Text()
				sf.State = tfState.Text()
				sf.Zip = tfZip.Text()
				sf.Country = tfCountry.Text()
				sf.Phone = tfPhone.Text()
				sf.PriceStr = tfPrice.Text()

				// Strip pipe characters to prevent delimiter injection
				strip := func(s string) string { return strings.ReplaceAll(s, "|", "") }
				shipID := fmt.Sprintf("shipping-to|%s|%s|%s|%s|%s|%s|%s",
					strip(sf.Name), strip(sf.Address), strip(sf.City), strip(sf.State), strip(sf.Zip), strip(sf.Country), strip(sf.Phone),
				)
				v := sf.PriceStr
				if v == "" {
					v = "0"
				}
				cents := priceToCents("$" + v)
				cart.SetShipping(shipID, cents)
			})
		})
		d.RunDialog(anchor)
	}

	// Build details every time
	detailsRoot.Updater(func() {
		if !col.Open {
			return
		}
		detailsRoot.DeleteChildren()

		// Table rows from cart (excluding shipping)
		tableRows = tableRows[:0]
		rowIDs = rowIDs[:0]
		for id, it := range cart.Items {
			if strings.HasPrefix(id, shippingPrefix) {
				continue
			}
			unit := priceToCents(it.Product.Price)
			name := it.Product.Name
			if name == "" {
				name = id
			}
			tableRows = append(tableRows, cartRow{
				Item:      name,
				UnitPrice: centsToMoney(unit),
				Qty:       it.Qty,
				LineTotal: centsToMoney(unit * it.Qty),
				Remove:    false,
			})
			rowIDs = append(rowIDs, id)
		}

		// Items TABLE (full width)
		tbl := core.NewTable(detailsRoot).SetSlice(&tableRows)
		tbl.SetReadOnly(false)
		tbl.Styler(func(s *styles.Style) {
			s.Grow.Set(1, 0)
		})
		tbl.OnChange(func(e events.Event) {
			for i := range tableRows {
				if i >= len(rowIDs) {
					continue
				}
				id := rowIDs[i]
				row := tableRows[i]
				cart.mu.Lock()
				it := cart.Items[id]
				cart.mu.Unlock()
				if it == nil {
					continue
				}
				if row.Remove || row.Qty <= 0 {
					cart.Remove(id)
					continue
				}
				if row.Qty != it.Qty {
					diff := row.Qty - it.Qty
					if diff > 0 {
						for n := 0; n < diff; n++ {
							cart.Inc(id)
						}
					} else if diff < 0 {
						for n := 0; n < -diff; n++ {
							cart.Dec(id)
						}
					}
				}
			}
		})

		// Shipping block (below the table)
		shipBox := core.NewFrame(detailsRoot)
		shipBox.Styler(func(s *styles.Style) {
			s.Direction = styles.Column
			s.Gap.Set(units.Dp(6))
			s.Padding.Set(units.Dp(10))
			s.Border.Radius.Set(units.Dp(6))
			s.Border.Width.Set(units.Dp(1))
		})

		core.NewText(shipBox).SetType(core.TextTitleSmall).SetText("Shipping")

		if sid, it, ok := getShipping(); ok {
			parts := strings.SplitN(sid, "|", 8)
			if len(parts) == 8 {
				addr := fmt.Sprintf("%s\n%s\n%s, %s %s\n%s\nPhone: %s",
					parts[1], parts[2], parts[3], parts[4], parts[5], parts[6], parts[7])
				core.NewText(shipBox).SetText(addr).Styler(func(s *styles.Style) {
					s.Text.WhiteSpace = text.WhiteSpacePre
				})
			}
			amt := centsToMoney(priceToCents(it.Product.Price))
			core.NewText(shipBox).SetText("Cost: " + amt).Styler(func(s *styles.Style) {
				s.Font.Weight = rich.Bold
			})
		} else {
			core.NewText(shipBox).SetText("No shipping address set").Styler(func(s *styles.Style) {
				s.Color = colors.Scheme.OnSurfaceVariant
			})
		}

		shipBtns := core.NewFrame(shipBox)
		shipBtns.Styler(func(s *styles.Style) {
			s.Direction = styles.Row
			s.Gap.Set(units.Dp(8))
		})
		addUpd := core.NewButton(shipBtns)
		if cart.HasShipping() {
			addUpd.SetText("Edit Shipping").SetIcon(icons.Edit)
		} else {
			addUpd.SetText("Add Shipping").SetIcon(icons.Box)
		}
		addUpd.OnClick(func(e events.Event) { openShippingDialog(addUpd) })
		if cart.HasShipping() {
			core.NewButton(shipBtns).SetText("Remove").SetIcon(icons.Delete).
				OnClick(func(e events.Event) { cart.RemoveShipping() })
		}

		// Footer: totals breakdown + actions
		footer := core.NewFrame(detailsRoot)
		footer.Styler(func(s *styles.Style) {
			s.Direction = styles.Column
			s.Gap.Set(units.Dp(6))
			s.Padding.Set(units.Dp(10))
			s.Border.Width.Top = units.Dp(1)
			s.Border.Color.Top = colors.Scheme.OutlineVariant
		})

		// Subtotal and shipping breakdown
		subtotal := cart.TotalCents()
		var shipCost int
		if _, it, ok := getShipping(); ok {
			shipCost = priceToCents(it.Product.Price)
			subtotal -= shipCost
		}
		totalsRow := core.NewFrame(footer)
		totalsRow.Styler(func(s *styles.Style) {
			s.Direction = styles.Column
			s.Gap.Set(units.Dp(2))
		})
		core.NewText(totalsRow).SetText("Subtotal: " + centsToMoney(subtotal))
		if shipCost > 0 {
			core.NewText(totalsRow).SetText("Shipping: " + centsToMoney(shipCost))
		}
		core.NewText(totalsRow).SetText("Total: " + centsToMoney(cart.TotalCents())).Styler(func(s *styles.Style) {
			s.Font.Weight = rich.Bold
			s.Font.Size = units.Dp(18)
		})

		acts := core.NewFrame(footer)
		acts.Styler(func(s *styles.Style) {
			s.Direction = styles.Row
			s.Gap.Set(units.Dp(8))
		})
		core.NewButton(acts).SetText("Clear Cart").SetIcon(icons.Delete).SetType(core.ButtonOutlined).OnClick(func(e events.Event) {
			cart.Clear()
		})
		checkoutBtn := core.NewButton(acts).SetText("Checkout").SetIcon(icons.ShoppingCart).SetType(core.ButtonFilled)
		checkoutBtn.Updater(func() {
			enabled := cart.CountNonShipping() > 0 && cart.HasShipping()
			checkoutBtn.SetEnabled(enabled)
		})
		checkoutBtn.OnClick(func(e events.Event) {
			StartCheckout(cart, checkoutBtn)
		})
	})

	// Initial paint and reactive updates
	detailsRoot.OnShow(func(_ events.Event) { detailsRoot.Update() })
	cart.OnChange(func() {
		core.TheApp.RunOnMain(func() {
			detailsRoot.Update()
			summary.Update()
		})
	})
	cart.OnChange(func() { SaveCartState(cart) })

	// HTML custom element handler (only when using the content system)
	if ctx != nil {
		ctx.ElementHandlers["my-button"] = func(hctx *htmlcore.Context) bool {
			id := htmlcore.GetAttr(hctx.Node, "id")
			pr, ok := prodByID[id]
			if !ok {
				return false
			}
			parent := inlineParentNode(hctx)
			core.NewButton(parent).SetText("add to cart").SetIcon(icons.Add).OnClick(func(e events.Event) { cart.Add(pr) })
			return true
		}
	}
}
