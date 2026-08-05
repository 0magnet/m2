// Package main order.go
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	htmpl "html/template"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/bitfield/script"
	"github.com/gofiber/fiber/v3"
	"github.com/stripe/stripe-go/v81"
	"github.com/stripe/stripe-go/v81/paymentintent"
)

// validPIID matches Stripe PaymentIntent IDs: "pi_" followed by alphanumeric chars.
// Also allows plain alphanumeric+underscore+hyphen for test order IDs.
var validPIID = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

func handleOrder(r *fiber.App) {
	r.Get("/checkout.css", func(c fiber.Ctx) error {
		c.Set("Content-Type", "text/css;charset=utf-8")
		_, err := c.Status(fiber.StatusOK).Write([]byte(h.CheckoutCSS()))
		return err
	})

	r.Get("/complete", func(c fiber.Ctx) error {
		// Complete template
		completetmpl := htmpl.New("index")
		if _, err := completetmpl.Parse(h.CompletePage()); err != nil {
			msg := fmt.Sprintf("Error parsing complete page template: %v", err)
			log.Println(msg)
			return c.Status(fiber.StatusInternalServerError).SendString(msg)
		}
		if _, err := completetmpl.New("wasm").Parse(h.Wasm()); err != nil {
			log.Println("Error parsing wasm template:", err)
			msg := fmt.Sprintf("Error parsing wasm template: %v", err)
			log.Println(msg)
			return c.Status(fiber.StatusInternalServerError).SendString(msg)
		}
		h1 := htmlPageTemplateData
		/*
			proto := "http"
			if c.Secure() {
				proto += "s"
			}
		*/
		proto := "https"
		h1.Canonical = proto + `://` + c.Hostname() + c.OriginalURL()
		h1.BaseURL = proto + `://` + c.Hostname()
		h1.RequestHost = c.Hostname()
		h1.Protocol = proto
		h1.Time = time.Now().Format(time.RFC3339Nano)
		h1.Year = fmt.Sprintf("%v", time.Now().Year())
		tmplData := map[string]interface{}{
			"Page": h1,
		}
		var result bytes.Buffer
		err := completetmpl.Execute(&result, tmplData)
		if err != nil {
			msg := fmt.Sprintf("Could not execute html template %v", err)
			log.Println(msg)
			return c.Status(fiber.StatusInternalServerError).SendString(msg)
		}
		c.Set("Content-Type", "text/html;charset=utf-8")
		return c.Status(fiber.StatusOK).Send(result.Bytes())
	})

	r.Get("/order/:piid", func(c fiber.Ctx) error {
		piid := c.Params("piid")
		if !validPIID.MatchString(piid) {
			return c.Status(fiber.StatusBadRequest).SendString("Invalid order ID")
		}
		order, err := script.File("orders/" + piid + ".json").Bytes()
		if err != nil {
			return c.Status(fiber.StatusNotFound).SendString("Order not found")
		}
		return c.Status(fiber.StatusOK).Send(order)
	})

	r.Get("/order/:piid/html", func(c fiber.Ctx) error {
		piid := c.Params("piid")
		if !validPIID.MatchString(piid) {
			return c.Status(fiber.StatusBadRequest).SendString("Invalid order ID")
		}
		order, err := script.File("orders/" + piid + ".json").Bytes()
		if err != nil {
			return c.Status(fiber.StatusNotFound).SendString("Order not found")
		}
		var m map[string]interface{}
		if err := json.Unmarshal(order, &m); err != nil {
			return c.Status(500).SendString("failed to unmarshal order json: " + err.Error())
		}
		receipt, err := buildReceipt(m, piid)
		if err != nil {
			return c.Status(500).SendString("failed to build receipt: " + err.Error())
		}
		return c.Status(200).SendString(string(receipt))
	})

	r.Post("/create-payment-intent", func(c fiber.Ctx) error {
		rawBody := c.Body()
		if rawBody == nil {
			log.Printf("Failed to read raw request body")
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to read request body"})
		}

		var req struct {
			Items []item `json:"items"`
		}
		if err := json.Unmarshal(rawBody, &req); err != nil {
			log.Printf("Failed to parse JSON: %v", err)
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}

		if len(req.Items) == 0 {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "No items in request"})
		}

		// Validate each item's amount against the server-side product catalog.
		// Client sends ID as "partno X qty" for products, or "shipping-to|..." for shipping.
		total := int64(0)
		for _, it := range req.Items {
			if it.Amount <= 0 {
				log.Printf("Rejected item with non-positive amount: %s = %d", it.ID, it.Amount)
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid item amount"})
			}
			if strings.HasPrefix(it.ID, "shipping-to|") {
				// Shipping line — accept the client-supplied amount
				total += it.Amount
				continue
			}
			// Extract partno and qty from "partno X qty"
			expectedAmt, err := validateItemAmount(it.ID, it.Amount)
			if err != nil {
				log.Printf("Item validation failed for %q: %v", it.ID, err)
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Item validation failed"})
			}
			total += expectedAmt
		}

		if total < 50 {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Order total must be at least $0.50"})
		}

		params := &stripe.PaymentIntentParams{
			Amount:   stripe.Int64(total),
			Currency: stripe.String(string(stripe.CurrencyUSD)),
		}
		pi, err := paymentintent.New(params)
		if err != nil {
			log.Printf("Failed to create PaymentIntent: %v", err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}

		log.Printf("Created PaymentIntent %s for %d cents", pi.ID, total)
		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"clientSecret":   pi.ClientSecret,
			"dpmCheckerLink": fmt.Sprintf("https://dashboard.stripe.com/settings/payment_methods/review?transaction_id=%s", pi.ID),
		})
	})

	r.Post("/submit-order", func(c fiber.Ctx) error {
		var requestData struct {
			LocalStorageData map[string]interface{} `json:"localStorageData"`
			PaymentIntentID  string                 `json:"paymentIntentId"`
		}

		if err := c.Bind().Body(&requestData); err != nil {
			log.Println(err)
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request data"})
		}

		if !validPIID.MatchString(requestData.PaymentIntentID) {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid payment intent ID"})
		}

		log.Printf("Received payment intent ID: %s\n", requestData.PaymentIntentID)

		paymentIntent, err := paymentintent.Get(requestData.PaymentIntentID, nil)
		if err != nil {
			log.Printf("Error retrieving payment intent: %v", err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Unable to verify payment"})
		}
		if paymentIntent.Status != stripe.PaymentIntentStatusSucceeded {
			log.Printf("Payment was not successful, status: %s", paymentIntent.Status)
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Payment not successful"})
		}

		ordersDir := "./orders"
		if err := os.MkdirAll(ordersDir, os.ModePerm); err != nil {
			log.Printf("Error creating orders directory: %v", err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Unable to save order"})
		}

		filePath := filepath.Join(ordersDir, fmt.Sprintf("%s.json", requestData.PaymentIntentID))

		// Idempotency: if the order file already exists, don't overwrite or reprint
		if _, err := os.Stat(filePath); err == nil {
			log.Printf("Order %s already exists, skipping duplicate submission", requestData.PaymentIntentID)
			return c.Status(fiber.StatusOK).JSON(fiber.Map{"message": "Order already submitted"})
		}

		// Include the verified Stripe amount alongside the client-supplied data
		orderData := map[string]interface{}{
			"clientData":    requestData.LocalStorageData,
			"verifiedCents": paymentIntent.Amount,
			"currency":      string(paymentIntent.Currency),
			"stripeStatus":  string(paymentIntent.Status),
			"submittedAt":   time.Now().Format(time.RFC3339),
		}

		data, err := json.MarshalIndent(orderData, "", "  ")
		if err != nil {
			log.Printf("Error marshaling data to json: %v", err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Unable to save order"})
		}
		if err := os.WriteFile(filePath, data, 0o644); err != nil {
			log.Printf("Error writing data to file: %v", err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Unable to save order"})
		}

		// ---- Print receipt via CUPS (non-blocking so your response is snappy)
		go func(pid string, local map[string]interface{}) {
			receipt, err := buildReceipt(local, pid)
			if err != nil {
				log.Printf("build receipt failed: %v", err)
				return
			}
			if err := sendToCUPS(receipt, "Order "+pid); err != nil {
				log.Printf("print failed: %v", err)
				_ = os.WriteFile(filepath.Join(ordersDir, pid+".print_failed"), []byte(err.Error()), 0o644)
			}
		}(requestData.PaymentIntentID, requestData.LocalStorageData)

		return c.Status(fiber.StatusOK).JSON(fiber.Map{"message": "Order submitted successfully"})
	})

	/*
			r.Post("/reprint/:pid", func(c fiber.Ctx) error {
		    pid := c.Params("pid")
		    b, err := os.ReadFile(filepath.Join("./orders", pid+".json"))
		    if err != nil { return c.Status(404).SendString("not found") }
		    var m map[string]interface{}
		    if err := json.Unmarshal(b, &m); err != nil { return c.Status(500).SendString(err.Error()) }
		    receipt, err := buildReceipt(m, pid)
		    if err != nil { return c.Status(500).SendString(err.Error()) }
		    if err := sendToCUPS(receipt, "Order "+pid); err != nil {
		        return c.Status(500).SendString(err.Error())
		    }
		    return c.SendStatus(204)
		})
	*/
}

func buildReceipt(local map[string]interface{}, paymentIntentID string) ([]byte, error) {
	// Pretty JSON body from what you already persisted
	body, err := json.MarshalIndent(local, "", "  ")
	if err != nil {
		return nil, err
	}
	// Simple text receipt header
	ts := time.Now().Format("2006-01-02 15:04:05")
	hdr := fmt.Sprintf(
		"==================== ORDER ====================\n"+
			"PaymentIntent: %s\nTime: %s\n===============================================\n\n",
		paymentIntentID, ts,
	)
	// Footer (optional)
	ftr := "\n\n---------------------- END ---------------------\n"
	receipt := append([]byte(hdr), body...)
	receipt = append(receipt, []byte(ftr)...)
	return receipt, nil
}

// serverPriceCents looks up a product's price from the server-side catalog by part number.
func serverPriceCents(partno string) (int64, error) {
	allproductsMu.RLock()
	prods := allproducts
	allproductsMu.RUnlock()
	for _, prod := range prods {
		if prod.Partno == partno {
			return parsePriceCents(prod.Price), nil
		}
	}
	return 0, fmt.Errorf("product %q not found in catalog", partno)
}

// parsePriceCents converts a price string like "$1.23" or "1.23" to cents.
func parsePriceCents(s string) int64 {
	if s == "" {
		return 0
	}
	s = strings.TrimPrefix(s, "$")
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0
	}
	if f < 0 {
		return -int64(-f*100 + 0.5)
	}
	return int64(f*100 + 0.5)
}

// validateItemAmount parses a client item ID ("partno X qty"), looks up the
// server-side price, computes the expected total, and returns it. If the
// client-supplied amount doesn't match, an error is returned.
func validateItemAmount(itemID string, clientAmount int64) (int64, error) {
	// Parse "partno X qty"
	parts := strings.SplitN(itemID, " X ", 2)
	if len(parts) != 2 {
		return 0, fmt.Errorf("unexpected item ID format: %q", itemID)
	}
	partno := parts[0]
	qty, err := strconv.Atoi(parts[1])
	if err != nil || qty <= 0 {
		return 0, fmt.Errorf("invalid quantity in item ID %q", itemID)
	}

	unitCents, err := serverPriceCents(partno)
	if err != nil {
		return 0, err
	}
	expected := unitCents * int64(qty)
	if expected != clientAmount {
		return 0, fmt.Errorf("amount mismatch for %q: client sent %d cents, server expects %d cents", partno, clientAmount, expected)
	}
	return expected, nil
}

// escape for inclusion inside *double quotes* in a bash command string
func bashEscapeDoubleQuoted(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	s = strings.ReplaceAll(s, "$", `\$`)
	s = strings.ReplaceAll(s, "`", "\\`")
	return s
}

func sendToCUPS(receipt []byte, title string) error {
	if title == "" {
		title = "Order"
	}
	var cmd strings.Builder
	cmd.WriteString("lp")

	if f.PrinterName != "" {
		cmd.WriteString(` -d "`)
		cmd.WriteString(bashEscapeDoubleQuoted(f.PrinterName))
		cmd.WriteString(`"`)
	}

	cmd.WriteString(` -t "`)
	cmd.WriteString(bashEscapeDoubleQuoted(title))
	cmd.WriteString(`"`)

	if f.CupsOptions != "" {
		for _, opt := range strings.Split(f.CupsOptions, ",") {
			opt = strings.TrimSpace(opt)
			if opt == "" {
				continue
			}
			cmd.WriteString(` -o "`)
			cmd.WriteString(bashEscapeDoubleQuoted(opt))
			cmd.WriteString(`"`)
		}
	}

	full := fmt.Sprintf(`bash -lc %q`, cmd.String())

	_, err := script.Echo(string(receipt)).Exec(full).Stdout()
	if err != nil {
		return fmt.Errorf("lp failed: %v", err)
	}
	return nil
}
