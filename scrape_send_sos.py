import time
from bs4 import BeautifulSoup
from selenium import webdriver
from webdriver_manager.chrome import ChromeDriverManager
from selenium.webdriver.common.by import By
from selenium.webdriver.common.keys import Keys
from selenium.webdriver.chrome.service import Service
import random
import json
import requests
import mysql.connector
import os
from dotenv import load_dotenv
from selenium.common.exceptions import NoSuchElementException
from selenium.webdriver.common.by import By
from selenium.webdriver.support.ui import WebDriverWait
from selenium.webdriver.support import expected_conditions as EC
from selenium.common.exceptions import TimeoutException
from selenium.webdriver.support.ui import Select
import datetime

# Get the current date
current_date = datetime.datetime.now()

# Extract the year and month
year = current_date.year
month = current_date.month -1


# Initialize the driver
#driver = webdriver.Chrome(ChromeDriverManager().install())
option = webdriver.ChromeOptions()
option.add_argument("start-maximized")
option.add_argument("--headless=new")
#--headless --disable-gpu --disable-software-rasterizer --disable-extensions --no-sandbox
option.add_argument("--disable-gpu")
option.add_argument("--disable-software-rasterizer")
option.add_argument("--disable-extensions")
option.add_argument("--no-sandbox")

appsheet_id = "acf512aa-6952-4aaf-8d17-c200fefa116b"
appsheet_key = "V2-RIUo6-uKEV7-puGvy-TeVYT-K2ag9-85j8j-6IaP2-ZX7Rr"

driver = webdriver.Chrome(service=Service(ChromeDriverManager().install()),options=option)

# Define your URL, username, and password
username = "pedro.bergaglio@energiaglobal.com.ar"
password = "FP3235"

carga = 5
error_flag = True

# Function to parse the amount
def parse_amount(amount_str):

    # Replace commas with an auxiliary character
    amount_str = amount_str.replace(',', '')
    #abs
    # Replace dots with commas
    amount_str = amount_str.replace('.', ',')
    # Replace the auxiliary character with dots
    amount_str = amount_str.replace('X', '.')

    return amount_str

# Function to parse the amount
def parse_amount_abs(amount_str):

    # Replace commas with an auxiliary character
    amount_str = amount_str.replace(',', '')
    #abs
    amount_str = str(abs(float(amount_str)))
    # Replace dots with commas
    amount_str = amount_str.replace('.', ',')
    # Replace the auxiliary character with dots
    amount_str = amount_str.replace('X', '.')

    return amount_str

def login():
    try:
        driver.get("https://nuevo.soft.sos-contador.com/web/comprobante_altamodi.asp?operacion=2&listado=1")

        username_input = driver.find_element(By.NAME, "usuario")
        password_input = driver.find_element(By.NAME, "clave")
        login_button = driver.find_element(By.CLASS_NAME, "btn-success")

        username_input.send_keys(username)
        password_input.send_keys(password)
        login_button.click()

        time.sleep(carga)
        
    except NoSuchElementException as e:
        print(f"Element not found: {e}")
        error_flag = False
    except TimeoutException as e:
        print(f"Timeout error: {e}")
        error_flag = False
    except Exception as e:
        print(f"An error occurred: {e}")
        error_flag = False

def scrape_clientes(cuenta):
    # Call the login function to authenticate
    driver.get("https://nuevo.soft.sos-contador.com/web/asistente_deuda.asp?CP=C" )
    
    # Wait for the page to load after login 
    driver.implicitly_wait(carga)

    time.sleep(carga)

    # Now, you can locate the table and scrape its data using a loop
    # Locate the table by its ID
    table = driver.find_element(By.ID, "listado")

    # Locate the table body (inside the table)
    table_body = table.find_element(By.TAG_NAME, "tbody")

    # Find all rows in the table body
    rows = table_body.find_elements(By.TAG_NAME, "tr")

    counter = 0
    payload_data = []

    # Iterate through rows
    for row in rows:

        counter += 1
        # Find all columns (td elements) in the row
        columns = row.find_elements(By.TAG_NAME, "td")

        # Check if the row has at least 6 columns (to ensure you can access the 2nd, 3rd, 5th, and 6th columns)
        if len(columns) >= 6:
            # Extract content from the desired columns (0-based index)
            cliente = columns[1].text
            cuit = columns[2].text
            deuda = columns[4].text
            ultimopago = columns[5].text

            # Create a dictionary for the row
            row_data = {
                "number": counter,
                "customer": cliente,
                "cuit": cuit,
                "debt": parse_amount_abs(deuda),
                "last_payment": ultimopago,
                "cuenta": cuenta
            }

            payload_data.append(row_data)
    
    # Build the payload with all row data
    payload = {
        "Action": "Add",
        "Properties": {
            "Locale": "es-AR",
            "Timezone": "Argentina Standard Time"
        },
        "Rows": payload_data
    }

    #print(payload)

    # Define the request URL
    requestURL = f"https://api.appsheet.com/api/v2/apps/{appsheet_id}/tables/customer_accounts/Action"

    # Set request headers
    headers = {
        "Content-Type": "application/json",
        "ApplicationAccessKey": appsheet_key
    }

    # Send the request
    response = requests.post(requestURL, data=json.dumps(payload), headers=headers)

    # Check the response status code
    if response.status_code != 200:
        print(f"Request failed with status code: {response.status_code}")
        print(response.text)
        errorflag = False

