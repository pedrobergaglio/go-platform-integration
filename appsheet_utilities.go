package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"
)

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

	// Retry loop for handling 500 responses
	maxRetries := 3
	for retry := 0; retry < maxRetries; retry++ {
		if appsresp.StatusCode == http.StatusInternalServerError {
			fmt.Println("received 500 response. retrying...")
			time.Sleep(2 * time.Second) // Wait before retrying
			appsresp, err = client.Do(get)
			if err != nil {
				return "", "", "", err
			}
		} else {
			break // Exit the retry loop for non-500 responses
		}
	}

	// Retry loop for handling 500 responses
	for retry := 0; retry < maxRetries; retry++ {
		if appsresp.StatusCode == 400 {
			fmt.Print(find_in_stock)
			fmt.Println("received 400 response. retrying...")
			time.Sleep(2 * time.Second) // Wait before retrying
			appsresp, err = client.Do(get)
			if err != nil {
				return "", "", "", err
			}
		} else {
			break // Exit the retry loop for non-500 responses
		}
	}

	if appsresp.StatusCode != http.StatusOK {

		//fmt.Println("getProductStock", appsresp.StatusCode)
		//fmt.Print(product_id)
		//log.Fatal(find_in_stock)
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
	err = json.Unmarshal(body, &PlatformData)
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
