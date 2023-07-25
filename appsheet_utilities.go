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
	"strconv"
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
	ProductID       string `json:"product_id"`
	SalePrice       string `json:"sale_price"`
	WCID            string `json:"wc_id"`
	AlepheeID       string `json:"alephee_id"`
	MeliID          string `json:"meli_id"`
	MeliPriceMargin string `json:"meli_price_margin"`
}

type stockData struct {
	Oran      string `json:"oran"`
	Rodriguez string `json:"rodriguez"`
	Fabrica   string `json:"fabrica"`
	MarcosPaz string `json:"marcos_paz"`
	Total     string `json:"total_stock"`
	Product   string `json:"product_id"`
}

type PlatformsData struct {
	Product     string `json:"product_id"`
	WCID        string `json:"wc_id"`
	MeliID      string `json:"meli_id"`
	AlepheeID   string `json:"alephee_id"`
	StockMargin string `json:"stock_margin"`
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

	product_id, err := strconv.Atoi(convertToString(payload.ProductID))
	if err != nil {
		log.Println("error passing product_id to int:", err)
		return
	}

	if product_id == 0 {
		log.Println("error: movement received with product_id:", payload.ProductID)
		return
	} else {
		// Log the received payload
		log.Println("notification for movement in product_id:", payload.ProductID)
	}

	//Wait 3 seconds
	//log.Println("Waiting two seconds to update...")
	time.Sleep(2 * time.Second)

	//Get product data
	total, err := getProductStock(convertToString(payload.ProductID), "Orán")
	if err != nil {
		log.Println("error getting product total stock:", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	//Get platforms ids
	stock_margin, alephee_id, meli_id, wc_id, error := getPlatformsID(convertToString(payload.ProductID))
	if error != "" {
		log.Println(error)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	//Compute and format stock with margin substracted
	totalint, err := strconv.Atoi(total)
	if err != nil {
		log.Println("error passing string to int", err)
		return
	}
	marginint, err := strconv.Atoi(stock_margin)
	if err != nil {
		log.Println("error passing string to int", err)
		return
	}
	stock_minus_marginint := totalint - marginint

	if stock_minus_marginint < 0 {
		stock_minus_marginint = 0
	}

	stock_minus_margin := convertToString(stock_minus_marginint)

	// Print the values
	//log.Println("Woocommerce ID:", woo_id)

	flag := 0

	// Update the MELI product
	if meli_id != "0" && alephee_id == "0" {
		//if meli_id != "0" {
		error = updateMeli(meli_id, "available_quantity", stock_minus_margin)
		if error != "" {
			log.Println("error updating stock in meli:", error)
			flag = 1
		}
	} else {
		log.Println("product not linked to meli")
	}

	// Update the ALEPHEE product
	if alephee_id != "0" {
		error = updateAlephee(alephee_id, stock_minus_margin)
		if error != "" {
			log.Println("error updating stock in alephee:", error)
			flag = 1
		}
	} else {
		log.Println("product not linked to alephee")
	}

	// Update the WooCommerce product
	if wc_id != "0" {
		error = updateWC(wc_id, "stock_quantity", stock_minus_margin)
		if error != "" {
			log.Println("error updating stock in WC:", error)
			flag = 1
		}
	} else {
		log.Println("product not linked to wc")
	}

	if flag == 1 {
		return
	}

	// Write a success response if everything is processed successfully
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("webhook processed successfully"))
	//log.Println("movement processed")

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
	log.Println("notification for updated price received:", convertToString(payload.SalePrice))

	// Update the MELI product

	flag := 0

	if convertToString(payload.MeliID) != "0" {

		if convertToString(payload.AlepheeID) == "0" {

			sale_price, err := strconv.ParseFloat(payload.SalePrice, 64)
			if err != nil {
				log.Printf("error parsing sale price to float: %v", err)
				return
			}

			margin, err := strconv.Atoi(convertToString(payload.MeliPriceMargin))
			if err != nil {
				log.Printf("error parsing margin to float: %v", err)
				return
			}

			percent_margin := (1 + margin/100)
			meli_price := sale_price * float64(percent_margin)

			errr := updateMeli(convertToString(payload.MeliID), "price", convertToString(meli_price))
			if errr != "" {
				log.Println("error updating meli price:", errr)
				flag = 1
			}
		}
	} else {
		log.Println("product not linked to meli")
	}

	// Update the WooCommerce product
	if convertToString(payload.WCID) != "0" {
		errr := updateWC(convertToString(payload.WCID), "regular_price", `"`+payload.SalePrice+`"`)
		if errr != "" {
			log.Println("error updating Woocommerce price:", errr)
			flag = 1
		}
	} else {
		log.Println("product not linked to wc")
	}

	// Write a success response if everything is processed successfully
	if flag == 0 {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("webhook processed successfully"))
		log.Println("price updated")
	}

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
				"Selector": "Filter(PLATFORMS, ISNOTBANK([wc_id]))"
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

func addMovement(product_id string, fabrica string, oran string, rodriguez string, marcos_paz string, platform string) (string, error) {

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
				"fabrica": %s,
				"oran": %s,
				"rodriguez": %s,
				"marcos_paz": %s,
				"movement_type": "%s"
			}
		]
	}`, product_id, fabrica, oran, rodriguez, marcos_paz, platform)
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

func getProductStock(product_id string, location string) (string, error) {

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
			if location == "Orán" {
				return item.Oran, nil
			}
			if location == "Rodriguez" {
				return item.Rodriguez, nil
			}
			if location == "Marcos Paz" {
				return item.MarcosPaz, nil
			}
			if location == "Fábrica" {
				return item.Fabrica, nil
			}
			if location == "Total" {
				return item.Total, nil
			}
		}
	}

	return "", errors.New("product not found in stock table")

}

func getPlatformsID(product_id string) (string, string, string, string, string) {

	stockgetURL := fmt.Sprintf("https://api.appsheet.com/api/v2/apps/%s/tables/PLATFORMS/Action", os.Getenv("appsheet_id"))
	find_in_stock := `{
		"Action": "Find",
		  "Properties": {"Locale": "en-US","Location": "47.623098, -122.330184","Timezone": "Pacific Standard Time","UserSettings": {"Option 1": "value1","Option 2": "value2"}},
		"Rows": []}`
	get, err := http.NewRequest(http.MethodPost, stockgetURL, bytes.NewBufferString(find_in_stock))
	if err != nil {
		return "", "", "", "", fmt.Sprintf("Error creating request to find the product ID and quantity: %s", err)
	}

	get.Header.Set("Content-Type", "application/json")
	get.Header.Set("ApplicationAccessKey", os.Getenv("appsheet_key"))

	client := http.DefaultClient
	// Get product data
	appsresp, err := client.Do(get)
	if err != nil {
		return "", "", "", "", fmt.Sprintf("error geting product in Appsheet: %s", err)
	}
	defer appsresp.Body.Close()

	if err != nil {
		return "", "", "", "", fmt.Sprintf("unexpected status code from Appsheet: %d", appsresp.StatusCode)
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
				return item.StockMargin, item.AlepheeID, item.MeliID, item.WCID, ""
			}
			return "", "", "", "", "product not linked to any platform"
		}
	}

	return "", "", "", "", "product not found in platforms database"

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

	if resp.StatusCode != http.StatusOK {
		return "unexpected status code from meli:" + fmt.Sprint(resp.StatusCode)
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
				"Selector": "Filter(PLATFORMS, ISNOTBANK([meli_id]))"
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

type ASCountingsWebhookPayload struct {
	ID       string `json:"id"`
	Datetime string `json:"datetime"`
	User     string `json:"user"`
	Location string `json:"location"`
}

type ASItemsToCountWebhookPayload struct {
	ID       string `json:"product_id"`
	Quantity string `json:"quantity"`
	User     string `json:"user"`
}

func handleASCountingWebhook(w http.ResponseWriter, r *http.Request) {

	log.Println("counting added")

	// Ensure that the request method is POST
	if r.Method != http.MethodPost {
		log.Println(http.StatusMethodNotAllowed)
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	// Parse the incoming request body
	var counting ASCountingsWebhookPayload
	if err := json.NewDecoder(r.Body).Decode(&counting); err != nil {
		log.Println("error decoding payload:", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	log.Println(counting.ID, counting.Datetime, counting.Location, counting.User)

	// Prepare the payload for finding the product ID and quantity
	payload := fmt.Sprintf(`{
		"Action": "Find",
		"Properties": {
			"Locale": "es-US",
			"Selector": 'Filter(items_to_count, [user]="%s")',
			"Timezone": "Argentina Standard Time",
		},
		"Rows": []
	}`, counting.User)

	// Create the request
	requestURL := fmt.Sprintf("https://api.appsheet.com/api/v2/apps/%s/tables/items_to_count/Action", os.Getenv("appsheet_id"))
	key := os.Getenv("appsheet_key")
	req, err := http.NewRequest(http.MethodPost, requestURL, bytes.NewBufferString(payload))
	if err != nil {
		log.Printf("failed to create request for appsheet: %v", err)
		return
	}

	// Set request headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("ApplicationAccessKey", key)

	// Send the request
	client := http.DefaultClient
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	log.Printf("request sent to appsheet")

	// Check the response status code
	if resp.StatusCode != http.StatusOK {
		log.Printf("unexpected status code from appsheet: %d", resp.StatusCode)
		return
	}

	log.Printf("request returned correctly")

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("failed to read response body: %v", err)
		return
	}

	// Unmarshal the JSON data into the struct
	var responseData []ASItemsToCountWebhookPayload
	err = json.Unmarshal(body, &responseData)
	if err != nil {
		log.Printf("failed to unmarshal response data: %v", err)
		return
	}

	movement_type := fmt.Sprintf("Conteo %s", counting.ID)

	for _, item := range responseData {
		stock_value, err := getProductStock(item.ID, counting.Location)
		if err != nil {
			log.Printf("error getting product stock: %v", err)
			return
		}

		if stock_value == item.Quantity {
			continue
		}

		old, err := strconv.Atoi(stock_value)
		if err != nil {
			log.Printf("error parsing stock quantity to int: %v", err)
			return
		}
		new, err := strconv.Atoi(item.Quantity)
		if err != nil {
			log.Printf("error parsing stock quantity to int: %v", err)
			return
		}
		quantity := new - old
		quantitystr := convertToString(quantity)

		if counting.Location == "Fábrica" {
			addMovement(item.ID, quantitystr, "0", "0", "0", movement_type)
			log.Println("movement added")
		} else if counting.Location == "Orán" {
			addMovement(item.ID, "0", quantitystr, "0", "0", movement_type)
			log.Println("movement added")
		} else if counting.Location == "Rodriguez" {
			addMovement(item.ID, "0", "0", quantitystr, "0", movement_type)
			log.Println("movement added")
		} else if counting.Location == "Marcos Paz" {
			addMovement(item.ID, "0", "0", "0", quantitystr, movement_type)
			log.Println("movement added")
		} else {
			log.Println("movement not added")
		}

	}

}
