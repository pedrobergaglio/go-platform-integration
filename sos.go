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
	"strings"
)

type SOSResponse struct {
	Items []SOSData `json:"items"`
}

type SQLData struct {
	AppsheetID string
	SOSCode    string
	Cuit       string
}

type SOSData struct {
	Codigo string  `json:"codigo"`
	ID     int     `json:"id"`
	IVA    float64 `json:"tasaiva"`
}

// Obtiene las publicaciones SOS que no tienen ID por cuenta, obtiene la base de datos del SOS en ambas cuentas, busca cada uno allí y actualiza el id en Appsheet
func getSosId(w http.ResponseWriter, r *http.Request) {

	//**********************************************************************************************************************************
	//ITEC
	//**********************************************************************************************************************************
	//OBTIENE LAS PUBLICACIONES DE ITEC QUE NO ESTÁN CONECTADAS A SOS
	//**********************************************************************************************************************************

	fmt.Println("")
	fmt.Println("Obteniendo productos del SOS no sincronizados")

	// Fetch data from MySQL
	db, err := sql.Open("mysql", fmt.Sprintf("%s:%s@tcp(%s)/%s?charset=utf8", os.Getenv("database_user"), os.Getenv("database_pass"), os.Getenv("database_ip"), os.Getenv("database_name")))
	if err != nil {
		log.Fatal("error connecting to the database:", err)
	}
	defer db.Close()

	rows, err := db.Query(`SELECT id, sos_cuit, sos_code FROM PLATFORMS WHERE platform='SOS' AND platform_id = 0 AND sos_cuit = 'ITEC'`)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	// Get column names
	columns, err := rows.Columns()
	if err != nil {
		log.Fatal("error getting column names:", err)
	}

	// Create a slice to hold the row values
	values := make([]string, len(columns))
	for i := range values {
		var value string
		values[i] = value
	}

	var sqlDataList []SQLData
	var rowCount int

	// Loop through the rows to process data
	for rows.Next() {

		rowCount++

		// Create a new StockData instance
		var sqlData SQLData

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
			case "id":
				sqlData.AppsheetID = convertedVal.(string)
			case "sos_code":
				sqlData.SOSCode = convertedVal.(string)

			}

			// Append the struct to the slice
			sqlDataList = append(sqlDataList, sqlData)

			if err := rows.Err(); err != nil {
				log.Fatal("error retrieving rows:", err)
			}

			// Print the values obtained from MySQL
			/*for _, sqlData := range sqlDataList {
				fmt.Printf("AppsheetID: %s, SOSCode: %s\n", sqlData.AppsheetID, sqlData.SOSCode)
			}*/

		}
	}

	fmt.Printf("Filas de ITEC obtenidas en MySQL: %d\n", rowCount)

	flag := false
	foundscount := 0

	var sosData SOSResponse

	if rowCount != 0 {

		//**********************************************************************************************************************************
		//obtiene los productos de itec
		//actualiza los que encuentra
		//para los que no encuentra envía error al sistema ya que eran códigos de sos de itec exclusivamente
		//**********************************************************************************************************************************

		// Fetch data from the billing platform
		if true {
			url := "https://soft.sos-contador.com/apiv2/producto/listado?pagina=1&registros=200000"
			token := os.Getenv("sos_itec")

			req, err := http.NewRequest("GET", url, nil)
			if err != nil {
				log.Fatal(err)
			}

			req.Header.Add("Authorization", token)

			client := &http.Client{}
			resp, err := client.Do(req)
			if err != nil {
				log.Fatal(err)
			}
			defer resp.Body.Close()

			if err := json.NewDecoder(resp.Body).Decode(&sosData); err != nil {
				log.Fatal(err)
			}
		}

		fmt.Printf("Productos de la cuenta de ITEC: %d\n", len(sosData.Items))

		rowCount = 0
		flag = false
		foundscount = 0

		// Match and update data
	outerLoop2:
		for _, sqlData := range sqlDataList {
			rowCount++
			flag = false

			for _, sosData := range sosData.Items {

				//fmt.Printf("sos code: %s, sos id: %d\n", sosData.Codigo, sosData.ID)

				if sqlData.SOSCode == "" {
					flag = true
					//fmt.Println("vacio...", sqlData.AppsheetID)
				} else if strings.TrimSpace(sqlData.SOSCode) == strings.TrimSpace(sosData.Codigo) {
					flag = true
					foundscount++
					// Update the AppsheetID with the SOS ID
					fmt.Printf("Matched SOSCode: %s with Codigo: %s. Updating AppsheetID: %s with ID: %d\n", sqlData.SOSCode, sosData.Codigo, sqlData.AppsheetID, sosData.ID)
					// Implement your update logic here

					payload := fmt.Sprintf(`
				{
					"Action": "Edit",
					"Properties": {
						"Locale": "es-US",
						"Timezone": "Argentina Standard Time",
					},
					"Rows": [
						{
							"id": %s,
							"platform_id": %d,
							"product_iva":%f,
							"sos_cuit": "ITEC",
							"sos_id_ok":"Y"
							
						}
					]
				}`, sqlData.AppsheetID, sosData.ID, sosData.IVA)
					// Create the request
					requestURL := fmt.Sprintf("https://api.appsheet.com/api/v2/apps/%s/tables/platforms/Action", os.Getenv("appsheet_id"))
					req, err := http.NewRequest(http.MethodPost, requestURL, bytes.NewBufferString(payload))
					if err != nil {
						log.Fatal(fmt.Printf("failed to create request: %v", err))
					}

					// Set request headers
					req.Header.Set("Content-Type", "application/json")
					req.Header.Set("ApplicationAccessKey", os.Getenv("appsheet_key"))

					// Send the request
					client := http.DefaultClient
					resp, err := client.Do(req)
					if err != nil {
						log.Fatal(fmt.Printf("failed to send request: %v", err))
					}
					defer resp.Body.Close()

					// Check the response status code
					if resp.StatusCode != http.StatusOK {
						fmt.Println(payload)
						log.Fatal(fmt.Printf("unexpected status code: %d", resp.StatusCode))
					}
				}
			}

			if !flag {
				fmt.Printf("Producto de ITEC: %s, no encontrado", sqlData.SOSCode)
				fmt.Println("")

				//ENVIAR ERROR A APPSHEET
				payload := fmt.Sprintf(`
			{
				"Action": "Edit",
				"Properties": {
					"Locale": "es-US",
					"Timezone": "Argentina Standard Time",
				},
				"Rows": [
					{
						"id": %s,
						"sos_id_ok": "N"
						
					}
				]
			}`, sqlData.AppsheetID)
				// Create the request
				requestURL := fmt.Sprintf("https://api.appsheet.com/api/v2/apps/%s/tables/platforms/Action", os.Getenv("appsheet_id"))
				req, err := http.NewRequest(http.MethodPost, requestURL, bytes.NewBufferString(payload))
				if err != nil {
					log.Fatal(fmt.Printf("failed to create request: %v", err))
				}

				// Set request headers
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("ApplicationAccessKey", os.Getenv("appsheet_key"))

				// Send the request
				client := http.DefaultClient
				resp, err := client.Do(req)
				if err != nil {
					log.Fatal(fmt.Printf("failed to send request: %v", err))
				}
				defer resp.Body.Close()

				// Check the response status code
				if resp.StatusCode != http.StatusOK {
					fmt.Println(payload)
					log.Fatal(fmt.Printf("unexpected status code: %d", resp.StatusCode))
				}

				continue outerLoop2
			}

		}

		fmt.Printf("Productos buscados: %d, productos encontrados: %d", rowCount, foundscount)
	}

	//**********************************************************************************************************************************
	//ENERGÍA GLOBAL
	//**********************************************************************************************************************************
	//OBTIENE LAS PUBLICACIONES DE ITEC QUE NO ESTÁN CONECTADAS A SOS
	//**********************************************************************************************************************************

	// Fetch data from MySQL
	db, err = sql.Open("mysql", fmt.Sprintf("%s:%s@tcp(%s)/%s?charset=utf8", os.Getenv("database_user"), os.Getenv("database_pass"), os.Getenv("database_ip"), os.Getenv("database_name")))
	if err != nil {
		log.Fatal("error connecting to the database:", err)
	}
	defer db.Close()

	rows, err = db.Query(`SELECT id, sos_cuit, sos_code FROM PLATFORMS WHERE platform='SOS' AND platform_id = 0 AND sos_cuit = 'ENERGÍA GLOBAL'`)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	// Get column names
	columns, err = rows.Columns()
	if err != nil {
		log.Fatal("error getting column names:", err)
	}

	// Create a slice to hold the row values
	values = make([]string, len(columns))
	for i := range values {
		var value string
		values[i] = value
	}

	rowCount = 0

	// Loop through the rows to process data
	for rows.Next() {

		rowCount++

		// Create a new StockData instance
		var sqlData SQLData

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
			case "id":
				sqlData.AppsheetID = convertedVal.(string)
			case "sos_code":
				sqlData.SOSCode = convertedVal.(string)

			}

			// Append the struct to the slice
			sqlDataList = append(sqlDataList, sqlData)

			if err := rows.Err(); err != nil {
				log.Fatal("error retrieving rows:", err)
			}

			// Print the values obtained from MySQL
			/*for _, sqlData := range sqlDataList {
				fmt.Printf("AppsheetID: %s, SOSCode: %s\n", sqlData.AppsheetID, sqlData.SOSCode)
			}*/

		}
	}

	fmt.Printf("\nFilas de ENERGÍA obtenidas en MySQL: %d\n", rowCount)

	if rowCount == 0 {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Success"))
		return
	}

	//**********************************************************************************************************************************
	//obtiene los productos de ENERGÍA
	//actualiza los que encuentra
	//para los que no encuentra envía error al sistema ya que eran códigos de sos de itec exclusivamente
	//**********************************************************************************************************************************

	fmt.Println("")

	// Fetch data from the billing platform
	if true {
		url := "https://soft.sos-contador.com/apiv2/producto/listado?pagina=1&registros=200000"
		token := os.Getenv("sos_energia")

		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			log.Fatal(err)
		}

		req.Header.Add("Authorization", token)

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			log.Fatal(err)
		}
		defer resp.Body.Close()

		if err := json.NewDecoder(resp.Body).Decode(&sosData); err != nil {
			log.Fatal(err)
		}
	}

	flag = false
	foundscount = 0
	rowCount = 0

	fmt.Printf("Productos de la cuenta de ENERGÍA: %d", len(sosData.Items))

	// Match and update data
	for _, sqlData := range sqlDataList {
		rowCount++
		flag = false

		for _, sosData := range sosData.Items {

			//fmt.Printf("sos code: %s, sos id: %d\n", sosData.Codigo, sosData.ID)

			if sqlData.SOSCode == "" {
				flag = true
				//fmt.Println("vacio...", sqlData.AppsheetID)
			} else if strings.TrimSpace(sqlData.SOSCode) == strings.TrimSpace(sosData.Codigo) {
				flag = true
				foundscount++
				// Update the AppsheetID with the SOS ID
				fmt.Printf("\nENERGÍA SOS Code: %s with Codigo: %s. Updating AppsheetID: %s with ID: %d\n", sqlData.SOSCode, sosData.Codigo, sqlData.AppsheetID, sosData.ID)
				// Implement your update logic here

				payload := fmt.Sprintf(`
				{
					"Action": "Edit",
					"Properties": {
						"Locale": "es-US",
						"Timezone": "Argentina Standard Time",
					},
					"Rows": [
						{
							"id": %s,
							"platform_id": %d,
							"product_iva":%f,
							"sos_cuit": "ENERGÍA GLOBAL",
							"sos_id_ok":"Y"
							
						}
					]
				}`, sqlData.AppsheetID, sosData.ID, sosData.IVA)

				// Create the request
				requestURL := fmt.Sprintf("https://api.appsheet.com/api/v2/apps/%s/tables/platforms/Action", os.Getenv("appsheet_id"))
				req, err := http.NewRequest(http.MethodPost, requestURL, bytes.NewBufferString(payload))
				if err != nil {
					log.Fatal(fmt.Printf("failed to create request: %v", err))
				}

				// Set request headers
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("ApplicationAccessKey", os.Getenv("appsheet_key"))

				// Send the request
				client := http.DefaultClient
				resp, err := client.Do(req)
				if err != nil {
					log.Fatal(fmt.Printf("failed to send request: %v", err))
				}
				defer resp.Body.Close()

				// Check the response status code
				if resp.StatusCode != http.StatusOK {
					fmt.Println(payload)
					log.Fatal(fmt.Printf("unexpected status code: %d", resp.StatusCode))
				}
			}
		}

		if !flag {
			fmt.Printf("\nProducto de ENERGÍA: %s, no encontrado", sqlData.SOSCode)
			fmt.Println("")

			//ENVIAR ERROR A APPSHEET
			payload := fmt.Sprintf(`
			{
				"Action": "Edit",
				"Properties": {
					"Locale": "es-US",
					"Timezone": "Argentina Standard Time",
				},
				"Rows": [
					{
						"id": %s,
						"sos_id_ok": "N"
						
					}
				]
			}`, sqlData.AppsheetID)
			// Create the request
			requestURL := fmt.Sprintf("https://api.appsheet.com/api/v2/apps/%s/tables/platforms/Action", os.Getenv("appsheet_id"))
			req, err := http.NewRequest(http.MethodPost, requestURL, bytes.NewBufferString(payload))
			if err != nil {
				log.Fatal(fmt.Printf("failed to create request: %v", err))
			}

			// Set request headers
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("ApplicationAccessKey", os.Getenv("appsheet_key"))

			// Send the request
			client := http.DefaultClient
			resp, err := client.Do(req)
			if err != nil {
				log.Fatal(fmt.Printf("failed to send request: %v", err))
			}
			defer resp.Body.Close()

			// Check the response status code
			if resp.StatusCode != http.StatusOK {
				fmt.Println(payload)
				log.Fatal(fmt.Printf("unexpected status code: %d", resp.StatusCode))

			}

		}

	}

	fmt.Printf("Productos buscados: %d, productos encontrados: %d\n\n", rowCount, foundscount)

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Success"))
	return
}

func updateSos(price, iva, id, cuenta, producto, codigo string) string {

	url := fmt.Sprintf(`https://soft.sos-contador.com/apiv2/producto/%s`, id)

	payload := fmt.Sprintf(`
	{
		"codigo": "%s",
		"producto": "%s",
		"idunidad": 7,
		"precio1": %s,
		"product_iva":%s
	}`, codigo, producto, price, iva)

	req, err := http.NewRequest(http.MethodPut, url, bytes.NewBufferString(payload))
	if err != nil {
		return "error creating request for meli:" + fmt.Sprint(err)
	}

	auth := ""

	if cuenta == "ITEC" {
		auth = os.Getenv("sos_itec")
	} else if cuenta == "ENERGÍA GLOBAL" {
		auth = os.Getenv("sos_energia")
	} else {
		return "cuenta de SOS no especificada"
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", auth)

	client := http.DefaultClient
	resp, err := client.Do(req)
	if err != nil {
		return "error updating product in sos:" + fmt.Sprint(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return ""
	} else {
		errorBody, _ := io.ReadAll(resp.Body)
		fmt.Println(payload)
		return "error updating product in sos: " + string(errorBody)
	}

}
