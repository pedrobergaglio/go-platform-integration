package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"

	_ "github.com/go-sql-driver/mysql"
)

// handleASMovementWebhook receives a product_id which stock has been modified
// Then the function obtains the product stock in Oran, calculates the configured product stock margin
// and then updates that stock value in the online sales platforms for that specific product
type ASSupplierUSDWebhookPayload struct {
	Supplier    string `json:"supplier"`
	SupplierUSD string `json:"supplier_usd"`
}

type ASProductPricesSuppliers struct {
	ProductID string `json:"product_id"`
	Supplier  string `json:"supplier"`
	USDPrice  string `json:"sale_price_usd"`
	USDIva    string `json:"usd_price_iva"`
}

func handleASUsdWebhook(w http.ResponseWriter, r *http.Request) {

	// Ensure that the request method is POSTh
	if r.Method != http.MethodPost {
		log.Println(http.StatusMethodNotAllowed)
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	// Parse the incoming request body
	var supplier ASSupplierUSDWebhookPayload
	if err := json.NewDecoder(r.Body).Decode(&supplier); err != nil {
		log.Println("error decoding payload:", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	log.Println("supplier usd updated:", supplier.Supplier, supplier.SupplierUSD)

	//get the supplier products with usd price

	//format only one query with the update of all the prices for every product

	// Prepare the payload for finding the product ID and quantity
	payload := fmt.Sprintf(`{
		"Action": "Find",
		"Properties": {
			"Locale": "es-US",
			"Selector": 'Filter(stock, [supplier]="%s")',
			"Timezone": "Argentina Standard Time",
		},
		"Rows": []
	}`, supplier.Supplier)

	// Create the request
	requestURL := fmt.Sprintf("https://api.appsheet.com/api/v2/apps/%s/tables/stock/Action", os.Getenv("appsheet_id"))
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
	var ProductPrices []ASProductPricesSuppliers
	err = json.Unmarshal(body, &ProductPrices)
	if err != nil {
		log.Printf("failed to unmarshal response data: %v", err)
		return
	}

	payload = `{
		"Action": "Edit",
		"Properties": {
			"Locale": "es-US",
			"Timezone": "Argentina Standard Time"
		},
		"Rows": [`

	if len(ProductPrices) == 0 {
		log.Printf("error: no products for that supplier")
		return
	}

	for _, item := range ProductPrices {

		if item.Supplier == supplier.Supplier {

			if item.USDIva != "" {

				supplier_usd, err := strconv.ParseFloat(supplier.SupplierUSD, 64)
				if err != nil {
					log.Printf("%s error parsing supplier usd string to int: %v", item.ProductID, err)
					return
				}

				usd_price, err := strconv.ParseFloat(item.USDPrice, 64)
				if err != nil {
					log.Printf("%s error parsing price usd string to int: %v", item.ProductID, err)
					return
				}

				usd_iva, err := strconv.ParseFloat(item.USDIva, 64)
				if err != nil {
					log.Printf("%s error parsing iva string to int: %v", item.ProductID, err)
					return
				}

				ars_price := fmt.Sprintf("%.2f", (supplier_usd*(1+usd_iva/100))*usd_price)

				payload = payload + fmt.Sprintf(`{
												"product_id" : %s,
												"sale_price_ars" : %s,
												"supplier_usd" : %s
												},`, item.ProductID, ars_price, supplier.SupplierUSD)

			} else if item.USDIva == "" {

				payload = payload + fmt.Sprintf(`{
				"product_id" : %s,
				"supplier_usd" : %s
				},`, item.ProductID, supplier.SupplierUSD)

			}
		}
	}

	payload = payload[:len(payload)-1]

	payload = payload + "]}"

	fmt.Print(payload)

	//fmt.Print(payload)

	// Create the request
	requestURL = fmt.Sprintf("https://api.appsheet.com/api/v2/apps/%s/tables/stock/Action", os.Getenv("appsheet_id"))
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
		log.Printf("unexpected status code from appsheet when updating ars prices: %d", resp.StatusCode)
		log.Println(payload)

	}

}

/*// Connect to the MySQL database
db, err := sql.Open("mysql", "megared_pedro:Engsu_23@tcp(Mysql4.gohsphere.com)/megared_energiaglobal_23?charset=utf8")
if err != nil {
	log.Fatal("Error connecting to the database:", err)
}
defer db.Close()

// Execute the query
rows, err := db.Query(
	fmt.Sprintf(`
	UPDATE STOCK
	SET sale_price_ars = sale_price_usd * %s
	WHERE supplier = '%s';
	`, supplier.SupplierUSD, supplier.Supplier))

if err != nil {
	log.Fatal("Error executing query:", err)
} else {
	rows.Close()
}*/
