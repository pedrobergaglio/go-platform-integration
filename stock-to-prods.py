import mysql.connector
from tqdm import tqdm

# Connect to the MySQL database
conn = mysql.connector.connect(
    user= 'megared_pedro',
    password= 'Engsu_23',
    host= 'Mysql4.gohsphere.com',
    database= 'megared_energiaglobal_23',
    charset= 'utf8'
)

# Create a cursor object to execute SQL queries
cursor = conn.cursor()

# Select all rows from the STOCK table
select_query = "SELECT * FROM STOCK"
cursor.execute(select_query)
stock_rows = cursor.fetchall()

# Iterate over each row in the STOCK table
for i, stock_row in tqdm(enumerate(stock_rows)):
        # Skip rows until the starting row is reached
    if i < 1410:
        continue

    # Extract the necessary values from the STOCK row
    proveedor = stock_row[0]
    marca = stock_row[1]
    codigo = stock_row[2]
    producto = stock_row[3]
    fabrica = stock_row[4]
    ORÁN = stock_row[5]
    RODRIGUEz = stock_row[6]
    MARCOS_PAz = stock_row[7]
    TOTAl = stock_row[8]
    COSTo = stock_row[9]
    VALOr = stock_row[10]
    """WC_CODIGo = stock_row[11]
    ML_CODIGo = stock_row[12]
    GLOBAl = stock_row[13]
    CODIGO_Eg = stock_row[14]
    CODIGO_FABRICANTe = stock_row[15]"""
    # ... continue extracting other column values as needed
    
    # Check if the row exists in the PRODUCTOS table
    select_exists_query = "SELECT * FROM PRODUCTOS WHERE PRODUCTO = %s"
    cursor.execute(select_exists_query, (producto,))
    exists_row = cursor.fetchone()
    
    if not exists_row:
        print(f"producto no encontrado {producto}")
        # Row doesn't exist in PRODUCTOS, insert it
        insert_query = "INSERT INTO PRODUCTOS (PROVEEDOR, MARCA, CODIGO, PRODUCTO, FÁBRICA, `ORÁN`, RODRIGUEZ, MARCOS_PAZ, TOTAL, COSTO, VALOR, WC_CODIGO, ML_CODIGO, `GLOBAL?`, `CODIGO-EG`, `CODIGO_FABRICANTE`) VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s)"
        values = (proveedor, marca, codigo, producto, fabrica, ORÁN, RODRIGUEz, MARCOS_PAz, TOTAl, COSTo, VALOr, 0, 0, 0, 0, 0)
        cursor.execute(insert_query, values)
        conn.commit()
    

# Close the cursor and connection
cursor.close()
conn.close()