def scrape_proveedores(cuenta):
    # Call the login function to authenticate
    driver.get("https://nuevo.soft.sos-contador.com/web/asistente_deuda.asp?CP=P" )
    
    # Wait for the page to load after login 
    driver.implicitly_wait(carga)

    time.sleep(carga)

    # Now, you can locate the table and scrape its data using a loop
    # Locate the table by its ID
    table = driver.find_element(By.ID, "listado")

    # Locate the table body (inside the table)
    table_body = table.find_element(By.TAG_NAME, "tbody")

    # Find all rows in the table body
    rows = table_body.find_elements(By.TAG_NAME, "tr")

    counter = 0
    payload_data = []

    # Iterate through rows
    for row in rows:

        counter += 1
        # Find all columns (td elements) in the row
        columns = row.find_elements(By.TAG_NAME, "td")

        # Check if the row has at least 6 columns (to ensure you can access the 2nd, 3rd, 5th, and 6th columns)
        if len(columns) >= 6:
            # Extract content from the desired columns (0-based index)
            cliente = columns[1].text
            cuit = columns[2].text
            deuda = columns[4].text
            ultimopago = columns[5].text

            # Create a dictionary for the row
            row_data = {
                "number": counter,
                "customer": cliente,
                "cuit": cuit,
                "debt": parse_amount_abs(deuda),
                "last_payment": ultimopago,
                "cuenta": cuenta
            }

            payload_data.append(row_data)
    
    # Build the payload with all row data
    payload = {
        "Action": "Add",
        "Properties": {
            "Locale": "es-AR",
            "Timezone": "Argentina Standard Time"
        },
        "Rows": payload_data
    }

    #print(payload)

    # Define the request URL
    requestURL = f"https://api.appsheet.com/api/v2/apps/{appsheet_id}/tables/supplier_accounts/Action"

    # Set request headers
    headers = {
        "Content-Type": "application/json",
        "ApplicationAccessKey": appsheet_key
    }

    # Send the request
    response = requests.post(requestURL, data=json.dumps(payload), headers=headers)

    # Check the response status code
    if response.status_code != 200:
        print(f"Request failed with status code: {response.status_code}")
        print(response.text)
        errorflag = False

