// payment.go
package main

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"net/url"

	"cogentcore.org/core/core"
)

// Injected at build time, e.g.:
//   go build -ldflags "-X 'main.stripePK=pk_live_...'"
var stripePK string

// Base URL for your server API (adjust if different host/port/proto)
var serverBase = "https://" + siteName
func init() {
	serverBase = "https://" + siteName
}
// Toggle which flow to use:
//   true  -> Stripe Checkout Session
//   false -> Payment Intent + your hosted /pay page with Stripe Elements
const useCheckoutSession = false

// Public entry point called from the Checkout button in cart_ui.go
func CheckoutWithServer(cart *Cart) {
	if stripePK == "" {
		log.Println("Stripe public key (stripePK) is not set; pass via ldflags: -X 'main.stripePK=pk_live_...'\nCheckout aborted.")
		return
	}

	var err error
	if useCheckoutSession {
		err = checkoutViaSession(cart)
	} else {
		err = checkoutViaPaymentIntent(cart)
	}
	if err != nil {
		log.Println("Checkout error:", err)
	}
}

// ---------- Strategy A: Stripe Checkout Session ----------
func checkoutViaSession(cart *Cart) error {
	type req struct {
		Items []APIItem `json:"items"`
	}
	type resp struct {
		URL   string `json:"url"`
		Error string `json:"error,omitempty"`
		// (optionally the server may also return "id" for the session)
	}

	payload := req{Items: cart.ItemsForAPI()}
	body, _ := json.Marshal(payload)

	endpoint := serverBase + "/create-checkout-session"
	rp, err := http.Post(endpoint, "application/json", bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer rp.Body.Close()

	var out resp
	if err := json.NewDecoder(rp.Body).Decode(&out); err != nil {
		return err
	}
	if out.Error != "" {
		log.Println("server returned error:", out.Error)
		return nil
	}
	if out.URL == "" {
		log.Println("create-checkout-session returned empty url")
		return nil
	}

	openBrowser(out.URL)
	return nil
}

// ---------- Strategy B: Payment Intent + hosted Elements page ----------
func checkoutViaPaymentIntent(cart *Cart) error {
	type req struct {
		Items []APIItem `json:"items"`
	}
	type resp struct {
		ClientSecret string `json:"clientSecret"`
		Error        string `json:"error,omitempty"`
	}

	payload := req{Items: cart.ItemsForAPI()}
	body, _ := json.Marshal(payload)

	endpoint := serverBase + "/create-payment-intent"
	rp, err := http.Post(endpoint, "application/json", bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer rp.Body.Close()

	var out resp
	if err := json.NewDecoder(rp.Body).Decode(&out); err != nil {
		return err
	}
	if out.Error != "" {
		log.Println("server returned error:", out.Error)
		return nil
	}
	if out.ClientSecret == "" {
		log.Println("create-payment-intent returned empty clientSecret")
		return nil
	}

	// Open your hosted page that mounts Stripe Elements with the client secret.
	// If your page expects different param names, tweak here.
	payURL := serverBase + "/?" + url.Values{
		"client_secret": []string{out.ClientSecret},
		"pk":            []string{stripePK},
	}.Encode()

	openBrowser(payURL)
	return nil
}

// ---------- Open URL using Cogent Core ----------
func openBrowser(u string) {
	// Cogent Core handles native and WASM correctly:
	// - Native: opens system browser
	// - WASM: navigates the current window
	log.Println(u)
	core.TheApp.OpenURL(u)
}
