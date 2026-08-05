//go:build js

package main

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"
	"syscall/js"

	"cogentcore.org/core/core"
)

// StartCheckout (web): creates a plain <dialog> (verbatim to your old UI),
// loads Stripe.js, creates a Payment Intent, mounts Elements, confirms payment.
func StartCheckout(cart *Cart, _ core.Widget) {
	if stripePK == "" {
		log.Println("stripePK is empty; provide with -ldflags \"-X 'main.stripePK=pk_...'\"")
		return
	}
	go startStripeFlow(cart)
}

func startStripeFlow(cart *Cart) {
	if err := ensureStripe(); err != nil {
		showMessage("Stripe init failed.")
		log.Println("ensureStripe:", err)
		return
	}
	dlg := ensureDialog()
	hideSpinner()
	hideMessage()
	dlg.Call("showModal")

	clientSecret, err := createPaymentIntent(buildCheckoutPayload(cart))
	if err != nil || clientSecret == "" {
		if err != nil {
			log.Println("createPaymentIntent:", err)
		}
		showMessage("Failed to create payment intent.")
		return
	}
	if err := setupStripeElements(clientSecret); err != nil {
		showMessage("Failed to initialize payment form.")
		log.Println("setupStripeElements:", err)
		return
	}
	wireSubmit(clientSecret)
	wireCancel()
}

// ---------- Stripe + Elements ----------

var (
	stripeValue js.Value
	stripe      js.Value
	elements    js.Value
)

func ensureStripe() error {
	// If Stripe is already present, reuse it.
	stripeValue = js.Global().Get("Stripe")
	if !stripeValue.Truthy() {
		// Load script
		head := queryOne("head")
		if !head.Truthy() {
			return errStr("no <head/> for script injection")
		}
		script := createEl("script")
		script.Set("src", "https://js.stripe.com/v3/")
		script.Set("defer", true)

		done := make(chan struct{})
		onload := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			done <- struct{}{}
			return nil
		})
		script.Call("addEventListener", "load", onload)
		head.Call("appendChild", script)
		<-done
		onload.Release()

		stripeValue = js.Global().Get("Stripe")
		if !stripeValue.Truthy() {
			return errStr("Stripe global missing after load")
		}
	}
	if !stripe.Truthy() {
		stripe = stripeValue.Invoke(stripePK)
		if !stripe.Truthy() {
			return errStr("Stripe(pk) init failed")
		}
	}
	return nil
}

func setupStripeElements(clientSecret string) error {
	elements = stripe.Call("elements", map[string]any{
		"clientSecret": clientSecret,
	})
	if !elements.Truthy() {
		return errStr("elements() failed")
	}
	payment := elements.Call("create", "payment", map[string]any{"layout": "tabs"})
	if !payment.Truthy() {
		return errStr("create('payment') failed")
	}
	host := getEl("payment-element")
	if !host.Truthy() {
		return errStr("#payment-element not found")
	}
	host.Set("innerHTML", "")
	payment.Call("mount", "#payment-element")
	return nil
}

func wireSubmit(clientSecret string) {
	submit := getEl("submit")
	if !submit.Truthy() {
		return
	}
	cl := submit.Call("cloneNode", true)
	submit.Get("parentNode").Call("replaceChild", cl, submit)
	cl.Call("addEventListener", "click", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		args[0].Call("preventDefault")
		showSpinner(true)
		confirmPayment(clientSecret)
		return nil
	}))
}

func confirmPayment(clientSecret string) {
	loc := js.Global().Get("window").Get("location")
	origin := loc.Get("origin").String()
	returnURL := origin + "/complete"

	stripe.Call("confirmPayment", map[string]any{
		"elements": elements,
		"confirmParams": map[string]any{
			"return_url": returnURL,
		},
	}).Call("then", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		hideSpinner()
		res := args[0]
		err := res.Get("error")
		if err.Truthy() {
			msg := err.Get("message").String()
			if msg == "" {
				msg = "Payment failed."
			}
			showMessage(msg)
		} else {
			showMessage("Payment submitted. Redirecting…")
		}
		return nil
	}))
}

