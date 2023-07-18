import pandas as pd
import mysql.connector
import random

# MySQL database configuration
config = {
    'user': 'megared_pedro',
    'password': 'Engsu_23',
    'host': 'Mysql4.gohsphere.com',
    'database': 'megared_energiaglobal_23',
    'charset': 'utf8'
}

# Connect to the MySQL database
cnx = mysql.connector.connect(**config)
cursor = cnx.cursor()


# Query to retrieve data from the table
table_name = 'PRODUCTOS'
query = f"SELECT * FROM {table_name}"

# Execute the query
cursor.execute(query)

# Fetch all rows and column names
data = cursor.fetchall()
column_names = [i[0] for i in cursor.description]

# Create a Pandas DataFrame from the fetched data
df_productos = pd.DataFrame(data, columns=column_names)

# Query to retrieve data from the table
table_namem = 'BRANDS'
querym = f"SELECT * FROM {table_namem}"

# Execute the query
cursor.execute(querym)

# Fetch all rows and column names
data_marcas = cursor.fetchall()
column_names_marcas = [i[0] for i in cursor.description]

# Create a Pandas DataFrame from the fetched data
df_marcas = pd.DataFrame(data_marcas, columns=column_names_marcas)

# Query to retrieve data from the table
table_names = 'STOCK'
querys = f"SELECT * FROM {table_names}"

# Execute the query
cursor.execute(querys)

# Fetch all rows and column names
datastock = cursor.fetchall()
column_names_stock = [i[0] for i in cursor.description]

# Create a Pandas DataFrame from the fetched data
df_stock = pd.DataFrame(datastock, columns=column_names_stock)

# Query to retrieve data from the table
table_names = 'MOVIMIENTOS'
querys = f"SELECT * FROM {table_names}"

# Execute the query
cursor.execute(querys)

# Fetch all rows and column names
datastock = cursor.fetchall()
column_names_stock = [i[0] for i in cursor.description]

# Create a Pandas DataFrame from the fetched data
df_movimientos = pd.DataFrame(datastock, columns=column_names_stock)

# Query to retrieve data from the table
table_names = 'MOVEMENTS'
querys = f"SELECT * FROM {table_names}"

# Execute the query
cursor.execute(querys)

# Fetch all rows and column names
datastock = cursor.fetchall()
column_names_stock = [i[0] for i in cursor.description]

# Create a Pandas DataFrame from the fetched data
df_movements = pd.DataFrame(datastock, columns=column_names_stock)

print("Started to search")

flag = 0

# Process the data using a loop
#for index, row in df_productos.iterrows():

    # Perform your processing logic here
    # Modify the values in the row as needed
for indx, rw in df_movimientos.iterrows():

    # Generate the update query with a parameter
    update_query = "UPDATE MOVEMENTS SET product_id = %s WHERE product = %s"

    for index, row in df_movements.iterrows():
        if df_movements.at[index, 'product'] == df_movimientos.at[indx, 'product']:
            flag = 1

            # Execute the update query with the parameter values
            cursor.execute(update_query, (str(df_movimientos.at[index, 'product_id']), df_movements.at[index, 'product']))
            
            print(f"{df_movements.at[index, 'product']} updated : {indx}")


    if flag == 0 :
        print(f"product not found: {df_movements.at[index, 'product']}")
    
# Commit the changes and close the connection

cnx.commit()
cursor.close()
cnx.close()