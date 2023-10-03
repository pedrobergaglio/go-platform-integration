package main

import (
	"encoding/json"
	"log"
	"net/http"
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
	}*/

}
