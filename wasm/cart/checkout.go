//go:build js && wasm

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"syscall/js"
)

func addShippingInfo(this js.Value, args []js.Value) interface{} {
	event := args[0]
	form := args[1]
	event.Call("preventDefault")
	getFormValue := func(name string) string {
		return form.Call("querySelector", fmt.Sprintf("[name='%s']", name)).Get("value").String()
	}
	shippingInfo := fmt.Sprintf("shipping-to|%s|%s|%s|%s|%s|%s|%s",
		getFormValue("shipping-name"),
		getFormValue("shipping-address"),
		getFormValue("shipping-city"),
		getFormValue("shipping-state"),
		getFormValue("shipping-zip"),
		getFormValue("shipping-country"),
		getFormValue("shipping-phone"),
	)
	priceStr := getFormValue("shipping-price")
	price, err := strconv.ParseFloat(priceStr, 64)
	if err != nil {
		log.Println(wasmName+":", "Error: Failed to parse shipping price")
		price = 0.0
	}

	addToCart(js.Value{}, []js.Value{
		js.ValueOf(shippingInfo),
		js.ValueOf(int(price * 100)),
		js.ValueOf(1),
	})
	return false
}

var (
	elements       js.Value
	stripeValue    js.Value
	stripe         js.Value
	checkoutStripe = doc.Call("getElementById", "stripecheckout")
)

func goToCheckout(this js.Value, args []js.Value) any {
	if stripeValue.IsUndefined() {
		log.Println(`js.Global().Get("Stripe")`)
		stripeValue = js.Global().Get("Stripe")
		if stripeValue.IsUndefined() {
			log.Println(`Stripe is undefined, attempting to load Stripe.js`)

			doc := js.Global().Get("document")
			head := doc.Call("querySelector", "head")
			script := doc.Call("createElement", "script")
			script.Set("src", "https://js.stripe.com/v3/")
			script.Set("defer", true)

			done := make(chan bool)
			script.Call("addEventListener", "load", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
				log.Println(wasmName+":", "Stripe.js script has been loaded")
				done <- true
				return nil
			}))

			head.Call("appendChild", script)

			<-done

			stripeValue = js.Global().Get("Stripe")
			if stripeValue.IsUndefined() {
				log.Println(wasmName+":", "Failed to load Stripe.js")
				return nil
			}
		}
	}

	log.Println(wasmName+":", "Stripe.js loaded successfully")

	if stripe.IsUndefined() {
		log.Println(wasmName+":", "Invoking Stripe")
		stripe = stripeValue.Invoke(stripePK)
		if stripe.IsUndefined() {
			log.Println(wasmName+":", "Failed to invoke Stripe")
			return nil
		}
	}

	log.Println(wasmName+":", "Stripe initialized")
	checkoutStripe = doc.Call("getElementById", "stripecheckout")
	if checkoutStripe.IsUndefined() {
		log.Println(wasmName+":", "element with ID stripecheckout not found")
	}

	checkoutStripe.Call("showModal")
	log.Println(wasmName+":", "initializePayment()")
	initializePayment()

	return nil
}

func cancelCheckout(this js.Value, args []js.Value) any {
	log.Println(wasmName+":", "Canceling checkout ; closing dialog")
	checkoutStripe.Call("close")
	updateCartDisplay()
	return nil
}

