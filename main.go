package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
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

type AppsheetWebhookPayload struct {
	Product interface{} `json:"product_id"`
}

type stockData struct {
	Total   string `json:"total_stock"`
	Product string `json:"product_id"`
}

type WCData struct {
	WooID   string `json:"wc_id"`
	Product string `json:"product_id"`
}

func handleAppsheetWebhook(w http.ResponseWriter, r *http.Request) {
	// Ensure that the request method is POST
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	// Parse the incoming request body
	var payload AppsheetWebhookPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		log.Println("Error decoding payload:", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	// Log the received payload
	log.Println("Received movement, product ID:", payload.Product)

	//Wait 3 seconds
	log.Println("Waiting to update...")
	time.Sleep(2 * time.Second)

	//Get product data
	stockgetURL := fmt.Sprintf("https://api.appsheet.com/api/v2/apps/%s/tables/STOCK/Action", os.Getenv("appsheet_id"))
	find_in_stock := fmt.Sprintf(`{
		"Action": "Find",
		  "Properties": 
		  	{"Locale": "en-US",
			"Location": "47.623098, -122.330184",
			"Timezone": "Pacific Standard Time",
			"UserSettings": {
				"Option 1": "value1",
				"Option 2": "value2"}
			},
		"Rows": []}`)
	get, err := http.NewRequest(http.MethodPost, stockgetURL, bytes.NewBufferString(find_in_stock))
	if err != nil {
		log.Println("Error creating request to find the product ID and quantity:", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	get.Header.Set("Content-Type", "application/json")
	get.Header.Set("ApplicationAccessKey", os.Getenv("appsheet_key"))

	client := http.DefaultClient
	// Get product data
	appsresp, err := client.Do(get)

	if err != nil {
		log.Println("Error geting product in Appsheet:", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer appsresp.Body.Close()

	if appsresp.StatusCode != http.StatusOK {
		log.Println("Unexpected status code from Appsheet:", appsresp.StatusCode)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Read the response body
	body, err := ioutil.ReadAll(appsresp.Body)
	if err != nil {
		log.Fatal("Error reading response body:", err)
	}

	// Define a struct to hold the response data
	var responseData []stockData

	// Unmarshal the JSON data into the struct
	err = json.Unmarshal(body, &responseData)
	if err != nil {
		log.Fatal("Error unmarshaling response data:", err)
	}
	total := ""
	for _, item := range responseData {
		if item.Product == payload.Product {
			total = item.Total
		}
	}

	//wcCodigo := responseData[0].WcCodigo

	// Print the values
	log.Println("Total stock:", total)

	//Get WOOCOMMERCE data
	stockgetURL = fmt.Sprintf("https://api.appsheet.com/api/v2/apps/%s/tables/PLATFORMS/Action", os.Getenv("appsheet_id"))
	find_in_stock = fmt.Sprintf(`{
		"Action": "Find",
		  "Properties": {"Locale": "en-US","Location": "47.623098, -122.330184","Timezone": "Pacific Standard Time","UserSettings": {"Option 1": "value1","Option 2": "value2"}},
		"Rows": []}`)
	get, err = http.NewRequest(http.MethodPost, stockgetURL, bytes.NewBufferString(find_in_stock))
	if err != nil {
		log.Println("Error creating request to find the product ID and quantity:", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	get.Header.Set("Content-Type", "application/json")
	get.Header.Set("ApplicationAccessKey", os.Getenv("appsheet_key"))

	client = http.DefaultClient
	// Get product data
	appsresp, err = client.Do(get)

	if err != nil {
		log.Println("Error geting product in Appsheet:", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer appsresp.Body.Close()

	if appsresp.StatusCode != http.StatusOK {
		log.Println("Unexpected status code from Appsheet:", appsresp.StatusCode)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Read the response body
	body, err = ioutil.ReadAll(appsresp.Body)
	if err != nil {
		log.Fatal("Error reading response body:", err)
	}

	// Define a struct to hold the response data
	var PlatformData []WCData

	// Unmarshal the JSON data into the struct
	err = json.Unmarshal(body, &PlatformData)
	if err != nil {
		log.Fatal("Error unmarshaling response data:", err)
	}
	woo_id := ""
	for _, item := range PlatformData {
		if item.Product == payload.Product {
			woo_id = item.WooID
		}
	}
	//wcCodigo := responseData[0].WcCodigo

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

func main() {

	loadConfig()

	// Register the webhook handler function with the default server mux
	http.HandleFunc("/movimientos", handleAppsheetWebhook)
	http.HandleFunc("/woocommerce", handleWCWebhook)

	// Start the server and specify the port to listen on
	log.Println("Server listening on port 8080...")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