def scrape_ventas(cuit):
    # Call the login function to authenticate
    driver.get("https://nuevo.soft.sos-contador.com/web/comprobante_altamodi.asp?operacion=2&listado=1")
    
    #Wait for the page to load after login (you can adjust the time as needed)
    driver.implicitly_wait(carga)

    #Find the dropdown element by its class name
    length = driver.find_element(By.ID, 'listado_length')

    dropdown = length.find_element(By.CLASS_NAME, 'selectize-input')

    dropdown.click()

    driver.implicitly_wait(carga)

    dropdown = length.find_element(By.CLASS_NAME, 'selectize-dropdown-content')

    dropdown.find_element(By.CSS_SELECTOR, '[data-value="500"]').click()

    time.sleep(carga)

    #selectize-dropdown-content

    # Now, you can locate the table and scrape its data using a loop
    # Locate the table by its ID
    table = driver.find_element(By.ID, "listado")


    # Locate the table body (inside the table)
    table_body = table.find_element(By.TAG_NAME, "tbody")

    # Find all rows in the table body
    rows = table_body.find_elements(By.TAG_NAME, "tr")

    ######################################################################

    counter = 0
    payload_data = []
    
    # Iterate through rows
    for row in rows:
        counter += 1

        # Find all columns (td elements) in the row
        columns = row.find_elements(By.TAG_NAME, "td")

        # Check if the row has at least 6 columns (to ensure you can access the 2nd, 3rd, 5th, and 6th columns)
        if len(columns) >= 6:
            # Extract content from the desired columns (0-based index)
            content_2nd_td = columns[1].get_attribute("innerHTML")
            content_3rd_td = columns[2].get_attribute("innerHTML")
            content_4th_td = columns[3].get_attribute("innerHTML")
            monto = parse_amount(columns[4].text)
            cae = columns[5].text

            # Parse the date and modification
            parts = content_2nd_td.split("<br>")
            date = parts[0]
            second = parts[1].split('>')
            second = second[1].split('<')
            modification = second[0].split(' ')

            # Parse the cliente and comprobante
            parts = content_3rd_td.split("<br>")
            cliente = parts[0]
            second = content_4th_td.split('>')[1]
            comprobante = second.split('<')[0]

            # Create a dictionary for the row
            row_data = {
                "number": counter,
                "date": date,
                "last_edit": f"{modification[0]} {modification[1]}",
                "user": modification[2],
                "customer": cliente,
                "receipt": comprobante,
                "amount": monto,
                "CAE": cae,
                "payment_state": "Pago",
                "cuit":cuit
            }

            if cae == "Error": continue

            payload_data.append(row_data)



    # Build the payload with all row data
    payload = {
        "Action": "Add",
        "Properties": {
            "Locale": "es-AR",
            "Timezone": "Argentina Standard Time"
        },
        "Rows": payload_data
    }

    #print(payload)

    # Define the request URL
    requestURL = f"https://api.appsheet.com/api/v2/apps/{appsheet_id}/tables/MONTH_SALES_EG/Action"

    # Set request headers
    headers = {
        "Content-Type": "application/json",
        "ApplicationAccessKey": appsheet_key
    }

    # Send the request
    response = requests.post(requestURL, data=json.dumps(payload), headers=headers)

    # Check the response status code
    if response.status_code != 200:
        print(f"Request failed with status code: {response.status_code}")
        print(response.text)
        errorflag = False

def scrape_compras(cuit):
    # Call the login function to authenticate
    driver.get("https://nuevo.soft.sos-contador.com/web/comprobante_altamodi.asp?operacion=4&listado=1")
    
    #Wait for the page to load after login (you can adjust the time as needed)
    driver.implicitly_wait(carga)

    #Find the dropdown element by its class name
    length = driver.find_element(By.ID, 'listado_length')

    dropdown = length.find_element(By.CLASS_NAME, 'selectize-input')

    dropdown.click()

    driver.implicitly_wait(carga)

    dropdown = length.find_element(By.CLASS_NAME, 'selectize-dropdown-content')

    dropdown.find_element(By.CSS_SELECTOR, '[data-value="500"]').click()

    time.sleep(carga)

    #selectize-dropdown-content

    # Now, you can locate the table and scrape its data using a loop
    # Locate the table by its ID
    table = driver.find_element(By.ID, "listado")


    # Locate the table body (inside the table)
    table_body = table.find_element(By.TAG_NAME, "tbody")

    # Find all rows in the table body
    rows = table_body.find_elements(By.TAG_NAME, "tr")

    ######################################################################

    counter = 0
    payload_data = []
    
    # Iterate through rows
    for row in rows:
        counter += 1

        # Find all columns (td elements) in the row
        columns = row.find_elements(By.TAG_NAME, "td")

        # Check if the row has at least 6 columns (to ensure you can access the 2nd, 3rd, 5th, and 6th columns)
        if len(columns) >= 6:
            # Extract content from the desired columns (0-based index)
            content_2nd_td = columns[1].get_attribute("innerHTML")
            content_3rd_td = columns[2].get_attribute("innerHTML")
            content_4th_td = columns[3].get_attribute("innerHTML")
            monto = parse_amount(columns[4].text)
            cae = columns[5].text

            # Parse the date and modification
            parts = content_2nd_td.split("<br>")
            date = parts[0]
            second = parts[1].split('>')
            second = second[1].split('<')
            modification = second[0].split(' ')

            # Parse the cliente and comprobante
            parts = content_3rd_td.split("<br>")
            cliente = parts[0]
            second = content_4th_td.split('>')[1]
            comprobante = second.split('<')[0]

            # Create a dictionary for the row
            row_data = {
                "number": counter,
                "date": date,
                "last_edit": f"{modification[0]} {modification[1]}",
                "user": modification[2],
                "customer": cliente,
                "receipt": comprobante,
                "amount": monto,
                "CAE": cae,
                "payment_state": "Pago",
                "cuit":cuit
            }

            payload_data.append(row_data)



    # Build the payload with all row data
    payload = {
        "Action": "Add",
        "Properties": {
            "Locale": "es-AR",
            "Timezone": "Argentina Standard Time"
        },
        "Rows": payload_data
    }

    #print(payload)

    # Define the request URL
    requestURL = f"https://api.appsheet.com/api/v2/apps/{appsheet_id}/tables/MONTH_PURCHASES/Action"

    # Set request headers
    headers = {
        "Content-Type": "application/json",
        "ApplicationAccessKey": appsheet_key
    }

    # Send the request
    response = requests.post(requestURL, data=json.dumps(payload), headers=headers)

    # Check the response status code
    if response.status_code != 200:
        print(f"Request failed with status code: {response.status_code}")
        print(response.text)
        errorflag = False

