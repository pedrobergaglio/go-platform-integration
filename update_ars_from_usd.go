package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	_ "github.com/go-sql-driver/mysql"
)

// handleASMovementWebhook receives a product_id which stock has been modified
// Then the function obtains the product stock in Oran, calculates the configured product stock margin
// and then updates that stock value in the online sales platforms for that specific product
type ASSupplierUSDWebhookPayload struct {
	Supplier    string `json:"supplier"`
	SupplierUSD string `json:"supplier_usd"`
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

	// Connect to the MySQL database
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
	}

}
