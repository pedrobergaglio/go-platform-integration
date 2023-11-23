package main

/*
import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/spf13/viper"
)

type AlepheePriceResponse struct {
	TotalItemsRetrieved int          `json:"Total"`
	Results             []ResultItem `json:"Results"`
}

type ResultItem struct {
	Identification Identification `json:"Identification"`
	Price          []Price        `json:"Price"`
}

type Identification struct {
	SKU string `json:"SKU"`
}

type Price struct {
	Cost                       float64 `json:"Cost"`
	FinalPriceWithShippingCost float64 `json:"FinalPriceWithShippingCost"`
}

// Converts a number or string into string
func convertToString(value interface{}) string {
	switch v := value.(type) {
	case string:
		return v
	case int, int64, float32, float64:
		return fmt.Sprintf("%v", v)
	default:
		return ""
	}
}
func loadConfig() {
	viper.SetConfigFile(".env") // Set the path to your configuration file
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

// updateRumboPricesAlephee gets all the products in appsheet linked to alephee
// For each product, gets the price in alephee, and then updates it in appsheet.
func main() {
	loadConfig()

	log.Println("updating rumbo prices from alephee")

	//for each product, get the price and update it in the app
	//if index in a multiple of 30, sleep 1 minute

	//

	// Define the data struct for the response
	type ResponseData struct {
		ID        string `json:"product_id"`
		AlepheeID string `json:"platform_id"`
	}

	//*******************************
	//get products with alephee codes from appsheet
	//*******************************

	// Prepare the payload for finding the product ID and quantity
	payload := `{
		"Action": "Find",
		"Properties": {
			"Locale": "es-US",
			"Selector": 'Filter(PLATFORMS, [platform]="ALEPHEE")',
			"Timezone": "Argentina Standard Time"
		},
		"Rows": []
		}`

	// Create the request
	requestURL := fmt.Sprintf("https://api.appsheet.com/api/v2/apps/%s/tables/platforms/Action",
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

	// Check the response status code
	if resp.StatusCode != http.StatusOK {
		log.Printf("couldn't get products with alephee code: unexpected status code from appsheet: %d", resp.StatusCode)
		return
	}

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

		// Convert the total_stock value to an integer
		alephee_id_int, err := strconv.Atoi(item.AlepheeID)
		if err != nil {
			fmt.Println("error converting publication id to int:", err)
			return
		}
		// Format the total_stock with leading zeros (7 characters)
		item.AlepheeID = fmt.Sprintf("%07d", alephee_id_int)

		//get the sale_price from alephee

		URL := fmt.Sprintf(
			"https://api.alephcrm.com/v2/products?API_KEY=%s&accountId=%s&sku=%s",
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

		// RARO volver a llamar a toda la funcion
		if resp.StatusCode == 429 {
			fmt.Println("error too many requests. waiting for 1 minute")
			time.Sleep(1*time.Minute + 5*time.Second) // Wait for 1 minute and 5 seconds
			//updateRumboPricesAlephee()
			//return
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
			log.Println(item.ID, "possible error: more than one product found in alephee for this code")
		}

		// Iterate over the results to get the product price
		sale_price := ""
		cost_price := ""
		for _, orderItem := range response.Results {
			if orderItem.Identification.SKU == item.AlepheeID {
				sale_price = convertToString(orderItem.Price[0].FinalPriceWithShippingCost)
				cost_price = convertToString(orderItem.Price[0].Cost)
				break
			}
		}

		if sale_price == "" || cost_price == "" {
			log.Println(item.ID, "error: not sale or cost price assigned")
			continue
		}

		// Get the siva_cost price in usd
		cost_int, err := strconv.ParseFloat(cost_price, 64)
		if err != nil {
			log.Println("error converting cost to int:", err)
			return
		}
		// Format the total_stock with leading zeros (7 characters)
		cost_price = fmt.Sprintf("%.2f", (float64(cost_int) / float64(470)))

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
					"sale_price_ars": %s,
					"siva_cost": %s
				}
			]
		}`, item.ID, sale_price, cost_price)
		// Create the request
		requestURL := fmt.Sprintf("https://api.appsheet.com/api/v2/apps/%s/tables/stock/Action", os.Getenv("appsheet_id"))
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

		log.Println(item.ID, "done")

	}

	log.Println("finished obtaining rumbo prices")

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

/*
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
*/
