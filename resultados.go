package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
)

type Event struct {
	Event string `json:"event"`
	Fecha string `json:"date"`
}

/*type Data struct {
	RowNumber              string `json:"_RowNumber"`
	Fecha                  string `json:"FECHA"`
	DolarOficial           string `json:"DOLAR OFICIAL"`
	DolarBlue              string `json:"DOLAR BLUE"`
	EGCompras              string `json:"EG COMPRAS"`
	EGVentas               string `json:"EG VENTAS"`
	EGStock                string `json:"EG STOCK"`
	EGBancoFuturo          string `json:"EG BANCO FUTURO"`
	EGPedidosPendientes    string `json:"EG PEDIDOS PENDIENTES"`
	EGCaja                 string `json:"EG CAJA"`
	EGDeudasClientes       string `json:"EG DEUDAS CLIENTES"`
	EGDeudasProveedores    string `json:"EG DEUDAS PROVEEDORES"`
	EGGastosInfraestructura string `json:"EG GASTOS INFRAESTRUCTURA"`
	EGDeudasFiscales       string `json:"EG DEUDAS FISCALES"`
	EGCajaUSD              string `json:"EG CAJA USD"`
	EGSueldos              string `json:"EG SUELDOS"`
	EGImpuestos            string `json:"EG IMPUESTOS"`
	EGServicios            string `json:"EG SERVICIOS"`
	EGGastos               string `json:"EG GASTOS"`
	EGActivos              string `json:"EG ACTIVOS"`
	EGPasivos              string `json:"EG PASIVOS"`
	EGTotal                string `json:"EG TOTAL"`
	ITECCompras            string `json:"ITEC COMPRAS"`
	ITECVentas             string `json:"ITEC VENTAS"`
	ITECStock              string `json:"ITEC STOCK"`
	ITECBancoFuturo        string `json:"ITEC BANCO FUTURO"`
	ITECPedidosPendientes  string `json:"ITEC PEDIDOS PENDIENTES"`
	ITECCaja               string `json:"ITEC CAJA"`
	ITECDeudasClientes     string `json:"ITEC DEUDAS CLIENTES"`
	ITECDeudasProveedores  string `json:"ITEC DEUDAS PROVEEDORES"`
	ITECGastosInfraestructura string `json:"ITEC GASTOS INFRAESTRUCTURA"`
	ITECDeudasFiscales     string `json:"ITEC DEUDAS FISCALES"`
	ITECCajaUSD            string `json:"ITEC CAJA USD"`
	ITECSueldos            string `json:"ITEC SUELDOS"`
	ITECImpuestos          string `json:"ITEC IMPUESTOS"`
	ITECServicios          string `json:"ITEC SERVICIOS"`
	ITECGastos             string `json:"ITEC GASTOS"`
	ITECActivos            string `json:"ITEC ACTIVOS"`
	ITECPasivos            string `json:"ITEC PASIVOS"`
	ITECTotal              string `json:"ITEC TOTAL"`
	RelatedRESUMENs        string `json:"Related RESUMENs"`
	ITECCuentas            string `json:"ITEC CUENTAS"`
	EGCuentas              string `json:"EG CUENTAS"`
	FechaString            string `json:"fecha string"`
}

data := []Data{
	{
		RowNumber:              "1",
		Fecha:                  "11/30/2023",
		DolarOficial:           "378.37",
		DolarBlue:              "990",
		EGCompras:              "117546",
		EGVentas:               "195951",
		EGStock:                "1028750",
		EGBancoFuturo:          "29971",
		EGPedidosPendientes:    "0",
		EGCaja:                 "4723.81",
		EGDeudasClientes:       "56867.3",
		EGDeudasProveedores:    "88957.3",
		EGGastosInfraestructura: "0",
		EGDeudasFiscales:       "7127.73",
		EGCajaUSD:              "132333",
		EGSueldos:              "",
		EGImpuestos:            "",
		EGServicios:            "",
		EGGastos:               "",
		EGActivos:              "1120310",
		EGPasivos:              "96085",
		EGTotal:                "1024230",
		ITECCompras:            "51219.2",
		ITECVentas:             "8383.59",
		ITECStock:              "1733160",
		ITECBancoFuturo:        "-13618.3",
		ITECPedidosPendientes:  "0",
		ITECCaja:               "4723.81",
		ITECDeudasClientes:     "56867.3",
		ITECDeudasProveedores:  "0",
		ITECGastosInfraestructura: "0",
		ITECDeudasFiscales:     "58755.4",
		ITECCajaUSD:            "33090",
		ITECSueldos:            "0",
		ITECImpuestos:          "0",
		ITECServicios:          "0",
		ITECGastos:             "0",
		ITECActivos:            "1781130",
		ITECPasivos:            "58755.4",
		ITECTotal:              "1722380",
		RelatedRESUMENs:        "",
		ITECCuentas:            "81062.81",
		EGCuentas:              "134937.81",
		FechaString:            "11/30/2023",
	},
}*/

