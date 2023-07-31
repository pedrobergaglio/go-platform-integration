package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

type Notification struct {
	OrderIDLink string `json:"resource"`
}

type orderData struct {
	OrderItems []OrderItem `json:"order_items"`
}

type OrderItem struct {
	Item     Item `json:"item"`
	Quantity int  `json:"quantity"`
}

type Item struct {
	ID string `json:"id"`
}

type MeliItem struct {
	MeliID   string
	Quantity int
}

// handleMeliWebhook process an order notification from meli.
// Receives the order id, then requests the data of that order to get the items and quantities
// For each item, adds a movement in appsheet to substract the sold quantity
func handleMeliWebhook(w http.ResponseWriter, r *http.Request) {

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

	var notification Notification
	err = json.Unmarshal(body, &notification)
	if err != nil {
		log.Println("error unmarshaling request body:", err)
		return
	}

	//Get order info

	if notification.OrderIDLink == os.Getenv("last_order_id_link") {
		log.Println("notification received twice")
		return
	}

	log.Printf("received meli order notification: %s", notification.OrderIDLink)

	err = os.Setenv("last_order_id_link", notification.OrderIDLink)
	if err != nil {
		log.Println("failed to set env last_order_id_link:", err)
		return
	}

	URL := fmt.Sprintf("https://api.mercadolibre.com%s", notification.OrderIDLink)

	req, err := http.NewRequest(http.MethodGet, URL, nil)
	if err != nil {
		log.Println("failed to create request:", err)
		return
	}

	auth := "Bearer " + os.Getenv("MELI_ACCESS_TOKEN")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", auth)

	client := http.DefaultClient
	resp, err := client.Do(req)
	if err != nil {
		log.Println("error getting order data:", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != 403 {
		log.Println("meli error 403. retrying.")
		time.Sleep(time.Minute)

		URL := fmt.Sprintf("https://api.mercadolibre.com%s", notification.OrderIDLink)

		req, err := http.NewRequest(http.MethodGet, URL, nil)
		if err != nil {
			log.Println("failed to create request:", err)
			return
		}

		auth := "Bearer " + os.Getenv("MELI_ACCESS_TOKEN")
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json")
		req.Header.Set("Authorization", auth)

		client := http.DefaultClient
		resp, err := client.Do(req)
		if err != nil {
			log.Println("error getting order data:", err)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != 403 {
			log.Println("meli error 403. retrying.")
			time.Sleep(time.Minute)

			URL := fmt.Sprintf("https://api.mercadolibre.com%s", notification.OrderIDLink)

			req, err := http.NewRequest(http.MethodGet, URL, nil)
			if err != nil {
				log.Println("failed to create request:", err)
				return
			}

			auth := "Bearer " + os.Getenv("MELI_ACCESS_TOKEN")
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Accept", "application/json")
			req.Header.Set("Authorization", auth)

			client := http.DefaultClient
			resp, err := client.Do(req)
			if err != nil {
				log.Println("error getting order data:", err)
				return
			}
			defer resp.Body.Close()

		}

	}

	if resp.StatusCode != http.StatusOK {
		log.Println("unexpected status code from meli while getting order data:" + fmt.Sprint(resp.StatusCode))
		return
	}

	///////////////////////////////////////////////////////////////

	//Process order info

	// Read the request body
	body, err = io.ReadAll(resp.Body)
	if err != nil {
		log.Println("error reading order request body:", err)
		return
	}

	var response orderData
	err = json.Unmarshal(body, &response)
	if err != nil {
		log.Println("error unmarshaling order request body:", err)
		return
	}

	// Create a slice to store the items
	var items []MeliItem

	// Iterate over the line_items and save the product_id and quantity

	for _, orderItem := range response.OrderItems {
		item := MeliItem{
			MeliID:   convertToString(orderItem.Item.ID),
			Quantity: orderItem.Quantity,
		}
		items = append(items, item)
	}

	if len(items) == 0 {
		log.Println("no items received in notification")
		return
	}

	for _, item := range items {

		//trim the 'MLA' part
		item.MeliID = strings.TrimPrefix(item.MeliID, "MLA")

		product_id, err := productIDFromMeliID(item.MeliID)
		if err != nil {
			log.Println("error finding meli code in database:", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		log.Println("product_id:", product_id, "quantity:", item.Quantity)
		//log.Println("quantity:", item.Quantity)

		_, err = addMovement(product_id, "0", convertToString(-item.Quantity), "0", "0", "Mercado Libre")
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
	log.Println("meli notification processed")

}

// Updates meli a publication value (stock or price)
func updateMeli(meli_id string, field string, value interface{}) string {

	// Retry for a maximum of 3 times
	maxRetries := 3
	retries := 0

	for {

		URL := fmt.Sprintf("https://api.mercadolibre.com/items/MLA%s", fmt.Sprint(meli_id))
		payload := fmt.Sprintf(`{"%s": %s}`, fmt.Sprint(field), fmt.Sprint(value))

		req, err := http.NewRequest(http.MethodPut, URL, bytes.NewBufferString(payload))
		if err != nil {
			return "error creating request for meli:" + fmt.Sprint(err)
		}

		auth := "Bearer " + os.Getenv("MELI_ACCESS_TOKEN")
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json")
		req.Header.Set("Authorization", auth)

		client := http.DefaultClient
		resp, err := client.Do(req)
		if err != nil {
			return "error updating product in meli:" + fmt.Sprint(err)
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			return ""
		} else if resp.StatusCode != 403 {
			if retries < maxRetries {
				log.Println("meli error 403. retrying...")
				retries++
				time.Sleep(time.Minute)
			} else {
				return "max retries reached. giving up."
			}

		} else {
			return "unexpected status code from meli:" + fmt.Sprint(resp.StatusCode)
		}

	}
}