// ---------- Server call ----------

func createPaymentIntent(payload []byte) (string, error) {
	type resp struct {
		ClientSecret string `json:"clientSecret"`
		Error        string `json:"error,omitempty"`
	}
	rp, err := http.Post(serverBase+"/create-payment-intent", "application/json", bytes.NewReader(payload))
	if err != nil {
		return "", err
	}
	defer rp.Body.Close()
	if rp.StatusCode != http.StatusOK {
		return "", errStr("server returned " + rp.Status)
	}

	var out resp
	if err := json.NewDecoder(rp.Body).Decode(&out); err != nil {
		return "", err
	}
	if out.Error != "" {
		log.Println("server returned error:", out.Error)
	}
	return out.ClientSecret, nil
}

func buildCheckoutPayload(cart *Cart) []byte {
	type cItem struct {
		ID     string `json:"id"`
		Amount int    `json:"amount"`
	}
	type payload struct {
		Items []cItem `json:"items"`
	}
	var items []cItem
	cart.mu.Lock()
	for id, it := range cart.Items {
		if it.Qty <= 0 {
			continue
		}
		amt := priceToCents(it.Product.Price) * it.Qty
		label := id
		if !strings.HasPrefix(id, "shipping-to|") {
			label = id + " X " + strconv.Itoa(it.Qty)
		}
		items = append(items, cItem{ID: label, Amount: amt})
	}
	cart.mu.Unlock()
	b, err := json.Marshal(payload{Items: items})
	if err != nil {
		log.Printf("buildCheckoutPayload: marshal error: %v", err)
		return nil
	}
	return b
}

// ---------- Dialog (verbatim to your old UI) ----------

func ensureDialog() js.Value {
	doc := js.Global().Get("document")
	if dlg := doc.Call("getElementById", "stripecheckout"); dlg.Truthy() {
		return dlg
	}
	dlg := doc.Call("createElement", "dialog")
	dlg.Set("id", "stripecheckout")
	dlg.Set("innerHTML", `
<button id="close-dialog-button" onclick="cancelCheckout()" class="cancel-button">×</button>
<div class="checkout-container" id="checkout-container">
  <form id="payment-form" class="payment-form">
    <div id="payment-element"></div>
    <button id="submit" type="button">
      <div class="spinner hidden" id="spinner"></div>
      <span id="button-text">Pay now</span>
    </button>
    <div id="payment-message" class="hidden"></div>
  </form>
</div>`)
	doc.Get("body").Call("appendChild", dlg)
	return dlg
}

var cancelBound bool

func wireCancel() {
	if cancelBound {
		return
	}
	cancelBound = true
	js.Global().Set("cancelCheckout", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		getEl("stripecheckout").Call("close")
		return nil
	}))
}

// ---------- Tiny DOM helpers & UI bits ----------

func getEl(id string) js.Value {
	return js.Global().Get("document").Call("getElementById", id)
}
func queryOne(sel string) js.Value {
	return js.Global().Get("document").Call("querySelector", sel)
}
func createEl(tag string) js.Value {
	return js.Global().Get("document").Call("createElement", tag)
}

func showMessage(msg string) {
	m := getEl("payment-message")
	if !m.Truthy() {
		return
	}
	m.Set("textContent", msg)
	m.Get("classList").Call("remove", "hidden")
}
func hideMessage() {
	m := getEl("payment-message")
	if m.Truthy() {
		m.Get("classList").Call("add", "hidden")
		m.Set("textContent", "")
	}
}
func showSpinner(on bool) {
	sp := getEl("spinner")
	bt := getEl("button-text")
	if !sp.Truthy() || !bt.Truthy() {
		return
	}
	if on {
		sp.Get("classList").Call("remove", "hidden")
		bt.Get("classList").Call("add", "hidden")
	} else {
		sp.Get("classList").Call("add", "hidden")
		bt.Get("classList").Call("remove", "hidden")
	}
}
func hideSpinner() { showSpinner(false) }

type strErr string
func (e strErr) Error() string { return string(e) }
func errStr(s string) error    { return strErr(s) }
