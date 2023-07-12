package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/spf13/viper"
)

func loadConfig() {
	viper.SetConfigFile("resources/.env") // Set the path to your configuration file
	viper.SetConfigType("env")            // Specify the configuration file type (e.g., "env", "json", "yaml", etc.)

	err := viper.ReadInConfig() // Read the configuration file
	if err != nil {
		log.Fatalf("Failed to read configuration file: %s", err)
	}

	// Set environment variables based on the configuration values
	for key, value := range viper.AllSettings() {
		//log.Println("adding ", key)
		if err := os.Setenv(key, value.(string)); err != nil {
			log.Fatalf("Failed to set environment variable %s: %s", key, err)
		}
	}
}

type ASMovementWebhookPayload struct {
	ProductID interface{} `json:"product_id"`
}

type ASPriceWebhookPayload struct {
	ProductID interface{} `json:"product_id"`
	SalePrice string      `json:"sale_price"`
}

type stockData struct {
	Total   string `json:"total_stock"`
	Product string `json:"product_id"`
}

type WCData struct {
	WooID   string `json:"wc_id"`
	Product string `json:"product_id"`
}

func handleASMovementWebhook(w http.ResponseWriter, r *http.Request) {
	// Ensure that the request method is POST
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	// Parse the incoming request body
	var payload ASMovementWebhookPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		log.Println("Error decoding payload:", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	// Log the received payload
	log.Println("Received movement, product ID:", payload.ProductID)

	//Wait 3 seconds
	log.Println("Waiting to update...")
	time.Sleep(2 * time.Second)

	//Get product data
	total, err := getProductTotalStock(convertToString(payload.ProductID))
	if err != nil {
		log.Println("Error getting product total stock:", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Print the values
	log.Println("Total stock:", total)

	//Get WOOCOMMERCE data
	woo_id, error := getWCID(convertToString(payload.ProductID))
	if error != "" {
		log.Println(error)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Print the values
	log.Println("Woocommerce ID:", woo_id)

	// Update the WooCommerce product
	wcURL := "https://www.energiaglobal.com.ar/wp-json/wc/v3/products/" + fmt.Sprint(woo_id)
	wcPayload := `{"stock_quantity": ` + fmt.Sprint(total) + `}`

	req, err := http.NewRequest(http.MethodPut, wcURL, bytes.NewBufferString(wcPayload))
	if err != nil {
		log.Println("Error creating request for WooCommerce:", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(os.Getenv("wc_client"), os.Getenv("wc_secret"))

	client := http.DefaultClient
	resp, err := client.Do(req)
	if err != nil {
		log.Println("Error updating product in WooCommerce:", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Println("Unexpected status code from WooCommerce:", resp.StatusCode)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Write a success response if everything is processed successfully
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Webhook processed successfully"))
	log.Println("Movement processed")

}

func handleASPriceWebhook(w http.ResponseWriter, r *http.Request) {
	// Ensure that the request method is POST
	if r.Method != http.MethodPost {
		log.Println(http.StatusMethodNotAllowed)
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	// Parse the incoming request body
	var payload ASPriceWebhookPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		log.Println("Error decoding payload:", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	// Log the received payload
	log.Println("Updated price in product:", payload.ProductID)

	//Get WOOCOMMERCE data
	woo_id, error := getWCID(convertToString(payload.ProductID))
	if error != "" {
		log.Println("Error getting WC ID ", error)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Update the WooCommerce product
	wcURL := "https://www.energiaglobal.com.ar/wp-json/wc/v3/products/" + fmt.Sprint(woo_id)
	wcPayload := fmt.Sprintf(`{"regular_price": '%s'}`, fmt.Sprint(payload.SalePrice))

	req, err := http.NewRequest(http.MethodPut, wcURL, bytes.NewBufferString(wcPayload))
	if err != nil {
		log.Println("Error creating request for WooCommerce:", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(os.Getenv("wc_client"), os.Getenv("wc_secret"))

	client := http.DefaultClient
	resp, err := client.Do(req)
	if err != nil {
		log.Println("Error updating product in WooCommerce:", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Println("Unexpected status code from WooCommerce:", resp.StatusCode)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Write a success response if everything is processed successfully
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Webhook processed successfully"))
	log.Println("Price updated")

}

func main() {
	loadConfig()

	// Register the webhook handler functions with the default server mux
	http.HandleFunc("/movimientos", handleASMovementWebhook)
	http.HandleFunc("/woocommerce", handleWCWebhook)
	http.HandleFunc("/price", handleASPriceWebhook)

	// Use PORT environment variable provided by Railway or default to 8080
	port := ":" + os.Getenv("PORT")
	if port == ":" {
		port = ":8080"
	}

	// Start the server and specify the host and port
	log.Println("Server listening on", port)
	log.Fatal(http.ListenAndServe("0.0.0.0"+port, nil))
}
