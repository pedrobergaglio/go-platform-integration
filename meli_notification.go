package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
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

func handleMeliWebhook(w http.ResponseWriter, r *http.Request) {

	log.Println("received meli order notification")

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

	log.Print(string(body))

	var notification Notification
	err = json.Unmarshal(body, &notification)
	if err != nil {
		log.Println("error unmarshaling request body:", err)
		return
	}

	//Get order info

	order_link := notification.OrderIDLink

	URL := fmt.Sprintf("https://api.mercadolibre.com%s", fmt.Sprint(order_link))

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
		log.Println("error getting order info:", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Println("unexpected status code from meli:" + fmt.Sprint(resp.StatusCode))
		return
	}

	///////////////////////////////////////////////////////////////

	//Process order info

	// Read the request body
	body, err = io.ReadAll(resp.Body)
	if err != nil {
		log.Println("error reading request body:", err)
		return
	}

	var response orderData
	err = json.Unmarshal(body, &response)
	if err != nil {
		log.Println("error unmarshaling request body:", err)
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
	}

	for _, item := range items {
		log.Printf("meli_id: MLA%s", item.MeliID)
		log.Println("quantity:", item.Quantity)

		product_id, err := productIDFromMeli(item.MeliID)
		if err != nil {
			log.Println("error finding product in database or connecting to it:", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		_, err = postMovement(product_id, item.Quantity, "Mercado Libre")
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
