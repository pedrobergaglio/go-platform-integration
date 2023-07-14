package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/spf13/viper"
)

func loadConfig() {
	viper.SetConfigFile("resources/.env") // Set the path to your configuration file
	viper.SetConfigType("env")
	// Specify the configuration file type (e.g., "env", "json", "yaml", etc.)

	err := viper.ReadInConfig() // Read the configuration file
	if err != nil {
		log.Fatalf("failed to read configuration file: %s", err)
	}

	// Set environment variables based on the configuration values
	for key, value := range viper.AllSettings() {
		//log.Println("adding ", key)
		if err := os.Setenv(key, value.(string)); err != nil {
			log.Fatalf("failed to set environment variable %s: %s", key, err)
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

type PlatformsData struct {
	Product string `json:"product_id"`
	WCID    string `json:"wc_id"`
	MeliID  string `json:"meli_id"`
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
		log.Println("error decoding payload:", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	// Log the received payload
	log.Println("movement in product_id:", payload.ProductID)

	//Wait 3 seconds
	//log.Println("Waiting two seconds to update...")
	time.Sleep(2 * time.Second)

	//Get product data
	total, err := getProductTotalStock(convertToString(payload.ProductID))
	if err != nil {
		log.Println("error getting product total stock:", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Print the values
	log.Println("total_stock:", total)

	//Get WOOCOMMERCE data
	meli_id, wc_id, error := getPlatformsID(convertToString(payload.ProductID))
	if error != "" {
		log.Println(error)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Print the values
	//log.Println("Woocommerce ID:", woo_id)

	// Update the MELI product
	if meli_id != "0" {
		error = updateMeli(meli_id, "available_quantity", total)
		if error != "" {
			log.Println("error updating stock in MELI:", error)
			return
		} else {
			log.Println("entro meli")
		}
	} else {
		log.Println("product not linked to meli")
	}

	// Update the WooCommerce product
	if wc_id != "0" {
		error = updateWC(wc_id, "stock_quantity", total)
		if error != "" {
			log.Println("error updating stock in WC:", error)
			return
		} else {
			log.Println("entro wc")
		}
	} else {
		log.Println("product not linked to wc")
	}

	// Write a success response if everything is processed successfully
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("webhook processed successfully"))
	log.Println("movement processed")

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
		log.Println("error decoding payload:", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	// Log the received payload
	log.Println("updated price product in Appsheet:", payload.ProductID)

	//Get platforms data
	meli_id, wc_id, error := getPlatformsID(convertToString(payload.ProductID))
	if error != "" {
		log.Println("error getting platforms id:", error)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Update the MELI product
	if meli_id != "0" {
		error = updateMeli(meli_id, "price", payload.SalePrice)
		if error != "" {
			log.Println("error updating meli price:", error)
			w.WriteHeader(http.StatusInternalServerError)
			return
		} else {
			log.Println("entro meli")
		}
	} else {
		log.Println("product not linked to meli")
	}

	// Update the WooCommerce product
	if wc_id != "0" {
		error = updateWC(wc_id, "regular_price", `"`+payload.SalePrice+`"`)
		if error != "" {
			log.Println("error updating Woocommerce price:", error)
			w.WriteHeader(http.StatusInternalServerError)
			return
		} else {
			log.Println("entro wc")
		}
	} else {
		log.Println("product not linked to wc")
	}

	// Write a success response if everything is processed successfully
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("webhook processed successfully"))
	log.Println("price updated")

}

func productIDFromWC(wc_id string) (string, error) {

	// Define the data struct for the response
	type ResponseData struct {
		AppsheetProductID string `json:"product_id"`
		AppsheetWCID      string `json:"wc_id"`
	}

	// Prepare the payload for finding the product ID and quantity
	payload := `{
			"Action": "Find",
			"Properties": {
				"Locale": "es-US",
				"Timezone": "Argentina Standard Time",
			},
			"Rows": []
		}`

	//"Selector": "Filter("PLATFORMS", ISNOTBLANK([wc_id]))",

	// Create the request
	requestURL := fmt.Sprintf("https://api.appsheet.com/api/v2/apps/%s/tables/PLATFORMS/Action", os.Getenv("appsheet_id"))
	key := os.Getenv("appsheet_key")
	req, err := http.NewRequest(http.MethodPost, requestURL, bytes.NewBufferString(payload))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %v", err)
	}

	// Set request headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("ApplicationAccessKey", key)

	// Send the request
	client := http.DefaultClient
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	// Check the response status code
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %v", err)
	}

	// Unmarshal the JSON data into the struct
	var responseData []ResponseData
	err = json.Unmarshal(body, &responseData)
	if err != nil {
		return "", fmt.Errorf("failed to unmarshal response data: %v", err)
	}

	for _, item := range responseData {
		if item.AppsheetWCID == wc_id {
			return item.AppsheetProductID, nil
		}
	}

	return "", errors.New("product searched correctly but not found in database")
}

func postMovement(product_id string, quantity int, platform string) (string, error) {

	payload := fmt.Sprintf(`
	{
		"Action": "Add",
		"Properties": {
			"Locale": "es-US",
			"Timezone": "Argentina Standard Time",
		},
		"Rows": [
			{
				"product_id": %s,
				"oran": %s,
				"movement_type": "%s"
			}
		]
	}`, product_id, convertToString(-quantity), platform)
	// Create the request
	requestURL := fmt.Sprintf("https://api.appsheet.com/api/v2/apps/%s/tables/MOVEMENTS/Action", os.Getenv("appsheet_id"))
	req, err := http.NewRequest(http.MethodPost, requestURL, bytes.NewBufferString(payload))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %v", err)
	}

	// Set request headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("ApplicationAccessKey", os.Getenv("appsheet_key"))

	// Send the request
	client := http.DefaultClient
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	// Check the response status code
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return convertToString(resp.StatusCode), nil
}

func getProductTotalStock(product_id string) (string, error) {

	stockgetURL := fmt.Sprintf("https://api.appsheet.com/api/v2/apps/%s/tables/STOCK/Action", os.Getenv("appsheet_id"))
	find_in_stock := `{
		"Action": "Find",
		  "Properties": 
		  	{"Locale": "en-US",
			"Location": "47.623098, -122.330184",
			"Timezone": "Pacific Standard Time",
			"UserSettings": {
				"Option 1": "value1",
				"Option 2": "value2"}
			},
		"Rows": []}`
	get, err := http.NewRequest(http.MethodPost, stockgetURL, bytes.NewBufferString(find_in_stock))
	if err != nil {
		return "", err
	}

	get.Header.Set("Content-Type", "application/json")
	get.Header.Set("ApplicationAccessKey", os.Getenv("appsheet_key"))

	client := http.DefaultClient
	// Get product data
	appsresp, err := client.Do(get)

	if err != nil {
		return "", err
	}
	defer appsresp.Body.Close()

	if appsresp.StatusCode != http.StatusOK {
		return "", errors.New(convertToString(appsresp.StatusCode))
	}

	// Read the response body
	body, err := io.ReadAll(appsresp.Body)
	if err != nil {
		log.Fatal("error reading response body:", err)
	}

	// Define a struct to hold the response data
	var responseData []stockData

	// Unmarshal the JSON data into the struct
	err = json.Unmarshal(body, &responseData)
	if err != nil {
		log.Fatal("error unmarshaling response data:", err)
	}
	for _, item := range responseData {
		if item.Product == product_id {
			return item.Total, nil
		}
	}

	return "", errors.New("product not found in stock table")

}

func getPlatformsID(product_id string) (string, string, string) {

	stockgetURL := fmt.Sprintf("https://api.appsheet.com/api/v2/apps/%s/tables/PLATFORMS/Action", os.Getenv("appsheet_id"))
	find_in_stock := `{
		"Action": "Find",
		  "Properties": {"Locale": "en-US","Location": "47.623098, -122.330184","Timezone": "Pacific Standard Time","UserSettings": {"Option 1": "value1","Option 2": "value2"}},
		"Rows": []}`
	get, err := http.NewRequest(http.MethodPost, stockgetURL, bytes.NewBufferString(find_in_stock))
	if err != nil {
		return "", "", fmt.Sprintf("Error creating request to find the product ID and quantity: %s", err)
	}

	get.Header.Set("Content-Type", "application/json")
	get.Header.Set("ApplicationAccessKey", os.Getenv("appsheet_key"))

	client := http.DefaultClient
	// Get product data
	appsresp, err := client.Do(get)
	if err != nil {
		return "", "", fmt.Sprintf("error geting product in Appsheet: %s", err)
	}
	defer appsresp.Body.Close()

	if err != nil {
		return "", "", fmt.Sprintf("unexpected status code from Appsheet: %d", appsresp.StatusCode)
	}

	// Read the response body
	body, err := io.ReadAll(appsresp.Body)
	if err != nil {
		log.Fatal("error reading response body:", err)
	}

	// Define a struct to hold the response data
	var PlatformData []PlatformsData

	// Unmarshal the JSON data into the struct
	err = json.Unmarshal(body, &PlatformData)
	if err != nil {
		log.Fatal("error unmarshaling response data:", err)
	}

	for _, item := range PlatformData {
		if item.Product == product_id {
			if item.WCID != "0" || item.MeliID != "0" {
				return item.MeliID, item.WCID, ""
			}
			return "", "", "product not linked to any platform"
		}
	}

	return "", "", "product not found in platforms database"

}

func updateWC(wc_id string, field string, value interface{}) string {

	wcURL := fmt.Sprintf("https://www.energiaglobal.com.ar/wp-json/wc/v3/products/%s", fmt.Sprint(wc_id))
	wcPayload := fmt.Sprintf(`{"%s": %s}`, fmt.Sprint(field), fmt.Sprint(value))

	req, err := http.NewRequest(http.MethodPut, wcURL, bytes.NewBufferString(wcPayload))
	if err != nil {
		return "error creating request for WooCommerce:" + fmt.Sprint(err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(os.Getenv("wc_client"), os.Getenv("wc_secret"))

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

func updateMeli(meli_id string, field string, value interface{}) string {

	URL := fmt.Sprintf("https://api.mercadolibre.com/items/MLA%s", fmt.Sprint(meli_id))
	payload := fmt.Sprintf(`{"%s": %s}`, fmt.Sprint(field), fmt.Sprint(value))

	req, err := http.NewRequest(http.MethodPut, URL, bytes.NewBufferString(payload))
	if err != nil {
		return "error creating request for MELI:" + fmt.Sprint(err)
	}

	auth := "Bearer " + os.Getenv("MELI_ACCESS_TOKEN")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", auth)

	client := http.DefaultClient
	resp, err := client.Do(req)
	if err != nil {
		return "error updating product in MELI:" + fmt.Sprint(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "unexpected status code from MELI:" + fmt.Sprint(resp.StatusCode)
	}

	return ""
}

func productIDFromMeli(meli_id string) (string, error) {

	// Define the data struct for the response
	type ResponseData struct {
		AppsheetProductID string `json:"product_id"`
		AppsheetMeliID    string `json:"meli_id"`
	}

	// Prepare the payload for finding the product ID and quantity
	payload := `{
			"Action": "Find",
			"Properties": {
				"Locale": "es-US",
				"Timezone": "Argentina Standard Time",
			},
			"Rows": []
		}`

	//"Selector": "Filter("PLATFORMS", ISNOTBLANK([wc_id]))",

	// Create the request
	requestURL := fmt.Sprintf("https://api.appsheet.com/api/v2/apps/%s/tables/PLATFORMS/Action", os.Getenv("appsheet_id"))

	req, err := http.NewRequest(http.MethodPost, requestURL, bytes.NewBufferString(payload))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %v", err)
	}

	// Set request headers
	key := os.Getenv("appsheet_key")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("ApplicationAccessKey", key)

	// Send the request
	client := http.DefaultClient
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	// Check the response status code
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %v", err)
	}

	// Unmarshal the JSON data into the struct
	var responseData []ResponseData
	err = json.Unmarshal(body, &responseData)
	if err != nil {
		return "", fmt.Errorf("failed to unmarshal response data: %v", err)
	}

	for _, item := range responseData {
		if item.AppsheetMeliID == meli_id {
			return item.AppsheetProductID, nil
		}
	}

	return "", errors.New("product searched correctly but not found in database")
}
