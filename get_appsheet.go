package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"

	"log"
	"net/http"
	"os"
)

/*func getProductData(table, product_id, returnColumn string) (string, error) {
	// Define the data struct for the response
	type ResponseData struct {
		Total    string `json:"total_stock"`
		WcCodigo string `json:"wc_code"`
	}

	// Prepare the payload for finding the product ID and quantity
	payload := fmt.Sprintf(`{
		"Action": "Find",
		"Properties": {
			"Locale": "en-US",
			"Location": "47.623098, -122.330184",
			"Timezone": "Pacific Standard Time",

			"UserSettings": {
				"Option 1": "value1",
				"Option 2": "value2"
			}
		},
		"Rows": []
	}`)

	//"Selector": "Filter("PLATFORMS", ISNOTBLANK([wc_id]))",
	// Create the request
	requestURL := fmt.Sprintf("https://api.appsheet.com/api/v2/apps/%s/tables/%s/Action", Getenv("APPSHEET_ID"), table)
	req, err := http.NewRequest(http.MethodPost, requestURL, bytes.NewBufferString(payload))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %v", err)
	}

	// Set request headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("ApplicationAccessKey", Getenv("APPSHEET_KEY"))

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
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %v", err)
	}

	// Unmarshal the JSON data into the struct
	var responseData []ResponseData
	err = json.Unmarshal(body, &responseData)
	if err != nil {
		return "", fmt.Errorf("failed to unmarshal response data: %v", err)
	}

	// Access the first element and extract the value
	switch returnColumn {
	case "total_stock":
		return responseData[0].Total, nil
	case "wc_codigo":
		return responseData[0].WcCodigo, nil
	default:
		return "", fmt.Errorf("unknown return column: %s", returnColumn)
	}
}*/

func getProductID(wc_id string) (string, error) {

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
	body, err := ioutil.ReadAll(resp.Body)
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
	for _, item := range responseData {
		if item.Product == product_id {
			return item.Total, nil
		}
	}

	return "", errors.New("product not found in stock table")

}

func getWCID(product_id string) (string, string) {

	stockgetURL := fmt.Sprintf("https://api.appsheet.com/api/v2/apps/%s/tables/PLATFORMS/Action", os.Getenv("appsheet_id"))
	find_in_stock := `{
		"Action": "Find",
		  "Properties": {"Locale": "en-US","Location": "47.623098, -122.330184","Timezone": "Pacific Standard Time","UserSettings": {"Option 1": "value1","Option 2": "value2"}},
		"Rows": []}`
	get, err := http.NewRequest(http.MethodPost, stockgetURL, bytes.NewBufferString(find_in_stock))
	if err != nil {
		return "", fmt.Sprintf("Error creating request to find the product ID and quantity: %s", err)
	}

	get.Header.Set("Content-Type", "application/json")
	get.Header.Set("ApplicationAccessKey", os.Getenv("appsheet_key"))

	client := http.DefaultClient
	// Get product data
	appsresp, err := client.Do(get)
	if err != nil {
		return "", fmt.Sprintf("Error geting product in Appsheet: %s", err)
	}
	defer appsresp.Body.Close()

	if err != nil {
		return "", fmt.Sprintf("Unexpected status code from Appsheet: %d", appsresp.StatusCode)
	}

	// Read the response body
	body, err := ioutil.ReadAll(appsresp.Body)
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

	for _, item := range PlatformData {
		if item.Product == product_id {
			return item.WooID, ""
		}
	}

	return "", "Product ID searched correctly in PLATFORMS but not found"

}