#energía 1, itec 2
def cuit(cuit):

    dropdown = driver.find_element(By.ID, "dropdown-cuit")
    
    dropdown.click()

    driver.implicitly_wait(carga)

    driver.find_element(By.ID, "menu-cuits").find_element(By.XPATH, f'//*[@id="lista-cuits"]/li[{cuit}]').click()

    time.sleep(carga)

def truncate():
    # Load environment variables from the .env file
    load_dotenv(dotenv_path='resources/.env')

    # Verify environment variables
    print("DATABASE_IP:", os.getenv("DATABASE_IP"))
    print("DATABASE_USER:", os.getenv("DATABASE_USER"))
    print("DATABASE_PASS:", os.getenv("DATABASE_PASS"))
    print("DATABASE_NAME:", os.getenv("DATABASE_NAME"))

    # Define the queries
    queries = [
        "TRUNCATE CUSTOMER_ACCOUNTS;",
        "TRUNCATE SUPPLIER_ACCOUNTS;",
        "TRUNCATE MONTH_SALES_EG;",
        "TRUNCATE MONTH_PURCHASES;"
        #"TRUNCATE RESUMEN;"
    ]

    # Establish a database connection
    connection = None
    
    try:
        connection = mysql.connector.connect(
            host=os.getenv("DATABASE_IP"),
            user=os.getenv("DATABASE_USER"),
            password=os.getenv("DATABASE_PASS"),
            database=os.getenv("DATABASE_NAME")
        )

        if connection.is_connected():
            #print("Connected to the database")

            cursor = connection.cursor()

            for query in queries:
                cursor.execute(query)
                #print(f"Query executed: {query}")

    except mysql.connector.Error as error:
        print(f"Error: {error}")
        error_flag = False

    finally:
        if connection is not None and connection.is_connected():
            cursor.close()
            connection.close()

#energía 1, itec 2
def periodo():

    dropdown = driver.find_element(By.ID, "muestra-periodo")
    
    dropdown.click()

    driver.implicitly_wait(carga)

    lista = driver.find_element(By.ID, "lista-periodos")
    #.find_element(By.XPATH, f'//*[@id="lista-cuits"]/li[{cuit}]').click()

    # Find all rows in the table body
    rows = lista.find_elements(By.TAG_NAME, "li")

    counter = 0
    payload_data = []

    flag = True

    # Iterate through rows
    for row in rows:
        try:
            # Try to find the <a> element
            a_element = row.find_element(By.TAG_NAME, "a")
            
            # Check if data-anio and data-mes match
            if a_element.get_attribute("data-anio") == str(year) and a_element.get_attribute("data-mes") == str(month):
                # Click the <a> element if it matches
                a_element.click()
                flag= False
                break
            
        except NoSuchElementException:
            continue

    if flag: print("periodo actual no encontrado, utilizando el actual")
    time.sleep(carga)

