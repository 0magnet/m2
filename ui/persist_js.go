//go:build js

package main

import (
	"encoding/json"
	"strconv"
	"strings"
	"syscall/js"

	p "github.com/0magnet/m2/pkg/product"
)

const cartStorageKey = "cartItems" // must match server-side expectation

// server-compatible storage shape
type legacyItem struct {
	ID       string `json:"id"`       // product part number or special shipping id
	Amount   int    `json:"amount"`   // per-unit amount in cents
	Quantity int    `json:"quantity"` // quantity
}

// LoadCartState restores the cart from window.localStorage using the server format:
// [
//   {"id": "rect-VS-43CTQ", "amount": 104, "quantity": 2},
//   {"id": "shipping-to|...","amount": 700, "quantity": 1}
// ]
func LoadCartState() (map[string]*CartItem, bool) {
	ls := js.Global().Get("localStorage")
	if ls.IsUndefined() || ls.IsNull() {
		return nil, false
	}

	raw := ls.Call("getItem", cartStorageKey)
	if raw.IsUndefined() || raw.IsNull() {
		return nil, false
	}

	var arr []legacyItem
	if err := json.Unmarshal([]byte(raw.String()), &arr); err != nil {
		return nil, false
	}

	items := make(map[string]*CartItem, len(arr))
	for _, it := range arr {
		// Best-effort name: shipping entries get "Shipping", others default to empty (UI can look up)
		name := ""
		if strings.HasPrefix(it.ID, "shipping-") || strings.HasPrefix(it.ID, "shipping|") || strings.HasPrefix(it.ID, "shipping-to|") {
			name = "Shipping"
		}

		items[it.ID] = &CartItem{
			Product: p.Product{
				Partno: it.ID,
				Name:   name,
				// Store a display string (e.g., "$7.00") computed from cents for UI convenience
				Price: centsToDollarString(it.Amount),
			},
			Qty: it.Quantity,
		}
	}

	return items, true
}

// SaveCartState persists the cart to window.localStorage in the server format noted above.
func SaveCartState(cart *Cart) {
	ls := js.Global().Get("localStorage")
	if ls.IsUndefined() || ls.IsNull() {
		return
	}

	var out []legacyItem

	cart.mu.Lock()
	for _, it := range cart.Items {
		out = append(out, legacyItem{
			ID:       it.Product.Partno,
			Amount:   dollarsStringToCents(it.Product.Price),
			Quantity: it.Qty,
		})
	}
	cart.mu.Unlock()

	b, _ := json.Marshal(out)
	ls.Call("setItem", cartStorageKey, string(b))
}

// dollarsStringToCents converts price strings to integer cents.
// Accepts formats like "$7.50", "7.50", "7", "0.1" (=> 10 cents), and trims whitespace.
// Invalid/empty input returns 0.
func dollarsStringToCents(s string) int {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	if strings.HasPrefix(s, "$") {
		s = strings.TrimSpace(s[1:])
	}
	// Normalize to exactly two decimal places without using float math
	// Split on ".", pad/truncate fraction to 2 digits.
	parts := strings.SplitN(s, ".", 3)
	intPart := parts[0]
	frac := "00"
	if len(parts) >= 2 {
		frac = parts[1]
	}
	if len(frac) == 0 {
		frac = "00"
	} else if len(frac) == 1 {
		frac = frac + "0"
	} else if len(frac) > 2 {
		frac = frac[:2] // truncate extra precision
	}

	// Remove any stray non-digits from intPart (e.g., commas)
	intPart = strings.ReplaceAll(intPart, ",", "")

	i, err1 := strconv.Atoi(intPart)
	f, err2 := strconv.Atoi(frac)
	if err1 != nil || err2 != nil || i < 0 || f < 0 {
		return 0
	}
	return i*100 + f
}

// centsToDollarString renders cents as a display string like "$7.00"
func centsToDollarString(cents int) string {
	if cents < 0 {
		cents = 0
	}
	dollars := cents / 100
	remainder := cents % 100
	return "$" + strconv.Itoa(dollars) + "." + twoDigits(remainder)
}

func twoDigits(n int) string {
	if n < 10 {
		return "0" + strconv.Itoa(n)
	}
	return strconv.Itoa(n)
}
