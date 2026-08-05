// Package main complete.go
package main

import (
	"encoding/json"
	"log"
	"syscall/js"
)

func completeLogic() {
	initializeStripe()
}

func initializeStripe() {
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
				return
			}
		}
	}

	log.Println(wasmName+":", "Stripe.js loaded successfully")

	if stripe.IsUndefined() {
		log.Println(wasmName+":", "Invoking Stripe")
		stripe = stripeValue.Invoke(stripePK)
		if stripe.IsUndefined() {
			log.Println(wasmName+":", "Failed to invoke Stripe")
			return
		}
	}

	log.Println(wasmName+":", "Stripe initialized")
	checkStatus()
}

var (
	successIcon = `<svg width="16" height="14" viewBox="0 0 16 14" fill="none" xmlns="http://www.w3.org/2000/svg">
		<path fill-rule="evenodd" clip-rule="evenodd" d="M15.4695 0.232963C15.8241 0.561287 15.8454 1.1149 15.5171 1.46949L6.14206 11.5945C5.97228 11.7778 5.73221 11.8799 5.48237 11.8748C5.23253 11.8698 4.99677 11.7582 4.83452 11.5681L0.459523 6.44311C0.145767 6.07557 0.18937 5.52327 0.556912 5.20951C0.924454 4.89575 1.47676 4.93936 1.79051 5.3069L5.52658 9.68343L14.233 0.280522C14.5613 -0.0740672 15.1149 -0.0953599 15.4695 0.232963Z" fill="white"/>
	</svg>`

	errorIcon = `<svg width="16" height="16" viewBox="0 0 16 16" fill="none" xmlns="http://www.w3.org/2000/svg">
		<path fill-rule="evenodd" clip-rule="evenodd" d="M1.25628 1.25628C1.59799 0.914573 2.15201 0.914573 2.49372 1.25628L8 6.76256L13.5063 1.25628C13.848 0.914573 14.402 0.914573 14.7437 1.25628C15.0854 1.59799 15.0854 2.15201 14.7437 2.49372L9.23744 8L14.7437 13.5063C15.0854 13.848 15.0854 14.402 14.7437 14.7437C14.402 15.0854 13.848 15.0854 13.5063 14.7437L8 9.23744L2.49372 14.7437C2.15201 15.0854 1.59799 15.0854 1.25628 14.7437C0.914573 14.402 0.914573 13.848 1.25628 13.5063L6.76256 8L1.25628 2.49372C0.914573 2.15201 0.914573 1.59799 1.25628 1.25628Z" fill="white"/>
	</svg>`

	infoIcon = `<svg width="14" height="14" viewBox="0 0 14 14" fill="none" xmlns="http://www.w3.org/2000/svg">
		<path fill-rule="evenodd" clip-rule="evenodd" d="M10 1.5H4C2.61929 1.5 1.5 2.61929 1.5 4V10C1.5 11.3807 2.61929 12.5 4 12.5H10C11.3807 12.5 12.5 11.3807 12.5 10V4C12.5 2.61929 11.3807 1.5 10 1.5ZM4 0C1.79086 0 0 1.79086 0 4V10C0 12.2091 1.79086 14 4 14H10C12.2091 14 14 12.2091 14 10V4C14 1.79086 12.2091 0 10 0H4Z" fill="white"/>
		<path fill-rule="evenodd" clip-rule="evenodd" d="M5.25 7C5.25 6.58579 5.58579 6.25 6 6.25H7.25C7.66421 6.25 8 6.58579 8 7V10.5C8 10.9142 7.66421 11.25 7.25 11.25C6.83579 11.25 6.5 10.9142 6.5 10.5V7.75H6C5.58579 7.75 5.25 7.41421 5.25 7Z" fill="white"/>
		<path d="M5.75 4C5.75 3.31075 6.31075 2.75 7 2.75C7.68925 2.75 8.25 3.31075 8.25 4C8.25 4.68925 7.68925 5.25 7 5.25C6.31075 5.25 5.75 4.68925 5.75 4Z" fill="white"/>
	</svg>`
)

func setErrorState() {
	js.Global().Get("document").Call("querySelector", "#status-icon").Set("style", map[string]interface{}{"backgroundColor": "#DF1B41"})
	js.Global().Get("document").Call("querySelector", "#status-icon").Set("innerHTML", errorIcon)
	js.Global().Get("document").Call("querySelector", "#status-text").Set("textContent", "Something went wrong, please try again.")
	js.Global().Get("document").Call("querySelector", "#details-table").Call("classList").Call("add", "hidden")
	js.Global().Get("document").Call("querySelector", "#view-details").Call("classList").Call("add", "hidden")
}