def update():

    payload = {
    "Action": "Actualizar2",
    "Properties": {
        "Locale": "en-US",
        "Location": "47.623098, -122.330184",
        "Timezone": "Pacific Standard Time",
        "UserSettings": {
            "Option 1": "value1",
            "Option 2": "value2"
        }
    },
    "Rows": [
        {
            "ID": "hola"
        }
    ]
    }

    # Define the request URL
    requestURL = f"https://api.appsheet.com/api/v2/apps/{appsheet_id}/tables/RESUMEN/Action"

    # Set request headers
    headers = {
        "Content-Type": "application/json",
        "ApplicationAccessKey": appsheet_key
    }

    # Send the request
    response = requests.post(requestURL, data=json.dumps(payload), headers=headers)

    # Check the response status code
    if response.status_code != 200:
        print(f"Request failed with status code: {response.status_code}")
        print(response.text)
        errorflag = False

def login_afip():

    # Define your URL, username, and password
    afip_username = "20180595733"
    afip_password = "EgMrGi2023"

    try:
        driver.get("https://ctacte.cloud.afip.gob.ar/contribuyente/externo")

        username_input = driver.find_element(By.ID, "F1:username")
        username_input.send_keys(afip_username)
        driver.find_element(By.ID, "F1:btnSiguiente").click()
        driver.implicitly_wait(carga)
        password_input = driver.find_element(By.ID, "F1:password")
        password_input.send_keys(afip_password)
        login_button = driver.find_element(By.ID, "F1:btnIngresar")
        login_button.click()

        driver.implicitly_wait(carga)
        
    except NoSuchElementException as e:
        print(f"Element not found: {e}")
        error_flag = False
    except TimeoutException as e:
        print(f"Timeout error: {e}")
        error_flag = False
    except Exception as e:
        print(f"An error occurred: {e}")
        error_flag = False

def acceso_sistema_de_cuentas():
    # Call the login function to authenticate
    driver.get("https://portalcf.cloud.afip.gob.ar/portal/app/" )
    
    # Wait for the page to load after login 
    driver.implicitly_wait(carga)

    # VER TODOS
    #servicios = driver.find_element(By.CLASS_NAME, "row panels-row m-x-0")
    driver.find_element(By.CSS_SELECTOR,  "#serviciosMasUtilizados > div > div > div > div:nth-child(5) > div > a").click()
    driver.implicitly_wait(carga)

    time.sleep(carga)

    # CONTAINER SERVICIOS
    #container = driver.find_element(By.CSS_SELECTOR, "#root > div > main > div > section > div > div.row.panels-row")

    # CONTAINER SERVICIOS
    container = driver.find_element(By.CSS_SELECTOR, "#root > div > main > div > section > div > div.row.panels-row")

    # Use WebDriverWait to wait for the container to be present
    wait = WebDriverWait(driver, 10)
    container = wait.until(EC.presence_of_element_located((By.CSS_SELECTOR, "#root > div > main > div > section > div > div.row.panels-row")))

    #print(container.get_attribute("outerHTML"))
    servicios = container.find_elements(By.CSS_SELECTOR, "div.col-xs-12.col-sm-6.col-md-4")

    counter = 0

    # Iterate through rows
    for servicio in servicios:

        counter += 1
            # Find the h3 element within each row
        h3_element = servicio.find_element(By.TAG_NAME, "h3")

        # Get the text content of the h3 element
        h3_text = h3_element.text

        # Check if the text is "Aceptación de Datos Biométricos"
        if h3_text == "SISTEMA DE CUENTAS TRIBUTARIAS":
            print("Found SISTEMA DE CUENTAS TRIBUTARIAS")

            notfound = False

            # Scroll into view using JavaScript
            driver.execute_script("arguments[0].scrollIntoView();", servicio)

            # Click the element
            # Click the element using JavaScript
            driver.execute_script("arguments[0].click();", servicio)
            return
    
    if notfound: print("SISTEMA DE CUENTAS TRIBUTARIAS no encontrado"); error_flag = False

