package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
)

type WebhookPayload struct {
	Product string `json:"product_id"`
}

type ResponseData struct {
	Total interface{} `json:"total_stock"`
	//WcCodigo string `json:"wc_code"`
}

func handleWebhook(w http.ResponseWriter, r *http.Request) {
	// Ensure that the request method is POST
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	// Parse the incoming request body
	var payload WebhookPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		log.Println("Error decoding payload:", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	log.Println(payload)
	// Log the received payload
	log.Println("Received webhook payload")
	log.Println("Producto:", payload.Product)

	//Wait 3 seconds
	//time.Sleep(3 * time.Second)

	//Get product data
	stockgetURL := "https://api.appsheet.com/api/v2/apps/69d001e5-d43f-4dfe-a7ed-fbbe155ab9b8/tables/stock/Action"
	find_in_stock := `{
		"Action": "Find",
		  "Properties": {"Locale": "en-US","Location": "47.623098, -122.330184","Timezone": "Pacific Standard Time","UserSettings": {"Option 1": "value1","Option 2": "value2"}},
		"Rows": [{"product_id" : "` + payload.Product + `"}]}`
	get, err := http.NewRequest(http.MethodPost, stockgetURL, bytes.NewBufferString(find_in_stock))
	if err != nil {
		log.Println("Error creating request to find the product ID and quantity:", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	get.Header.Set("Content-Type", "application/json")
	get.Header.Set("ApplicationAccessKey", "V2-MbFFj-boUAV-MLqf1-Pqxgl-be4j3-WGRlm-dYVTO-aObJo")

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
	var responseData []ResponseData

	// Unmarshal the JSON data into the struct
	err = json.Unmarshal(body, &responseData)
	if err != nil {
		log.Fatal("Error unmarshaling response data:", err)
	}

	// Access the first element and extract the values
	total := responseData[0].Total
	//wcCodigo := responseData[0].WcCodigo

	// Print the values
	log.Println("Total:", total)
	return
	//log.Println("WC_CODIGO:", wcCodigo)
	wcCodigo := "0"
	// Update the WooCommerce product
	wcURL := "https://www.energiaglobal.com.ar/wp-json/wc/v3/products/" + wcCodigo
	wcPayload := `{"stock_quantity": ` + fmt.Sprint(total) + `}`

	req, err := http.NewRequest(http.MethodPut, wcURL, bytes.NewBufferString(wcPayload))
	if err != nil {
		log.Println("Error creating request for WooCommerce:", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth("ck_59e0243e799c898a806853f6b43b8b8d26f746e7", "cs_21419fd0b431d050179cd24f03399747286cd8db")

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

}

func main() {
	// Register the webhook handler function with the default server mux
	http.HandleFunc("/movimientos", handleWebhook)

	// Start the server and specify the port to listen on
	log.Println("Server listening on port 8080...")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
