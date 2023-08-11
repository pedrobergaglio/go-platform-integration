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
type ASProductIDWebhookPayload struct {
	ProductID string `json:"product_id"`
}

func handleASMovementWebhook(w http.ResponseWriter, r *http.Request) {
	// Ensure that the request method is POST
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	// Parse the incoming request body
	var payload ASProductIDWebhookPayload
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
	ProductPlatformsData, error := getPlatformsID(convertToString(payload.ProductID))
	if error != "" {
		log.Println(error)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	//Get product data
	total := ""
	_, stock_margin, total, err := getProductStock(convertToString(payload.ProductID), "")
	if err != nil {
		log.Println("error getting product total stock:", err)
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

	for _, item := range ProductPlatformsData {

		// Update the MELI product
		if item.Platform == "MELI" {

			error = updateMeli(item.PlatformID, "available_quantity", stock_minus_margin)
			if error != "" {
				log.Println("error updating stock in meli:", error)
				flag = 1
			}

			// Update the ALEPHEE product
		} else if item.Platform == "ALEPHEE" {
			error = updateAlephee(item.PlatformID, stock_minus_margin)
			if error != "" {
				log.Println("error updating stock in alephee:", error)
				flag = 1
			}

			// Update the WooCommerce product
		} else if item.Platform == "WC" {
			error = updateWC(item.PlatformID, "stock_quantity", stock_minus_margin)
			if error != "" {
				log.Println("error updating stock in wc:", error)
				flag = 1
			}
		}

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

func handleASPriceWebhook(w http.ResponseWriter, r *http.Request) {
	// Ensure that the request method is POST
	if r.Method != http.MethodPost {
		log.Println(http.StatusMethodNotAllowed)
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	// Parse the incoming request body
	var payload ASProductIDWebhookPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		log.Println("error decoding payload:", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	ProductPlatformsData, err := getPlatformsID(payload.ProductID)
	if err != "" {
		log.Printf("getPlatformsID error getting platforms product data: %s", err)
		return
	}

	sale_pricestr, _, _, errr := getProductStock(payload.ProductID, "")
	if errr != nil {
		log.Printf("getProductStock error getting platforms product data: %v", errr)
		return
	}

	sale_price, errr := strconv.ParseFloat(sale_pricestr, 64)
	if errr != nil {
		log.Printf("error parsing sale price to float: %v", errr)
		return
	}

	// Log the received payload
	log.Println("notification for updated price received. product:", convertToString(payload.ProductID), "price:", convertToString(sale_price))

	if sale_price < 2500.0 {
		log.Println("price not updated: price too small (<$2500)")
		return
	}

	//flag := 0

	for _, item := range ProductPlatformsData {

		// Update the MELI product

		if item.Platform == "MELI" {

			margin, err := strconv.Atoi(item.MeliPriceMargin)
			if err != nil {
				log.Printf("error parsing price margin to float: %v", err)
				return
			}

			//calculate the margin
			percent_margin := (1.0 + float64(margin)/100.00)
			meli_price := sale_price * float64(percent_margin)

			//set the last digit to 0
			string_meli_price := convertToString(int(meli_price))
			length := len(string_meli_price)
			string_meli_price = string_meli_price[:length-1] + "0"

			errr := updateMeli(convertToString(item.PlatformID), "price", string_meli_price)
			if errr != "" {
				log.Println("error updating meli price:", errr)
				//flag = 1
			}

			// Update the WooCommerce product
		} else if item.Platform == "WC" {
			errr := updateWC(convertToString(item.PlatformID), "regular_price", `"`+sale_pricestr+`"`)
			if errr != "" {
				log.Println("error updating woocommerce price:", errr)
				//flag = 1
			}
		}

	}

	/* Write a success response if everything is processed successfully
	if flag == 0 {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("webhook processed successfully"))
		log.Println("price updated")
	}*/

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

	// Ensure that the request method is POSTh
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
		_, _, stock_value, err := getProductStock(item.ID, counting.Location)
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
	Oran        string `json:"oran"`
	Rodriguez   string `json:"rodriguez"`
	Fabrica     string `json:"fabrica"`
	MarcosPaz   string `json:"marcos_paz"`
	Total       string `json:"total_stock"`
	Product     string `json:"product_id"`
	StockScope  string `json:"stock_scope"`
	StockMargin string `json:"stock_margin"`
	SalePrice   string `json:"sale_price_ars"`
}

// Returns the stock of the product in the location specified. If location is "" returns the corresponding scope stock.
func getProductStock(product_id string, location string) (sale_price, stock_margin, stock string, err error) {

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
		return "", "", "", err
	}

	get.Header.Set("Content-Type", "application/json")
	get.Header.Set("ApplicationAccessKey", os.Getenv("appsheet_key"))

	client := http.DefaultClient
	// Get product data
	appsresp, err := client.Do(get)

	if err != nil {
		return "", "", "", err
	}
	defer appsresp.Body.Close()

	if appsresp.StatusCode != http.StatusOK {
		fmt.Print(find_in_stock)
		return "", "", "", errors.New(convertToString(appsresp.StatusCode))
	}

	// Read the response body
	body, err := io.ReadAll(appsresp.Body)
	if err != nil {
		return "", "", "", fmt.Errorf("error reading response body: %v", err)
	}

	// Define a struct to hold the response data
	var responseData []stockData

	// Unmarshal the JSON data into the struct
	err = json.Unmarshal(body, &responseData)
	if err != nil {
		return "", "", "", fmt.Errorf("error unmarshaling response data: %v", err)
	}
	for _, item := range responseData {
		if item.Product == product_id {

			sale_price = item.SalePrice
			stock_margin = item.StockMargin
			err = nil

			if location == "Orán" {
				stock = item.Oran
				return
			} else if location == "Rodriguez" {
				stock = item.Rodriguez
				return
			} else if location == "Marcos Paz" {
				stock = item.MarcosPaz
				return
			} else if location == "Fábrica" {
				stock = item.Fabrica
				return
			} else if location == "Total" {
				stock = item.Total
				return
			} else if location == "" {

				if item.StockScope == "N" {
					stock = item.Oran
					return
				}
				if item.StockScope == "Y" {
					stock = item.Total
					return
				}
			}
		}
	}

	return "", "", "", errors.New("product not found in stock table")

}

type PlatformsData struct {
	ProductID       string `json:"product_id"`
	PlatformID      string `json:"platform_id"`
	Platform        string `json:"platform"`
	MeliPriceMargin string `json:"meli_price_margin"`
}

// Returns the StockMargin, AlepheeID, MeliID, WCID based on a product_id
func getPlatformsID(product_id string) ([]PlatformsData, string) {

	var ProductPlatformsData []PlatformsData

	stockgetURL := fmt.Sprintf("https://api.appsheet.com/api/v2/apps/%s/tables/PLATFORMS/Action", os.Getenv("appsheet_id"))
	find_in_stock := `{
		"Action": "Find",
		  "Properties": {"Locale": "en-US","Location": "47.623098, -122.330184","Timezone": "Pacific Standard Time","UserSettings": {"Option 1": "value1","Option 2": "value2"}},
		"Rows": []}`
	get, err := http.NewRequest(http.MethodPost, stockgetURL, bytes.NewBufferString(find_in_stock))
	if err != nil {
		return ProductPlatformsData, fmt.Sprintf("Error creating request to find the product ID and quantity: %s", err)
	}

	get.Header.Set("Content-Type", "application/json")
	get.Header.Set("ApplicationAccessKey", os.Getenv("appsheet_key"))

	client := http.DefaultClient
	// Get product data
	appsresp, err := client.Do(get)
	if err != nil {
		return ProductPlatformsData, fmt.Sprintf("error getting product in Appsheet: %s", err)
	}
	defer appsresp.Body.Close()

	if err != nil {
		return ProductPlatformsData, fmt.Sprintf("unexpected status code from Appsheet: %d", appsresp.StatusCode)
	}

	// Read the response body
	body, err := io.ReadAll(appsresp.Body)
	if err != nil {
		return ProductPlatformsData, fmt.Sprintln("error reading response body:", err)

	}

	// Define a struct to hold the response data
	var PlatformData []PlatformsData

	// Unmarshal the JSON data into the struct
	err = json.Unmarshal([]byte(body), &PlatformData)
	if err != nil {
		fmt.Print(body)
		fmt.Print(product_id)
		return ProductPlatformsData, fmt.Sprintln("error unmarshaling response data:", err)
	}

	for _, item := range PlatformData {
		if item.ProductID == product_id {
			if item.Platform == "ALEPHEE" {
				//set it to len=7

				// Convert the total_stock value to an integer
				alephee_id_int, err := strconv.Atoi(item.PlatformID)
				if err != nil {
					return ProductPlatformsData, fmt.Sprintln("error converting publication id to int:", err)
				}
				// Format the total_stock with leading zeros (7 characters)
				item.PlatformID = fmt.Sprintf("%07d", alephee_id_int)
			}

			ProductPlatformsData = append(ProductPlatformsData, item)

		}
	}

	if len(ProductPlatformsData) == 0 {
		return ProductPlatformsData, "product not found in platforms database"
	}

	return ProductPlatformsData, ""

}