func initializePayment() {
	type cItem struct {
		ID     string `json:"id"`
		Amount int    `json:"amount"`
	}
	type checkout struct {
		Items []cItem `json:"items"`
	}
	payload := checkout{
		Items: func() []cItem {
			var items []cItem
			for _, it := range cart {
				items = append(items, cItem{ID: it.ID + " X " + strconv.Itoa(it.Qty), Amount: it.Amount})
			}
			return items
		}(),
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		log.Println(wasmName+":", "Error marshaling JSON:", err)
		return
	}
	fetchInit := map[string]interface{}{
		"method": "POST",
		"headers": map[string]interface{}{
			"Content-Type": "application/json",
		},
		"body": string(payloadJSON),
	}

	log.Println(wasmName+":", "fetch  /create-payment-intent")
	js.Global().Call("fetch", "/create-payment-intent", js.ValueOf(fetchInit)).
		Call("then", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			response := args[0]
			log.Println(wasmName+":", "got response from fetch /create-payment-intent")
			if !response.Get("ok").Bool() {
				log.Println(wasmName+":", "Fetch request failed with status:", response.Get("status").Int())
				showMessage("Failed to create payment intent: " + response.Get("status").String())
				return nil
			}
			response.Call("json").Call("then", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
				clientSecret := args[0].Get("clientSecret").String()
				log.Println(wasmName+":", "Client secret received:", clientSecret)
				setupStripeElements(clientSecret)
				return nil
			})).Call("catch", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
				log.Println(wasmName+":", "Error parsing JSON response:", args[0])
				showMessage("Failed to parse payment intent response.")
				return nil
			}))
			return nil
		})).
		Call("catch", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			log.Println(wasmName+":", "Error in fetch request:", args[0])
			showMessage("Failed to communicate with the server.")
			return nil
		}))
}

func setupStripeElements(clientSecret string) {
	elements = stripe.Call("elements", map[string]interface{}{
		"clientSecret": clientSecret,
	})
	if elements.IsUndefined() {
		log.Println(wasmName+":", "Failed to initialize Stripe Elements")
		showMessage("Failed to initialize payment elements.")
		return
	}
	paymentElement := elements.Call("create", "payment", map[string]interface{}{
		"layout": "tabs",
	})
	if paymentElement.IsUndefined() {
		log.Println(wasmName+":", "Failed to create payment element")
		showMessage("Failed to create payment element.")
		return
	}
	paymentElement.Call("mount", "#payment-element")
	submitButton := doc.Call("getElementById", "submit")
	submitButton.Call("addEventListener", "click", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		args[0].Call("preventDefault")
		showSpinner(true)
		confirmPayment(clientSecret)
		return nil
	}))
}

func confirmPayment(clientSecret string) {

	windowLocation := js.Global().Get("window").Get("location")
	protocol := windowLocation.Get("protocol").String()
	hostname := windowLocation.Get("hostname").String()
	port := windowLocation.Get("port").String()

	baseURL := protocol + "//" + hostname
	if port != "" {
		baseURL += ":" + port
	}
	//	path := windowLocation.Get("pathname").String()
	//    baseURL += strings.Split(path, "?")[0]
	//    log.Println(wasmName+":","return url ", baseURL)

	returnURL := baseURL + "/complete"
	returnURL += "?payment_intent=" + clientSecret // + "#complete"
	log.Println(wasmName+":", "Return URL for payment:", returnURL)

	stripe.Call("confirmPayment", map[string]interface{}{
		"elements": elements,
		"confirmParams": map[string]interface{}{
			"return_url": returnURL,
		},
	}).Call("then", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		result := args[0]
		if result.Get("error").IsUndefined() {
			log.Println(wasmName+":", "Payment successful:", result)
			showMessage("Payment successful! Thank you for your order.")
		} else {
			log.Println(wasmName+":", "Payment error:", result.Get("error").Get("message").String())
			showMessage("Payment failed: " + result.Get("error").Get("message").String())
		}

		showSpinner(false)
		return nil
	}))
}

func showMessage(message string) {
	messageElement := doc.Call("getElementById", "payment-message")
	messageElement.Set("innerText", message)
	messageElement.Set("className", "")
}

func showSpinner(isLoading bool) {
	spinner := doc.Call("getElementById", "spinner")
	buttonText := doc.Call("getElementById", "button-text")

	if isLoading {
		spinner.Set("className", "")
		buttonText.Set("className", "hidden")
	} else {
		spinner.Set("className", "hidden")
		buttonText.Set("className", "")
	}
}
