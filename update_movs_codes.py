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
table_namem = 'MARCAS'
querym = f"SELECT * FROM {table_namem}"

# Execute the query
cursor.execute(querym)

# Fetch all rows and column names
datam = cursor.fetchall()
column_namesm = [i[0] for i in cursor.description]

# Create a Pandas DataFrame from the fetched data
df_marcas = pd.DataFrame(datam, columns=column_namesm)

# Query to retrieve data from the table
table_nameT = 'MOVIMIENTOS'
queryT = f"SELECT * FROM {table_nameT}"

# Execute the query
cursor.execute(queryT)

# Fetch all rows and column names
dataT = cursor.fetchall()
column_namesT = [i[0] for i in cursor.description]

# Create a Pandas DataFrame from the fetched data
df_movs = pd.DataFrame(dataT, columns=column_namesT)

# Process the data using a loop
for index, row in df_movs.iterrows():
    flag = 0
    # Perform your processing logic here
    # Modify the values in the row as needed
    for indx, rw in df_productos.iterrows():
        if df_productos.at[indx, 'PRODUCTO'] == df_movs.at[index, 'PRODUCTO']:  
            flag = 1
            codigo = df_productos.at[indx, 'CODIGO']
            df_movs.at[index, 'CODIGO'] = codigo
    
    if flag == 0:
        print(f"No se encontró el codigo del producto: {df_movs.at[index, 'PRODUCTO']}")

    # Example: Multiply the 'quantity' column by 2
    #df.at[index, 'quantity'] = row['quantity'] * 2

print("Códigos modificados")


# Update the table in the MySQL database
for index, row in df_movs.iterrows():
    # Extract the values from the row
    values = tuple(row[column] for column in column_namesT)
    
    # Generate the update query
    update_query = f"UPDATE {table_nameT} SET "
    update_query += ', '.join([f"`{column}` = %s" for column in column_namesT])
    update_query += f" WHERE {column_namesT[3]} = %s"  # Assuming the first column is the primary key
    
    # Append the primary key value to the tuple of values
    values += (row[column_namesT[3]],)
    
    # Execute the update query
    cursor.execute(update_query, values)

    print(str(codigo) + " : " + df_movs.at[index, 'PRODUCTO'])

# Commit the changes and close the connection
cnx.commit()
cursor.close()
cnx.close()