//go:build js && wasm

// Package main complete.go
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"
	"syscall/js"
)

func updateCartDisplayWrapper(this js.Value, args []js.Value) interface{} {
	updateCartDisplay()
	return nil
}

func saveCart() {
	cartJSON, err := json.Marshal(cart)
	if err != nil {
		log.Println(wasmName+":", "Error saving cart:", err)
		return
	}
	js.Global().Get("localStorage").Call("setItem", "cartItems", string(cartJSON))
	updateCartDisplay()
}

// addToCart is called from Go rather than registered with js.FuncOf, so it
// keeps the callback shape its callers build but returns nothing.
//
// The guard reads three arguments, not two: qty is args[2]. Checking for two
// and then indexing the third is an out-of-range panic for any caller that
// passes exactly two, which is what the check was there to prevent.
func addToCart(_ js.Value, args []js.Value) {
	if len(args) < 3 {
		log.Println(wasmName+":", "addToCart: missing arguments")
		return
	}
	var cartItem item
	index := -1
	id := args[0].String()
	qty := args[2].Int()
	if qty == 0 {
		qty = 1
	}
	amount := int(args[1].Float()) * qty
	for i := range cart {
		if strings.Split(cart[i].ID, "|")[0] == strings.Split(id, "|")[0] {
			index = i
		}
	}
	if index > -1 {
		// update shipping
		if strings.Split(cart[index].ID, "|")[0] == "shipping-to" {
			cart[index].ID = id
			cart[index].Qty = 1
			cart[index].Amount = amount
		} else {
			cart[index].Qty = cart[index].Qty + qty
			cart[index].Amount = cart[index].Amount + amount
		}
	} else {
		cartItem = item{
			ID:     id,
			Amount: amount,
			Qty:    qty,
		}
		cart = append(cart, cartItem)
	}
	saveCart()
}

func addUnToCart(this js.Value, args []js.Value) interface{} {
	if len(args) < 2 {
		return "Error: Missing arguments"
	}
	id := args[0].String()
	price := args[1].Float()
	quantityInput := doc.Call("getElementById", fmt.Sprintf("qty-%s", id))
	if !quantityInput.Truthy() {
		log.Println(wasmName+":", "Error: Quantity input not found for item", id)
		return nil
	}
	quantity, err := strconv.Atoi(quantityInput.Get("value").String())
	if err != nil || quantity < 1 {
		quantity = 1
	}

	addToCart(js.Value{}, []js.Value{
		js.ValueOf(id),
		js.ValueOf(int(price * 100)),
		js.ValueOf(quantity),
	})
	return nil
}

func removeFromCart(this js.Value, inputs []js.Value) interface{} {
	id := inputs[0].String()
	newCart := []item{}
	for _, m := range cart {
		if m.ID != id {
			newCart = append(newCart, m)
		}
	}
	cart = newCart
	saveCart()
	return nil
}

func loadCart() {
	storedCart := js.Global().Get("localStorage").Call("getItem", "cartItems")
	if !storedCart.IsUndefined() && !storedCart.IsNull() {
		err := json.Unmarshal([]byte(storedCart.String()), &cart)
		if err != nil {
			log.Println(`can't unmarshal cart from local storage`)
			cart = []item{}
		}
	}
}

func emptyCart(this js.Value, inputs []js.Value) interface{} {
	js.Global().Get("localStorage").Call("removeItem", "cartItems")
	cart = []item{}
	updateCartDisplay()
	return nil
}

func clearAll(this js.Value, inputs []js.Value) interface{} {
	js.Global().Get("localStorage").Call("clear")
	cart = []item{}
	updateCartDisplay()
	return nil
}

func updateCartDisplay() {
	cartContainer := doc.Call("getElementById", "cart-items")
	totalPriceElement := doc.Call("getElementById", "total-price")
	table := cartContainer.Call("querySelector", "table")
	if table.IsNull() {
		table = doc.Call("createElement", "table")
		thead := doc.Call("createElement", "thead")
		thead.Set("innerHTML", `<tr><th>Item</th><th>Price</th><th>Quantity</th><th>Actions</th></tr>`)
		table.Call("appendChild", thead)
		tbody := doc.Call("createElement", "tbody")
		tbody.Set("id", "cart-tbody")
		table.Call("appendChild", tbody)
		cartContainer.Call("appendChild", table)
	}
	tbody := doc.Call("getElementById", "cart-tbody")
	tbody.Set("innerHTML", "")

	total := 0
	hasShipping := false
	for _, m := range cart {
		total += m.Amount
		row := doc.Call("createElement", "tr")

		row.Set("innerHTML", fmt.Sprintf(`<td>%s</td><td>$%.2f</td><td>%s</td><td><button onclick='removeFromCart("%s")'>Remove</button></td>`,
			func() string {
				parts := strings.Split(m.ID, "|")
				if len(parts) < 8 {
					return m.ID
				}
				hasShipping = true
				return fmt.Sprintf("%s:<br>%s<br>%s<br>%s, %s %s<br>%s<br>%s", parts[0], parts[1], parts[2], parts[3], parts[4], parts[5], parts[6], parts[7])
			}(),
			float64(m.Amount)/100,
			func() string {
				if len(strings.Split(m.ID, "|")) == 8 {
					return ""
				}
				return fmt.Sprintf(`<input type='number' value='%d' min='1' onchange='updateItemQuantity("%s", this.value)'>`, m.Qty, m.ID)
			}(),
			m.ID,
		))
		tbody.Call("appendChild", row)
	}
	totalPriceElement.Set("textContent", fmt.Sprintf("Total: $%.2f", float64(total)/100))

	checkoutbutton := doc.Call("getElementById", "checkout-button")
	if !checkoutbutton.Truthy() {
		return
	}

	if len(cart) > 1 && hasShipping {
		checkoutbutton.Call("removeAttribute", "disabled")
	} else {
		checkoutbutton.Call("setAttribute", "disabled", "true")
	}
}

func updateItemQuantity(this js.Value, args []js.Value) interface{} {
	id := args[0].String()
	qty, err := strconv.Atoi(args[1].String())
	if err != nil {
		log.Println(err)
	}
	for i := range cart {
		if cart[i].ID == id {
			unitPrice := cart[i].Amount / cart[i].Qty
			cart[i].Qty = qty
			cart[i].Amount = unitPrice * qty
			break
		}
	}
	saveCart()
	return nil
}
