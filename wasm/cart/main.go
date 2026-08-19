//go:build js && wasm

package main

import (
	"log"
	"syscall/js"
)

// set client pk on compile
var stripePK string

type item struct {
	ID     string `json:"id"`
	Amount int    `json:"amount"`
	Qty    int    `json:"quantity"`
}

var (
	wasmName string
	doc      = js.Global().Get("document")
	cart     []item
)

func main() {
	ready := make(chan struct{})

	document := js.Global().Get("document")
	readyState := document.Get("readyState").String()
	if readyState == "interactive" || readyState == "complete" {
		log.Println(wasmName+":", "WASM: DOM already fully loaded")
		close(ready)
	} else {
		cb := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			log.Println(wasmName+":", "WASM: DOM fully loaded and parsed")
			close(ready)
			return nil
		})
		defer cb.Release()

		document.Call("addEventListener", "DOMContentLoaded", cb)
		log.Println(wasmName+":", "WASM: waiting for DOM to load")
	}

	<-ready

	c := make(chan struct{})
	if stripePK == "" {
		log.Fatal("Stripe PK not found!")
	}
	window := js.Global().Get("window")
	location := window.Get("location")
	pathname := location.Get("pathname").String()

	switch pathname {
	case "/complete":
		completeLogic()
	default:
		defaultLogic()
	}
	<-c
}

func defaultLogic() {
	js.Global().Set("addToCart", js.FuncOf(addUnToCart))
	js.Global().Set("clearStorage", js.FuncOf(clearAll))
	js.Global().Set("emptyCart", js.FuncOf(emptyCart))
	js.Global().Set("updateItemQuantity", js.FuncOf(updateItemQuantity))
	js.Global().Set("removeFromCart", js.FuncOf(removeFromCart))
	js.Global().Set("addShippingInfo", js.FuncOf(addShippingInfo))
	js.Global().Set("goToCheckout", js.FuncOf(goToCheckout))
	js.Global().Set("cancelCheckout", js.FuncOf(cancelCheckout))
	js.Global().Set("callUpdateCartDisplay", js.FuncOf(updateCartDisplayWrapper))
	loadCart()
	updateCartDisplay()
}
