import mysql.connector

# Replace with your database connection details
db = mysql.connector.connect(
    user= 'megared_pedro',
    password= 'Engsu_23',
    host= 'Mysql4.gohsphere.com',
    database= 'megared_energiaglobal_23',
    charset= 'utf8'
)
cursor = db.cursor()

# Fetch data from MOVEMENTS and DATES tables
query = "SELECT DATE(datetime) AS Date, SUM(total_stock) AS TotalStock FROM MOVEMENTS GROUP BY Date"
cursor.execute(query)
result = cursor.fetchall()

print("started")

# Populate the new cumulative total stock table
cumulative_total = 0
for row in result:
    date, total_stock = row
    cumulative_total += total_stock
    print(date, total_stock)
    insert_query = "UPDATE DAILY_TOTAL_STOCK SET total_stock=%s WHERE date = %s"
    cursor.execute(insert_query, (cumulative_total, date))

print("done")

db.commit()
cursor.close()
db.close()