// Updates meli a publication value (stock or price)
func updateHistorialResultados2(w http.ResponseWriter, r *http.Request) {

	appsheet_id := "acf512aa-6952-4aaf-8d17-c200fefa116b"
	appsheet_key := "V2-RIUo6-uKEV7-puGvy-TeVYT-K2ag9-85j8j-6IaP2-ZX7Rr"

	if r.Method != http.MethodPost {
		log.Println("Invalid request method")
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	// Parse the incoming request body
	var event Event
	if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
		log.Println("error decoding payload:", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	log.Println("evento reconocido en el historial de resultados:", event.Event)

	payload := fmt.Sprintf(`{
		"Action": "Find",
		"Properties": {
		  "Locale": "es-US",
		  "Timezone": "Argentina Standard Time",
		  "Selector": 'Filter("HISTORIAL MES A MES", [FECHA]="%s")'
		},
		"Rows": []
	  }`, event.Fecha)

	requestURL := fmt.Sprintf("https://api.appsheet.com/api/v2/apps/%s/tables/HISTORIAL MES A MES/Action", appsheet_id)
	req, err := http.NewRequest(http.MethodPost, requestURL, bytes.NewBufferString(payload))
	if err != nil {
		log.Printf("failed to create request for appsheet: %v", err)
		return
	}

	// Set request headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("ApplicationAccessKey", appsheet_key)

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

	// Read the response body into jsonData
	jsonData, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("failed to read response body: %v", err)
		return
	}

	// Define a struct to represent the JSON structure
	var data []map[string]interface{}

	// Unmarshal the JSON data into the struct
	err = json.Unmarshal(jsonData, &data)
	if err != nil {
		log.Fatalf("Error decoding JSON: %v", err)
	}

	// Initialize a slice to store the transformed data
	var transformedData []map[string]interface{}

	// Extract fecha from the first row (assuming fecha is the same for all rows)
	fecha := data[0]["FECHA"].(string)
	eg_total := "0"
	itec_total := "0"

	// Iterate through each variable and create a new map for each row
	for key, value := range data[0] {
		if key != "FECHA" && key != "_RowNumber" && key != "Related RESUMENs" && key != "fecha string" && key != "DOLAR OFICIAL" && key != "DOLAR BLUE" {
			if key == "EG TOTAL" {
				eg_total = data[0]["EG TOTAL"].(string)
			}
			if key == "ITEC TOTAL" {
				itec_total = data[0]["ITEC TOTAL"].(string)
			}

			row := map[string]interface{}{
				"fecha":    fecha,
				"variable": key,
				"valor":    value,
			}
			transformedData = append(transformedData, row)
		}
	}

	// Initialize a slice to store the rows
	var rows []byte
	total := "0"

	// Iterate through each transformed row and create a JSON object
	for _, row := range transformedData {

		//if variable starts with EG
		if row["variable"].(string)[:2] == "EG" {
			total = eg_total
		}

		//if variable starts with ITEC
		if row["variable"].(string)[:4] == "ITEC" {
			total = itec_total
		}

		// Create a map for the row
		rowMap := map[string]interface{}{
			"fecha":             row["fecha"],
			"variable":          row["variable"],
			"valor usd oficial": row["valor"],
			"total":             total,
		}

		// Marshal the map into JSON
		rowJSON, err := json.Marshal(rowMap)
		if err != nil {
			log.Printf("failed to marshal row JSON: %v", err)
			continue
		}

		// Append the JSON row to the rows slice
		rows = append(rows, rowJSON...)
		rows = append(rows, ',') // Add a comma to separate rows
	}

	// Remove the trailing comma
	if len(rows) > 0 {
		rows = rows[:len(rows)-1]
	}

	// Add square brackets to make it a valid JSON array
	rows = append([]byte{'['}, rows...)
	rows = append(rows, ']')

	payload = fmt.Sprintf(`{
		"Action": "%s",
		"Properties": {
		"Locale": "es-US",
		"Timezone": "Argentina Standard Time"
		},
		"Rows": %s
		 }`, event.Event, string(rows))

	//LOG PAYLOAD
	//log.Println(payload)

	requestURL = fmt.Sprintf("https://api.appsheet.com/api/v2/apps/%s/tables/HISTORIAL RESULTADOS 2/Action", appsheet_id)

	req, err = http.NewRequest(http.MethodPost, requestURL, bytes.NewBufferString(payload))
	if err != nil {
		log.Printf("failed to create request for appsheet: %v", err)
		return
	}

	// Set request headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("ApplicationAccessKey", appsheet_key)

	// Send the request
	client = http.DefaultClient
	resp, err = client.Do(req)
	if err != nil {
		log.Printf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	// Check the response status code
	if resp.StatusCode != http.StatusOK {
		// Read and log the error response body
		errorBody, _ := io.ReadAll(resp.Body)
		log.Printf("handle publication request unexpected status code from appsheet: %s", errorBody)
		return
	}

	//LOG A SUCCESS MESSAGE
	log.Println("Historial de resultados actualizado exitosamente")
}
