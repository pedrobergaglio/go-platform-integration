package main

import (
	//"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	//"time"
)

func convertToString(value interface{}) string {
	switch v := value.(type) {
	case string:
		return v
	case int, int64, float32, float64:
		return fmt.Sprintf("%v", v)
	default:
		return ""
	}
}

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

func handleWCWebhook(w http.ResponseWriter, r *http.Request) {

	log.Println("received wc order notification")

	// Ensure that the request method is POST
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	// Read the request body
	body, err := ioutil.ReadAll(r.Body)
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
	var items []MeliItem

	// Iterate over the line_items and save the product_id and quantity
	for _, lineItem := range payload.LineItems {
		item := MeliItem{
			ProductID: convertToString(lineItem.ProductID),
			Quantity:  lineItem.Quantity,
		}
		items = append(items, item)
	}

	if len(items) == 0 {
		log.Println("no items received in notification")
	}
	// Process the webhook request or perform any desired actions using the items slice

	// Print the items
	for _, item := range items {
		log.Println("product_id:", item.ProductID)
		log.Println("quantity:", item.Quantity)

		product_id, err := productIDFromWC(item.ProductID)
		if err != nil {
			log.Println("error finding product in database or connecting to it:", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		_, err = postMovement(product_id, item.Quantity, "Woocommerce")
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
