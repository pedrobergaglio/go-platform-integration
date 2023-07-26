package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	//"time"
	"os"
)

type WCItem struct {
	ProductID string
	Quantity  int
}

type WCWebhookPayload struct {
	LineItems []struct {
		ProductID interface{} `json:"product_id"`
		Quantity  int         `json:"quantity"`
	} `json:"line_items"`
}

// Process an order notification from woocommerce
// For each item in the order, adds a movement in appsheet to substract the quantity sold
func handleWCWebhook(w http.ResponseWriter, r *http.Request) {

	log.Println("received wc order notification")

	// Ensure that the request method is POST
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	// Read the request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Println("error reading request body:", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Parse the request body into the WebhookPayload struct
	var payload WCWebhookPayload
	err = json.Unmarshal(body, &payload)
	if err != nil {
		log.Println("error parsing request body:", err)
		log.Println(body)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Create a slice to store the items
	var items []WCItem

	// Iterate over the line_items and save the product_id and quantity
	for _, lineItem := range payload.LineItems {
		item := WCItem{
			ProductID: convertToString(lineItem.ProductID),
			Quantity:  lineItem.Quantity,
		}
		items = append(items, item)
	}

	if len(items) == 0 {
		log.Println("no items received in notification")
		return
	}
	// Process the webhook request or perform any desired actions using the items slice

	// Print the items
	for _, item := range items {

		product_id, err := productIDFromWCID(item.ProductID)
		if err != nil {
			log.Println("error finding product in database:", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		_, err = addMovement(product_id, "0", convertToString(-item.Quantity), "0", "0", "Woocommerce")
		if err != nil {
			log.Println("error posting movement:", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

	}

	// Print the raw request body
	//log.Println(string(body))

	// Respond with a success status
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("webhook processed successfully"))
	log.Println("woocommerce notification processed")
}

// Updates a wc product property
func updateWC(wc_id string, field string, value interface{}) string {

	wcURL := fmt.Sprintf("https://www.energiaglobal.com.ar/wp-json/wc/v3/products/%s", fmt.Sprint(wc_id))
	wcPayload := fmt.Sprintf(`{"%s": %s}`, fmt.Sprint(field), fmt.Sprint(value))

	req, err := http.NewRequest(http.MethodPut, wcURL, bytes.NewBufferString(wcPayload))
	if err != nil {
		return "error creating request for WooCommerce:" + fmt.Sprint(err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(os.Getenv("wc_client"), os.Getenv("wc_secret"))

	client := http.DefaultClient
	resp, err := client.Do(req)
	if err != nil {
		return "error updating product in WooCommerce:" + fmt.Sprint(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "unexpected status code from WooCommerce:" + fmt.Sprint(resp.StatusCode)
	}

	return ""
}
