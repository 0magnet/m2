//Package main 🮕.go
package main

import (
	"fmt"
	"strconv"
	"sync"
	"strings"

	p "github.com/0magnet/m2/pkg/product"
)

type CartItem struct {
	Product p.Product `edit:"-"`
	Qty     int
	// For shipping line: use Product.Partno == "shipping-to|<...>" and Price in Product.Price
}

type Cart struct {
	mu       sync.Mutex
	Items    map[string]*CartItem // key: Partno
	onChange []func()
}

func NewCart() *Cart { return &Cart{Items: make(map[string]*CartItem)} }

func (c *Cart) OnChange(fn func()) { c.mu.Lock(); c.onChange = append(c.onChange, fn); c.mu.Unlock() }
func (c *Cart) notify()             { for _, fn := range c.onChange { fn() } }

// ---- Mutators (unlock BEFORE notify) ----
func (c *Cart) Add(prod p.Product) {
	c.mu.Lock()
	if it, ok := c.Items[prod.Partno]; ok {
		it.Qty++
	} else {
		c.Items[prod.Partno] = &CartItem{Product: prod, Qty: 1}
	}
	c.mu.Unlock()
	c.notify()
}
func (c *Cart) Inc(id string)   { c.mu.Lock(); if it, ok := c.Items[id]; ok { it.Qty++ }; c.mu.Unlock(); c.notify() }
func (c *Cart) Dec(id string)   { c.mu.Lock(); if it, ok := c.Items[id]; ok { it.Qty--; if it.Qty <= 0 { delete(c.Items, id) } }; c.mu.Unlock(); c.notify() }
func (c *Cart) Remove(id string){ c.mu.Lock(); delete(c.Items, id); c.mu.Unlock(); c.notify() }
func (c *Cart) Clear()          { c.mu.Lock(); clear(c.Items); c.mu.Unlock(); c.notify() }

// ---- Shipping helpers ----
const shippingPrefix = "shipping-to|"

func (c *Cart) SetShipping(shippingID string, amountCents int) {
	// shippingID should start with shippingPrefix and contain the piped fields
	if shippingID == "" {
		return
	}
	price := fmt.Sprintf("$%d.%02d", amountCents/100, amountCents%100)
	shipProd := p.Product{
		Partno: shippingID, // e.g., "shipping-to|Name|Addr|City|State|Zip|Country|Phone"
		Name:   "Shipping",
		Price:  price,
	}
	c.mu.Lock()
	// remove any existing shipping line
	for id := range c.Items {
		if len(id) >= len(shippingPrefix) && id[:len(shippingPrefix)] == shippingPrefix {
			delete(c.Items, id)
		}
	}
	// add/replace shipping as qty=1
	c.Items[shipProd.Partno] = &CartItem{Product: shipProd, Qty: 1}
	c.mu.Unlock()
	c.notify()
}

func (c *Cart) HasShipping() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	for id := range c.Items {
		if len(id) >= len(shippingPrefix) && id[:len(shippingPrefix)] == shippingPrefix {
			return true
		}
	}
	return false
}

func (c *Cart) CountNonShipping() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	n := 0
	for id, it := range c.Items {
		if len(id) >= len(shippingPrefix) && id[:len(shippingPrefix)] == shippingPrefix {
			continue
		}
		n += it.Qty
	}
	return n
}

func (c *Cart) RemoveShipping() {
	c.mu.Lock()
	for id := range c.Items {
		if strings.HasPrefix(id, shippingPrefix) {
			delete(c.Items, id)
			break
		}
	}
	c.mu.Unlock()
	c.notify()
}

// ---- Queries ----
func (c *Cart) Count() int {
	c.mu.Lock(); defer c.mu.Unlock()
	n := 0
	for _, it := range c.Items {
		n += it.Qty
	}
	return n
}
func (c *Cart) TotalCents() int {
	c.mu.Lock(); defer c.mu.Unlock()
	total := 0
	for _, it := range c.Items {
		total += priceToCents(it.Product.Price) * it.Qty
	}
	return total
}

// For server payload (mirrors your wasm code: "ID + ' X ' + qty")
type APIItem struct {
	ID     string `json:"id"`
	Amount int    `json:"amount"`
}

func (c *Cart) ItemsForAPI() []APIItem {
	c.mu.Lock()
	defer c.mu.Unlock()
	res := make([]APIItem, 0, len(c.Items))
	for id, it := range c.Items {
		res = append(res, APIItem{
			ID:     fmt.Sprintf("%s X %d", id, it.Qty),
			Amount: priceToCents(it.Product.Price) * it.Qty,
		})
	}
	return res
}

// ---- Money helpers ----
func priceToCents(s string) int {
	if s == "" {
		return 0
	}
	s = strings.TrimPrefix(s, "$")
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0
	}
	if f < 0 {
		return -int(-f*100 + 0.5)
	}
	return int(f*100 + 0.5)
}

func centsToMoney(c int) string {
	if c < 0 {
		return fmt.Sprintf("-$%d.%02d", -c/100, -c%100)
	}
	return fmt.Sprintf("$%d.%02d", c/100, c%100)
}
