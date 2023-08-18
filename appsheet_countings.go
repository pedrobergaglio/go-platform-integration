package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	_ "github.com/go-sql-driver/mysql"
)

// handleASCountingWebhook process the notification of a new counting executed.
// It has too modes
// RECONTEO gets the items to count of the user that saved the counting.
// Then compares each item to the value in the location of the counting
// Adds movements to set the stock value equal to the quantity counted by the user
// LIMPIEZA gets products stock from sql for the selected brands.
// Then for each product, if it is in the counted list,
// sets that value to the stock through a movement. Else, sets the stock to zero.
type ASCountingsWebhookPayload struct {
	ID       string `json:"id"`
	Datetime string `json:"datetime"`
	User     string `json:"user"`
	Location string `json:"location"`
	Mode     string `json:"counting_mode"`
	Brands   string `json:"counting_brands"`
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

	if counting.Mode == "RECONTEO" {
		RECONTEOCounting(counting)
	} else if counting.Mode == "LIMPIEZA (NUEVO)" {
		LIMPIEZACounting(counting)
	} else {
		log.Println("error: no counting mode detected")
		w.WriteHeader(http.StatusBadRequest)
		return
	}
}

func RECONTEOCounting(counting ASCountingsWebhookPayload) {

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

		errr := setProductStock(item.ID, counting.Location, item.Quantity, movement_type)
		if errr != "" {
			log.Printf("error, failed to set product stock: %v", err)
			return
		}

	}

}

type StockData struct {
	ProductID  string
	Brand      string
	Fabrica    string
	Oran       string
	Rodriguez  string
	MarcosPaz  string
	TotalStock string
}

func LIMPIEZACounting(counting ASCountingsWebhookPayload) {

	// Initialize the ASCountingsWebhookPayload struct
	/*counting := ASCountingsWebhookPayload{
		Brands:   "CZERWENY",
		ID:       "999",
		Datetime: "",
		User:     "pedrobergaglio04@gmail.com",
		Location: "Marcos Paz",
		Mode:     "LIMPIEZA (NUEVO)",
	}*/

	// Connect to the MySQL database
	db, err := sql.Open("mysql", "megared_pedro:Engsu_23@tcp(Mysql4.gohsphere.com)/megared_energiaglobal_23?charset=utf8")
	if err != nil {
		log.Fatal("error connecting to the database:", err)
	}
	defer db.Close()

	// *********************
	// Query the products stock with the brands selected
	// *********************

	query := `SELECT product_id, brand, fabrica, oran, rodriguez, marcos_paz, total_stock 
	FROM STOCK
	WHERE `

	brands := strings.Split(counting.Brands, ",")

	for i, brand := range brands {

		if i != 0 {
			query = query + " OR "
		}

		query = query + "brand = '" + strings.TrimSpace(brand) + "'"

	}

	log.Print(query)

	// Move the cursor to the beginning of the result set
	rows, err := db.Query(query)
	if err != nil {
		log.Fatal("error executing query:", err)
	}
	defer rows.Close()

	// Count the number of rows
	var rowCount int

	// Get column names
	columns, err := rows.Columns()
	if err != nil {
		log.Fatal("error getting column names:", err)
	}

	// Create a slice to hold the row values
	values := make([]interface{}, len(columns))
	for i := range values {
		var value interface{}
		values[i] = &value
	}

	var stockDataList []StockData

	// Loop through the rows to process data
	for rows.Next() {

		rowCount++

		// Create a new StockData instance
		var stockData StockData

		// Get the row values and columns
		values := make([]interface{}, len(columns))
		for i := range columns {
			values[i] = new(interface{})
		}
		err := rows.Scan(values...)
		if err != nil {
			log.Fatal("error scanning row:", err)
		}

		// Loop through the columns and assign values to the struct
		for i, col := range columns {
			val := *(values[i].(*interface{}))

			// Convert the byte slice to its appropriate type
			var convertedVal interface{}
			switch v := val.(type) {
			case []byte:
				convertedVal = string(v)
			default:
				convertedVal = v
			}

			// Assign the value to the corresponding field in the struct
			switch col {
			case "product_id":
				stockData.ProductID = convertedVal.(string)
			case "brand":
				stockData.Brand = convertedVal.(string)
			case "fabrica":
				stockData.Fabrica = convertedVal.(string)
			case "oran":
				stockData.Oran = convertedVal.(string)
			case "rodriguez":
				stockData.Rodriguez = convertedVal.(string)
			case "marcos_paz":
				stockData.MarcosPaz = convertedVal.(string)
			case "total_stock":
				stockData.TotalStock = convertedVal.(string)
			}
		}

		// Append the struct to the slice
		stockDataList = append(stockDataList, stockData)
	}

	if err := rows.Err(); err != nil {
		log.Fatal("error retrieving rows:", err)
	}

	// *********************
	// Query the counted products
	// *********************

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
		log.Print(payload)
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

	var flag bool

	// *********************
	// Set the stock as we want
	// *********************

	// Print the stored StockData values
	for _, stockData := range stockDataList {

		flag = true

		//find product if counted
		for _, item := range responseData {

			if stockData.ProductID == item.ID {

				log.Print(stockData.ProductID)

				setProductStock(stockData.ProductID, counting.Location, item.Quantity, movement_type)

				flag = false

			}

		}

		//if this product wasn't counted, then there's no stock
		if flag {
			setProductStock(stockData.ProductID, counting.Location, "0", movement_type)
		}

	}

}

// Pass location param formatted as the user interface.
func setProductStock(product_id, location, str_quantity, movement_type string) (err string) {

	_, _, str_stock, errr := getProductStock(product_id, location)
	if errr != nil {
		return fmt.Sprintf("%s error getting product stock: %v", product_id, errr)
	}

	quantity, errr := strconv.Atoi(str_quantity)
	if errr != nil {
		return fmt.Sprintf("%s error parsing string to int: %v", product_id, errr)
	}

	stock, errr := strconv.Atoi(str_stock)
	if errr != nil {
		return fmt.Sprintf("%s error parsing string to int: %v", product_id, errr)
	}

	if stock != quantity {

		if quantity == 0 {
			log.Printf("setting %s to zero stock in %s", product_id, location)
		}

		stock_difference := convertToString(quantity - stock)

		if location == "Fábrica" {
			_, errr = addMovement(product_id, stock_difference, "0", "0", "0", movement_type)
			if errr != nil {
				return fmt.Sprintf("%s error adding movement to appsheet: %v", product_id, errr)
			}
		} else if location == "Orán" {
			_, errr = addMovement(product_id, "0", stock_difference, "0", "0", movement_type)
			if errr != nil {
				return fmt.Sprintf("%s error adding movement to appsheet: %v", product_id, errr)
			}
		} else if location == "Rodriguez" {
			_, errr = addMovement(product_id, "0", "0", stock_difference, "0", movement_type)
			if errr != nil {
				return fmt.Sprintf("%s error adding movement to appsheet: %v", product_id, errr)
			}
		} else if location == "Marcos Paz" {
			_, errr = addMovement(product_id, "0", "0", "0", stock_difference, movement_type)
			if errr != nil {
				return fmt.Sprintf("%s error adding movement to appsheet: %v", product_id, errr)
			}
		} else {
			log.Println(movement_type, "no location designed")
		}

	}

	return ""

}