func checkStatus() {
	clientSecret := js.Global().Get("URLSearchParams").New(js.Global().Get("window").Get("location").Get("search")).Call("get", "payment_intent_client_secret").String()

	if clientSecret == "" {
		setErrorState()
		return
	}

	if stripe.IsUndefined() {
		log.Println(wasmName+":", "Stripe is not initialized")
		setErrorState()
		return
	}

	stripe.Call("retrievePaymentIntent", clientSecret).Call("then", js.FuncOf(func(this js.Value, p []js.Value) interface{} {
		paymentIntent := p[0].Get("paymentIntent")
		setPaymentDetails(paymentIntent)
		return nil
	})).Call("catch", js.FuncOf(func(this js.Value, p []js.Value) interface{} {
		setErrorState()
		return nil
	}))
}

func getAllLocalStorageData() map[string]interface{} {
	localStorage := js.Global().Get("localStorage")
	keys := js.Global().Get("Object").Call("keys", localStorage)
	data := make(map[string]interface{})

	for i := 0; i < keys.Length(); i++ {
		key := keys.Index(i).String()
		value := localStorage.Call("getItem", key).String()
		var parsedValue interface{}
		err := json.Unmarshal([]byte(value), &parsedValue)
		if err != nil {
			parsedValue = value // If not JSON, store raw value
		}
		data[key] = parsedValue
	}
	return data
}

func submitOrder(localStorageData map[string]interface{}, paymentIntentId string) {
	orderData := map[string]interface{}{
		"localStorageData": localStorageData,
		"paymentIntentId":  paymentIntentId,
	}

	body, err := json.Marshal(orderData)
	if err != nil {
		log.Println(wasmName+":", "Error marshalling order data:", err)
		return
	}

	fetch := js.Global().Get("fetch")
	if fetch.IsUndefined() {
		log.Println(wasmName+":", "Fetch API is not available")
		return
	}

	options := map[string]interface{}{
		"method": "POST",
		"headers": map[string]interface{}{
			"Content-Type": "application/json",
		},
		"body": string(body),
	}

	fetch.Invoke("/submit-order", js.ValueOf(options)).Call("then", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
	    response := args[0]
	    if !response.Get("ok").Bool() {
	        response.Call("text").Call("then", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
	            errText := args[0].String()
	            log.Println(wasmName+":", "Order submit failed:", errText)
	            js.Global().Call("alert", "Order submission failed:\n"+errText)
	            return nil
	        }))
	        return nil
	    }
	    response.Call("json").Call("then", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
	        data := args[0]
	        log.Println(wasmName+":", "Order submitted successfully:", data)
	        return nil
	    }))
	    return nil
	}))
}

func setPaymentDetails(intent js.Value) {
	var statusText, iconColor, icon string
	statusText = "Something went wrong, please try again."
	iconColor = "#DF1B41"
	icon = errorIcon

	if !intent.IsUndefined() {
		intentStatus := intent.Get("status").String()
		intentID := intent.Get("id").String()

		allLocalStorageData := getAllLocalStorageData()

		switch intentStatus {
		case "succeeded":
			statusText = "Payment succeeded"
			iconColor = "#30B130"
			icon = successIcon
			if len(allLocalStorageData) > 0 {
				submitOrder(allLocalStorageData, intentID)
			} else {
				log.Println(wasmName+":", "No data found in localStorage; order not submitted.")
			}
		case "processing":
			statusText = "Your payment is processing."
			iconColor = "#6D6E78"
			icon = infoIcon
			if len(allLocalStorageData) > 0 {
				submitOrder(allLocalStorageData, intentID)
			} else {
				log.Println(wasmName+":", "No data found in localStorage; order not submitted.")
			}
		case "requires_payment_method":
			statusText = "Your payment was not successful, please try again."
		default:
			statusText = "Unknown payment status."
		}

		// Update the status icon, text, and links
		js.Global().Get("document").Call("querySelector", "#status-icon").Set("style", map[string]interface{}{"backgroundColor": iconColor})
		js.Global().Get("document").Call("querySelector", "#status-icon").Set("innerHTML", icon)
		js.Global().Get("document").Call("querySelector", "#status-text").Set("textContent", statusText)
		js.Global().Get("document").Call("querySelector", "#intent-id").Set("textContent", intentID)
		js.Global().Get("document").Call("querySelector", "#intent-status").Set("textContent", intentStatus)
		js.Global().Get("document").Call("querySelector", "#view-details").Set("href", "https://dashboard.stripe.com/payments/"+intentID)

		// Update the "Order Details" link with the paymentIntent ID
		orderDetailsLink := js.Global().Get("document").Call("querySelector", "#order-details-link")
		orderDetailsLink.Set("href", "/order/"+intentID)
		orderDetailsLink.Set("onclick", nil) // Allow default behavior (navigation)

	} else {
		setErrorState()
	}
}
