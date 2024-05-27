package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"strings"
)

type Publication struct {
	ID        string `json:"id"`
	Platform  string `json:"platform"`
	Price     string `json:"price"`
	Stock     string `json:"stock"`
	SIVAPrice string `json:"siva_price"`
	Cuenta    string `json:"sos_cuit"`
	Producto  string `json:"product"`
	SOSCode   string `json:"sos_code"`
	IVA       string `json:"product_iva"`
}

func handlePublicationUpdate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		log.Println("Invalid request method")
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	// Parse the incoming request body
	var publication Publication
	if err := json.NewDecoder(r.Body).Decode(&publication); err != nil {
		log.Println("error decoding payload:", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	log.Println("actualizando la publicacion", publication.ID,
		"en", publication.Platform,
		"stock y precio:", publication.Stock, publication.Price)

	price_float, err := strconv.ParseFloat(publication.Price, 64)
	if err != nil {
		log.Println("error converting publication id to int:", err)
		return
	}

	// si el precio es muy bajo, desactivamos la publicación con un stock 0
	// salvo en alephee que siempre tiene los precios bien
	if price_float < 1000 {
		log.Println("error price under 1000:", publication.Price, "setting stock to 0")

		if publication.Platform == "MELI" {

			error := updateMeli(publication.ID, "", "0", publication.Cuenta)
			if error != "" {
				log.Println("error updating product in meli:", error)
			}

		} else if publication.Platform == "ALEPHEE" {

			error := updateAlephee(publication.ID, publication.Stock)
			if error != "" {
				log.Println("error updating stock in alephee:", error)
			}

		} else if publication.Platform == "WC" {

			error := updateWC(publication.ID, "", "0")
			if error != "" {
				log.Println("error updating product in wc:", error)
			}

		} else if publication.Platform == "SOS" {
		} else {
			log.Println("no platform matched")
		}

		return
	}

	if publication.Platform == "MELI" {

		error := updateMeli(publication.ID, publication.Price, publication.Stock, publication.Cuenta)
		if error != "" {
			log.Println("error updating product in meli:", error)
		}

	} else if publication.Platform == "ALEPHEE" {

		error := updateAlephee(publication.ID, publication.Stock)
		if error != "" {
			log.Println("error updating stock in alephee:", error)
		}

	} else if publication.Platform == "WC" {

		error := updateWC(publication.ID, publication.Price, publication.Stock)
		if error != "" {
			log.Println("error updating product in wc:", error)
		}

	} else if publication.Platform == "SOS" {

		error := updateSos(publication.SIVAPrice, publication.IVA, publication.ID, publication.Cuenta, publication.Producto, publication.SOSCode)
		if error != "" {
			log.Println("error updating product in sos:", error)
		}

	} else {
		log.Println("no platform matched")
	}

	w.WriteHeader(http.StatusOK)
}

type PublicationRequest struct {
	Publications string `json:"publications"`
	ID           string `json:"id"`
}

// Recibe las publicaciones de un producto y updatea cada una en appsheet
func handlePublicationRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		log.Println("Invalid request method")
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var requestData PublicationRequest
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&requestData); err != nil {
		log.Println("Failed to decode JSON request body:", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	log.Println("Producto actualizado: ", requestData.ID)
	log.Println("Publicaciones:", requestData.Publications)

	// Split the comma-separated codes into an array
	codes := strings.Split(requestData.Publications, ",")

	if codes[0] == "" {
		return
	}

	// Initialize the payload with the common part
	payload := `{
		"Action": "Edit",
		"Properties": {
			"Locale": "es-US",
			"Timezone": "Argentina Standard Time"
		},
		"Rows": [`

	// Generate payload entries for each code
	for _, code := range codes {
		// Assuming you have a function to generate a random integer
		randomNum := rand.Intn(1000000)
		// Add an entry for the code and randomNum
		payload += fmt.Sprintf(`{
			"id" : "%s",
			"updatecol" : "%d"
		},`, strings.TrimSpace(code), randomNum)
	}

	// Complete the payload
	payload = payload[:len(payload)-1]
	payload += `]}`

	// Create the request

	requestURL := fmt.Sprintf("https://api.appsheet.com/api/v2/apps/%s/tables/platforms/Action", os.Getenv("appsheet_id"))
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
		// Read and log the error response body
		errorBody, _ := io.ReadAll(resp.Body)
		log.Println(payload)
		log.Printf("handle publication request unexpected status code from appsheet: %s", errorBody)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// handleASMovementWebhook receives a product_id which stock has been modified
// Then the function obtains the product stock in Oran, calculates the configured product stock margin
// and then updates that stock value in the online sales platforms for that specific product
type ASProductIDWebhookPayload struct {
	ProductID string `json:"product_id"`
}

/*
func handleASMovementWebhook(w http.ResponseWriter, r *http.Request) {
	// Ensure that the request method is POST
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		log.Println("error: method is not post")
		return
	}

	// Parse the incoming request body
	var payload ASProductIDWebhookPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		log.Println("error decoding payload:", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// string to
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

			error = updateMeli(item.PlatformID, "", stock_minus_margin)
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
		log.Printf("getPlatformsID %s error getting platforms product data: %s", payload.ProductID, err)
		return
	}

	sale_pricestr, _, _, errr := getProductStock(payload.ProductID, "")
	if errr != nil {
		log.Printf("getProductStock %s error getting platforms product data: %v", payload.ProductID, errr)
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
				log.Println(item.PlatformID, "error updating meli price:", errr)
				//flag = 1
			}

			// Update the WooCommerce product
		} else if item.Platform == "WC" {
			errr := updateWC(convertToString(item.PlatformID), "regular_price", `"`+sale_pricestr+`"`)
			if errr != "" {
				log.Println(item.PlatformID, "error updating woocommerce price:", errr)
				//flag = 1
			}
		}

	}

	/* Write a success response if everything is processed successfully
	if flag == 0 {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("webhook processed successfully"))
		log.Println("price updated")
	}

}
*/
