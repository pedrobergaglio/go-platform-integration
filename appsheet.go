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
)

// handleASMovementWebhook receives a product_id which stock has been modified
// Then the function obtains the product stock in Oran, calculates the configured product stock margin
// and then updates that stock value in the online sales platforms for that specific product
type ASMovementWebhookPayload struct {
	ProductID interface{} `json:"product_id"`
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

	//Get platforms ids
	stock_scope, stock_margin, alephee_id, meli_id, wc_id, error := getPlatformsID(convertToString(payload.ProductID))
	if error != "" {
		log.Println(error)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	//Get product data
	total := ""
	if stock_scope == "N" {
		total, err = getProductStock(convertToString(payload.ProductID), "Orán")
		if err != nil {
			log.Println("error getting product total stock:", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	} else if stock_scope == "Y" {
		total, err = getProductStock(convertToString(payload.ProductID), "Total")
		if err != nil {
			log.Println("error getting product total stock:", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
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
	if meli_id != "0" {
		if alephee_id == "0000000" {
			//if meli_id != "0" {
			error = updateMeli(meli_id, "available_quantity", stock_minus_margin)
			if error != "" {
				log.Println("error updating stock in meli:", error)
				flag = 1
			}
		}
	} else {
		//log.Println("product not linked to meli")
	}

	// Update the ALEPHEE product
	if alephee_id != "0000000" {
		error = updateAlephee(alephee_id, stock_minus_margin)
		if error != "" {
			log.Println("error updating stock in alephee:", error)
			flag = 1
		}
	} else {
		//log.Println("product not linked to alephee")
	}

	// Update the WooCommerce product
	if wc_id != "0" {
		error = updateWC(wc_id, "stock_quantity", stock_minus_margin)
		if error != "" {
			log.Println("error updating stock in WC:", error)
			flag = 1
		}
	} else {
		//log.Println("product not linked to wc")
	}

	if flag == 1 {
		return
	}

	// Write a success response if everything is processed successfully
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("webhook processed successfully"))
	log.Println("movement processed correctly")

}

// handleASPriceWebhook receives a product data with a price value that has been modified
// Then the function calculates the configured price margin for meli,
// and updates the prices in the online platforms for that specific product
type ASPriceWebhookPayload struct {
	ProductID       string `json:"product_id"`
	SalePrice       string `json:"sale_price"`
	WCID            string `json:"wc_id"`
	AlepheeID       string `json:"alephee_id"`
	MeliID          string `json:"meli_id"`
	MeliPriceMargin string `json:"meli_price_margin"`
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

	sale_price, err := strconv.ParseFloat(payload.SalePrice, 64)
	if err != nil {
		log.Printf("error parsing sale price to float: %v", err)
		return
	}

	if sale_price < 2500.0 {
		log.Println("price not updated: price too small (<$2500)")
		return
	}

	//set it to len=7

	// Convert the total_stock value to an integer
	alephee_id_int, err := strconv.Atoi(payload.AlepheeID)
	if err != nil {
		return
	}
	// Format the total_stock with leading zeros (7 characters)
	formatted_alephee_id := fmt.Sprintf("%07d", alephee_id_int)

	// Log the received payload
	log.Println("notification for updated price received. product:", convertToString(payload.ProductID), "price:", convertToString(payload.SalePrice))

	// Update the MELI product

	flag := 0

	if convertToString(payload.MeliID) != "0" {

		if convertToString(formatted_alephee_id) == "0000000" {

			margin, err := strconv.Atoi(convertToString(payload.MeliPriceMargin))
			if err != nil {
				log.Printf("error parsing margin to float: %v", err)
				return
			}

			//calculate the margin
			percent_margin := (1 + margin/100)
			meli_price := sale_price * float64(percent_margin)

			//set the last digit to 0
			string_meli_price := convertToString(meli_price)
			length := len(string_meli_price)

			errr := updateMeli(convertToString(payload.MeliID), "price", string_meli_price[:length-1]+"0")
			if errr != "" {
				log.Println("error updating meli price:", errr)
				flag = 1
			}
		}
	} else {
		//log.Println("product not linked to meli")
	}

	// Update the WooCommerce product
	if convertToString(payload.WCID) != "0" {
		errr := updateWC(convertToString(payload.WCID), "regular_price", `"`+payload.SalePrice+`"`)
		if errr != "" {
			log.Println("error updating Woocommerce price:", errr)
			flag = 1
		}
	} else {
		//log.Println("product not linked to wc")
	}

	// Write a success response if everything is processed successfully
	if flag == 0 {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("webhook processed successfully"))
		//log.Println("price updated")
	}

}

// handleASCountingWebhook gets the items to count of the user that saved the counting.
// Then compares each item to the value in the location of the counting
// Adds movements to set the stock value equal to the quantity counted by the user
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

	log.Println("counting added:", counting.ID, counting.Datetime, counting.Location, counting.User)

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

	// Check the response status code
	if resp.StatusCode != http.StatusOK {
		log.Printf("unexpected status code from appsheet: %d", resp.StatusCode)
		return
	}

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
		} else if counting.Location == "Orán" {
			addMovement(item.ID, "0", quantitystr, "0", "0", movement_type)
		} else if counting.Location == "Rodriguez" {
			addMovement(item.ID, "0", "0", quantitystr, "0", movement_type)
		} else if counting.Location == "Marcos Paz" {
			addMovement(item.ID, "0", "0", "0", quantitystr, movement_type)
		} else {
			log.Println("movement not added")
		}

	}
}

// productIDFromWCID lookups the id of a product based on the wc id
func productIDFromWCID(wc_id string) (string, error) {

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

// Adds a movement in appsheet with the product_id, stock in each location, and movement_type
func addMovement(product_id string, fabrica string, oran string, rodriguez string, marcos_paz string, movement_type string) (string, error) {

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
	}`, product_id, fabrica, oran, rodriguez, marcos_paz, movement_type)
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

type stockData struct {
	Oran      string `json:"oran"`
	Rodriguez string `json:"rodriguez"`
	Fabrica   string `json:"fabrica"`
	MarcosPaz string `json:"marcos_paz"`
	Total     string `json:"total_stock"`
	Product   string `json:"product_id"`
}

// Returns the stock of the product in the location specified
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

type PlatformsData struct {
	Product     string `json:"product_id"`
	WCID        string `json:"wc_id"`
	MeliID      string `json:"meli_id"`
	AlepheeID   string `json:"alephee_id"`
	StockMargin string `json:"stock_margin"`
	StockScope  string `json:"stock_scope"`
}

// Returns the StockMargin, AlepheeID, MeliID, WCID based on a product_id
func getPlatformsID(product_id string) (string, string, string, string, string, string) {

	stockgetURL := fmt.Sprintf("https://api.appsheet.com/api/v2/apps/%s/tables/PLATFORMS/Action", os.Getenv("appsheet_id"))
	find_in_stock := `{
		"Action": "Find",
		  "Properties": {"Locale": "en-US","Location": "47.623098, -122.330184","Timezone": "Pacific Standard Time","UserSettings": {"Option 1": "value1","Option 2": "value2"}},
		"Rows": []}`
	get, err := http.NewRequest(http.MethodPost, stockgetURL, bytes.NewBufferString(find_in_stock))
	if err != nil {
		return "", "", "", "", "", fmt.Sprintf("Error creating request to find the product ID and quantity: %s", err)
	}

	get.Header.Set("Content-Type", "application/json")
	get.Header.Set("ApplicationAccessKey", os.Getenv("appsheet_key"))

	client := http.DefaultClient
	// Get product data
	appsresp, err := client.Do(get)
	if err != nil {
		return "", "", "", "", "", fmt.Sprintf("error geting product in Appsheet: %s", err)
	}
	defer appsresp.Body.Close()

	if err != nil {
		return "", "", "", "", "", fmt.Sprintf("unexpected status code from Appsheet: %d", appsresp.StatusCode)
	}

	// Read the response body
	body, err := io.ReadAll(appsresp.Body)
	if err != nil {
		log.Fatal("error reading response body:", err)
	}

	// Define a struct to hold the response data
	var PlatformData []PlatformsData

	// Unmarshal the JSON data into the struct
	err = json.Unmarshal([]byte(body), &PlatformData)
	if err != nil {
		log.Fatal("error unmarshaling response data:", err)
	}

	for _, item := range PlatformData {
		if item.Product == product_id {
			if item.WCID != "0" || item.MeliID != "0" {
				//set it to len=7

				// Convert the total_stock value to an integer
				alephee_id_int, err := strconv.Atoi(item.AlepheeID)
				if err != nil {
					return "", "", "", "", "", fmt.Sprintln("Error converting total_stock to int:", err)
				}
				// Format the total_stock with leading zeros (7 characters)
				formatted_alephee_id := fmt.Sprintf("%07d", alephee_id_int)

				return item.StockScope, item.StockMargin, formatted_alephee_id, item.MeliID, item.WCID, ""
			}
			return "", "", "", "", "", "product not linked to any platform"
		}
	}

	return "", "", "", "", "", "product not found in platforms database"

}

// productIDFromMeliID lookups the id of a product based on the meli id
func productIDFromMeliID(meli_id string) (string, error) {

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