def scrape_deudas():

    driver.implicitly_wait(carga)

    # Call the login function to authenticate
    driver.get("https://ctacte.cloud.afip.gob.ar/contribuyente/externo" )
    
    # Wait for the page to load after login 
    driver.implicitly_wait(carga)
     
    # Assuming the dropdown is inside an element with id "cuitForm"
    dropdown_element = driver.find_element(By.ID, "cuitForm")

    # Create a Select object from the dropdown element
    dropdown = Select(dropdown_element.find_element(By.TAG_NAME, "select"))

    # Specify the target value
    target_value = "30716503387"

    # Iterate through all options in the dropdown
    for option in dropdown.options:
        # Check if the option's value matches the target value
        if option.text == target_value:
            # Select the option
            dropdown.select_by_visible_text(target_value)
            break  # Exit the loop once the target option is found

    value_element = WebDriverWait(driver, 10).until(
    EC.presence_of_element_located((By.CSS_SELECTOR, '.text-control input[name="text"]'))
    )

    # Now, retrieve the text value
    value = value_element.get_attribute('value')

    #take out the first two characters, and replace . with nothing, then , with ., then print
    value = value[2:].replace('.', '').replace(',', '.')
    itec_deuda = float(value)

    # Assuming the dropdown is inside an element with id "cuitForm"
    dropdown_element = driver.find_element(By.ID, "cuitForm")

    # Create a Select object from the dropdown element
    dropdown = Select(dropdown_element.find_element(By.TAG_NAME, "select"))

    # Specify the target value
    target_value = "30710746997"

    # Iterate through all options in the dropdown
    for option in dropdown.options:
        # Check if the option's value matches the target value
        if option.text == target_value:
            # Select the option
            dropdown.select_by_visible_text(target_value)
            break  # Exit the loop once the target option is found

    value_element = WebDriverWait(driver, 10).until(
    EC.presence_of_element_located((By.CSS_SELECTOR, '.text-control input[name="text"]'))
    )

    # Now, retrieve the text value
    value = value_element.get_attribute('value')

    #take out the first two characters, and replace . with nothing, then , with ., then print
    value = value[2:].replace('.', '').replace(',', '.')
    eg_deuda = float(value)

    
    # Build the payload with all row data
    payload = {
        "Action": "Edit",
        "Properties": {
            "Locale": "en-US",
            "Timezone": "Argentina Standard Time"
        },
        "Rows": [{
            "ID": "hola",
            "ITEC DEUDA AFIP": itec_deuda,
            "ENERGÍA DEUDA AFIP": eg_deuda
        }]
    }

    #print(payload)

    # Define the request URL
    requestURL = f"https://api.appsheet.com/api/v2/apps/{appsheet_id}/tables/RESUMEN/Action"

    # Set request headers
    headers = {
        "Content-Type": "application/json",
        "ApplicationAccessKey": appsheet_key
    }

    # Send the request
    response = requests.post(requestURL, data=json.dumps(payload), headers=headers)

    # Check the response status code
    if response.status_code != 200:
        print(f"Request failed with status code: {response.status_code}")
        print(response.text)
        errorflag = False

truncate()
login()
periodo()

cuit(1)
scrape_clientes("ENERGÍA")
scrape_ventas("ENERGÍA")
scrape_proveedores("ENERGÍA")
scrape_compras("ENERGÍA")

cuit(2)
scrape_clientes("ITEC")
scrape_ventas("ITEC")
scrape_proveedores("ITEC")
scrape_compras("ITEC")

update() 

login_afip()
scrape_deudas()


if error_flag:
    print("success: data scraped and updated")

# add a post request to an api to notify the process finished
now = time.time().strftime("%Y-%m-%d %H:%M:%S")
requests.get(
    f"http://servicio.smsmasivos.com.ar/enviar_sms.asp?api=1&apikey=p6hery1lu5i2ngp6lceafxpm3u6lj69v841gmp4gwnmgbuvqp33oafocryekqumaj6syrb4vevvejgi24458y8ga0gw4fvgn2okk&tos=1131500591&texto=Hola%21+Se+actualizaron+los+datos+de+facturacion+de+Energia+Global,+por+favor+verificarlos.+{now}"
)

driver.quit()


"""

"SELECT(CAJA CENTRAL[SALDO ACUM], [FECHA]=MAXROW("CAJA CENTRAL", "FECHA"))"

"ANY(SELECT(CAJA CENTRAL[SALDO ACUM], [FECHA]=MAXROW("CAJA CENTRAL", "FECHA", and(ISNOTBLANK([SALDO ACUM]), [SALDO ACUM]<>0))))"

SELECCIONA EL SALDO DE LA FILA CON LA FECHA MAS ALTA CON UN SALDO 
EN LA CAJA CENTRAL

"""