package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

type AlepheePriceResponse struct {
	TotalItemsRetrieved int           `json:"total"`
	ResultItems         []ResultItems `json:"results"`
}

type ResultItems struct {
	ItemSKU   string      `json:"sku"`
	ItemPrice interface{} `json:"price"`
}

// updateRumboPricesAlephee gets all the products in appsheet linked to alephee
// For each product, gets the price in alephee, and then updates it in appsheet.
func updateRumboPricesAlephee() {

	log.Println("updating rumbo prices")

	//for each product, get the price and update it in the app
	//if index in a multiple of 30, sleep 1 minute

	//

	// Define the data struct for the response
	type ResponseData struct {
		ID        string `json:"product_id"`
		AlepheeID string `json:"alephee_id"`
	}

	//*******************************
	//get products with alephee codes
	//*******************************

	// Prepare the payload for finding the product ID and quantity
	payload := `{
		"Action": "Find",
		"Properties": {
			"Locale": "es-US",
			"Selector": "Filter(PLATFORMS, [alephee_id]<>0)",
			"Timezone": "Argentina Standard Time"
		},
		"Rows": []
		}`

	// Create the request
	requestURL := fmt.Sprintf("https://api.appsheet.com/api/v2/apps/%s/tables/PLATFORMS/Action",
		os.Getenv("appsheet_id"))
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

	log.Println("send first request to appsheet")

	// Check the response status code
	if resp.StatusCode != http.StatusOK {
		log.Printf("couldn't get products with alephee code: unexpected status code from appsheet: %d", resp.StatusCode)
		return
	}

	log.Println("alephee products returned correctly")

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("failed to read response body: %v", err)
		return
	}

	// Unmarshal the JSON data into the struct
	var responseData []ResponseData
	err = json.Unmarshal(body, &responseData)
	if err != nil {
		log.Printf("failed to unmarshal response data: %v", err)
		return
	}

	//*******************************
	//for each product, get the price
	//*******************************

	requestCounter := 0

	for _, item := range responseData {

		requestCounter++

		// Check if the counter is a multiple of 30
		if requestCounter == 29 {
			//Sleep
			time.Sleep(1*time.Minute + 5*time.Second)
			// Reset the counter and update the last request time
			requestCounter = 0
		}

		//get the sale_price from alephee

		URL := fmt.Sprintf(
			"https://api.alephcrm.com/v2/productlistings/search?API_KEY=%s&accountId=%s&sku=%s",
			os.Getenv("alephee_api_key"),
			os.Getenv("alephee_account_id"),
			item.AlepheeID)

		req, err := http.NewRequest(http.MethodGet, URL, nil)
		if err != nil {
			log.Println("failed to create request:", err)
			return
		}

		client := http.DefaultClient
		resp, err := client.Do(req)
		if err != nil {
			log.Println(item.ID, "error getting product price:", err)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode == 429 {
			fmt.Println("error too many requests. waiting for 1 minute")
			time.Sleep(1*time.Minute + 5*time.Second) // Wait for 1 minute and 5 seconds
			updateRumboPricesAlephee()
			return
		}

		if resp.StatusCode != http.StatusOK {
			log.Println(item.AlepheeID, "unexpected status code from alephee while getting product price: "+fmt.Sprint(resp.StatusCode))
			continue
		}

		// Read the request body
		body, err = io.ReadAll(resp.Body)
		if err != nil {
			log.Println("error reading order request body:", err)
			return
		}

		var response AlepheePriceResponse
		err = json.Unmarshal(body, &response)
		if err != nil {
			log.Println("error unmarshaling order request body:", err)
			return
		}

		if response.TotalItemsRetrieved == 0 {
			log.Println(item.ID, "error: product not found in alephee")
			continue
		}

		if response.TotalItemsRetrieved > 1 {
			log.Println(item.ID, "possible error: more than one product found in alephee")
		}

		// Iterate over the results to get the product price
		sale_price := ""
		for _, orderItem := range response.ResultItems {
			if orderItem.ItemSKU == item.AlepheeID {
				sale_price = convertToString(orderItem.ItemPrice)
				break
			}
		}

		if sale_price == "" {
			log.Println(item.ID, "error: not sale price assigned")
			continue
		}

		//*******************************
		//for each product, update the price in the app
		//*******************************

		payload := fmt.Sprintf(`
		{
			"Action": "Edit",
			"Properties": {
				"Locale": "es-US",
				"Timezone": "Argentina Standard Time"
			},
			"Rows": [
				{
					"product_id": %s,
					"sale_price": %s
				}
			]
		}`, item.ID, sale_price)
		// Create the request
		requestURL := fmt.Sprintf("https://api.appsheet.com/api/v2/apps/%s/tables/platforms/Action", os.Getenv("appsheet_id"))
		req, err = http.NewRequest(http.MethodPost, requestURL, bytes.NewBufferString(payload))
		if err != nil {
			log.Printf("failed to create request: %v", err)
			return

		}

		// Set request headers
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("ApplicationAccessKey", os.Getenv("appsheet_key"))

		// Send the request
		client = http.DefaultClient
		resp, err = client.Do(req)
		if err != nil {
			log.Printf("failed to send request: %v", err)
			return
		}
		defer resp.Body.Close()

		// Check the response status code
		if resp.StatusCode != http.StatusOK {
			log.Printf("%s unexpected status code from appsheet when updating price: %d", item.ID, resp.StatusCode)
			continue
		}

	}

	log.Println("rumbo prices updated correctly")

}

/*
https://api.alephcrm.com/v2/products?API_KEY=8F509A97-B5C8-4B9E-8148-07C055C54C05&accountId=3319

[
    {
      "sku": "1025700",
      "stock": {
        "quantity": 2}
    }
  ]
*/

// Updates the stock of an alephee publication
func updateAlephee(alephee_id string, stock interface{}) string {

	URL := fmt.Sprintf("https://api.alephcrm.com/v2/products?API_KEY=%s&accountId=%s", os.Getenv("alephee_api_key"), os.Getenv("alephee_account_id"))
	payload := fmt.Sprintf(`
	[
		{
		  "sku": "%s",
		  "stock": {
			"quantity": %s
			}
		}
	  ]`, fmt.Sprint(alephee_id), fmt.Sprint(convertToString(stock)))

	req, err := http.NewRequest(http.MethodPost, URL, bytes.NewBufferString(payload))
	if err != nil {
		return "error creating request for alephee:" + fmt.Sprint(err)
	}

	req.Header.Set("Content-Type", "application/json")

	client := http.DefaultClient
	resp, err := client.Do(req)
	if err != nil {
		return "error updating product in alephee:" + fmt.Sprint(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != 412 && resp.StatusCode != 429 {
		return "unexpected status code from alephee:" + fmt.Sprint(resp.StatusCode)
	}

	if resp.StatusCode == 412 {
		return "412: no changes processed, possibly sku (alephee_id) not found"
	}

	if resp.StatusCode == 429 {
		fmt.Println("error too many requests. waiting for 1 minute and 5 seconds...")
		time.Sleep(1*time.Minute + 5*time.Second) // Wait for 1 minute and 5 seconds
		return updateAlephee(alephee_id, stock)
	}

	return ""
}
