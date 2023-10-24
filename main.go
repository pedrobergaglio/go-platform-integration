package main

//https://articulo.mercadolibre.com.ar/MLA-1449065018-item-de-prueba-por-favor-no-ofertar-_JM

/*
COMPRADOR
{
	"id":1422812884,
	"email":"test_user_941471677@testuser.com",
	"nickname":"TESTUSER941471677",
	"site_status":"active",
	"password":"nT3xqyQeup"
	}
*/

/*
VENDEDOR
{
	"id":1423001750,
	"email":"test_user_275613916@testuser.com",
	"nickname":"TESTUSER275613916",
	"site_status":"active",
	"password":"gzoLcRK5SJ"
}
*/

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

/*
func createUser() {

	// Define the request URL and payload
	url := "https://api.mercadolibre.com/users/test_user"
	payload := `{
		"site_id": "MLA"
	}`

	// Create the HTTP request
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBufferString(payload))
	if err != nil {
		fmt.Println(req)
		return
	}

	fmt.Println(os.Getenv("MELI_ACCESS_TOKEN"))

	// Set the request headers
	req.Header.Set("Authorization", "Bearer "+os.Getenv("MELI_ACCESS_TOKEN"))
	req.Header.Set("Content-type", "application/json")

	// Send the request
	client := http.DefaultClient
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("do")
		return
	}
	defer resp.Body.Close()

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("body")
		return
	}

	fmt.Println(string(body))
}*/

// Sets each enviroment variable declared in the .env file
func loadConfig() {
	viper.SetConfigFile("resources/.env") // Set the path to your configuration file
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

func main() {

	//loadConfig()
	//RunAtTime()

	// Get the path to the directory where the binary is located
	exe, err := os.Executable()
	if err != nil {
		log.Fatal(err)
	}

	// Get the directory of the executable
	dir := filepath.Dir(exe)

	// Specify the path to the .env file relative to the executable's directory
	envFile := filepath.Join(dir, "resources", ".env")

	// Load the .env file
	err = godotenv.Load(envFile)
	if err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}

	// Start a background goroutine to periodically refresh prices and tokens
	go refreshPeriodically()

	// Register the webhook handler functions with the default server mux
	http.HandleFunc("/movement", handleASMovementWebhook)
	http.HandleFunc("/price", handleASPriceWebhook)
	//http.HandleFunc("/meli", handleMeliWebhook)
	//http.HandleFunc("/woocommerce", handleWCWebhook)
	http.HandleFunc("/countings", handleASCountingWebhook)
	//http.HandleFunc("/usd", handleASUsdWebhook)
	http.HandleFunc("/publication", handlePublicationUpdate)
	http.HandleFunc("/product", handlePublicationRequest)
	http.HandleFunc("/sos", getSosId)

	// Root route handler
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Write the desired text response
		fmt.Fprintf(w, "ENERGÍA GLOBAL INTEGRATION SERVICE")
	})

	// Use PORT environment variable provided by Railway or default to 8080
	port := ":" + os.Getenv("port")
	if port == ":" {
		port = ":8080"
	}

	// Start the server and specify the host and port
	log.Println("server listening on", port)
	log.Fatal(http.ListenAndServe("0.0.0.0"+port, nil))

}

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
		fmt.Println("error converting publication id to int:", err)
		return
	}

	// si el precio es muy bajo, desactivamos la publicación con un stock 0
	// salvo en alephee que siempre tiene los precios bien
	if price_float < 1000 {
		fmt.Println("error price under 1000:", publication.Price, "setting stock to 0")

		if publication.Platform == "MELI" {

			error := updateMeli(publication.ID, "", "0")
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

		error := updateMeli(publication.ID, publication.Price, publication.Stock)
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
		fmt.Println(payload)
		log.Printf("handle publication request unexpected status code from appsheet: %s", errorBody)
		return
	}

	w.WriteHeader(http.StatusOK)
}
