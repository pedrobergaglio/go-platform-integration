package main

import (
	"log"
	"net/http"
)

func main() {
	// Define the handler function for incoming requests
	handler := func(w http.ResponseWriter, r *http.Request) {
		// Write the response
		w.Write([]byte("Hello, World!"))
	}

	// Register the handler function with the default server mux
	http.HandleFunc("/", handler)

	// Start the server and specify the port to listen on
	log.Println("Server listening on port 8080...")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
