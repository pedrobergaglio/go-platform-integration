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
table_names = 'STOCK'
querys = f"SELECT * FROM {table_names}"

# Execute the query
cursor.execute(querys)

# Fetch all rows and column names
datas = cursor.fetchall()
column_namess = [i[0] for i in cursor.description]

# Create a Pandas DataFrame from the fetched data
df_stock = pd.DataFrame(datas, columns=column_namess)

print("Started to search")

# Process the data using a loop
for index, row in df_stock.iterrows():

    flag = 0 
    # Perform your processing logic here
    # Modify the values in the row as needed
    for indx, rw in df_productos.iterrows():

        if df_productos.at[indx, 'PRODUCTO'] == df_stock.at[index, 'PRODUCTO']:  
            
            flag = 1
            print("Found")

    # Si no encontró el producto
    if flag == 0:
        # Insert the item in the productos table
        insert_query = f"INSERT INTO PRODUCTOS ({', '.join(column_namess)}) VALUES ({', '.join(['%s']*len(column_namess))})"
        values = tuple(row[column] for column in column_namess)
        cursor.execute(insert_query, values)

        print(f"Product added to PRODUCTOS: {df_stock.at[index, 'PRODUCTO']}")
        
        # Now lets update the 'MARCA' column

        # Refresh the products dataframe:

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

        print("Products refreshed")

        for jindx, jrw in df_marcas.iterrows():

            # Create code
            if df_marcas.at[jindx, 'MARCA'] == df_stock.at[index, 'MARCA']:  
                marca_id = int(df_marcas.at[jindx, 'CODIGO_MARCA'])*100000
                incode = random.randint(0, 99999)
                product_code = marca_id + incode

                # Search product and send update query
                for indx, rw in df_productos.iterrows():

                    if df_productos.at[indx, 'PRODUCTO'] == df_stock.at[index, 'PRODUCTO']:

                        print("Product found to update marca")

                        df_productos.at[indx, 'CODIGO'] = product_code  

                        # Update
            
                        values = tuple(row[column] for column in column_names)
    
                        # Generate the update query
                        update_query = f"UPDATE {table_name} SET "
                        update_query += f"MARCA = {product_code}"
                        update_query += f" WHERE {column_names[3]} = %s"  # Assuming the first column is the primary key
                        
                        # Append the primary key value to the tuple of values
                        values += (rw[column_names[3]],)
                        
                        # Execute the update query
                        cursor.execute(update_query, values)

                        print(f"Product updated: {df_stock.at[index, 'PRODUCTO']}")
        break

# Commit the changes and close the connection
cnx.commit()
cursor.close()
cnx.close()