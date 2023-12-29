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
	"fmt"
	"log"
	"net/http"
	"os"
)

func main() {

	loadConfig()

	//go updateSosAt18()
	go refreshPedidosProduccion()
	//go refreshPeriodically()
	//go refreshResumenBanco()

	// Register the webhook handler functions with the default server mux
	//http.HandleFunc("/movement", handleASMovementWebhook)
	//http.HandleFunc("/price", handleASPriceWebhook)
	//http.HandleFunc("/meli", handleMeliWebhook)
	//http.HandleFunc("/woocommerce", handleWCWebhook)
	http.HandleFunc("/countings", handleASCountingWebhook)
	//http.HandleFunc("/usd", handleASUsdWebhook)
	http.HandleFunc("/publication", handlePublicationUpdate)
	http.HandleFunc("/product", handlePublicationRequest)
	http.HandleFunc("/sos", getSosId)
	http.HandleFunc("/cuentas-sos", updateCuentasSosWeb)

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
