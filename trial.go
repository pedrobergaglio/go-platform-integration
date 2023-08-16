package main

/*

import (
	_ "github.com/go-sql-driver/mysql"
)

func main() {

	// Connect to the MySQL database
	db, err := sql.Open("mysql", "megared_pedro:Engsu_23@tcp(Mysql4.gohsphere.com)/megared_energiaglobal_23?charset=utf8")
	if err != nil {
		log.Fatal("error connecting to the database:", err)
	}
	defer db.Close()

	// Execute the query
	rows, err := db.Query(fmt.Sprintf("SELECT * FROM STOCK WHERE product_id = %s", "426301"))
	if err != nil {
		log.Fatal("error executing query:", err)
	} else {
		rows.Close()
	}

	for rows.Next() {
		var brand string
		if err := rows.Scan(&brand); err != nil {
			log.Fatal("error scanning row:", err)
		}

		// Now you have the 'brand' value from each row
		fmt.Println("Brand:", brand)
	}

	if err := rows.Err(); err != nil {
		log.Fatal("error retrieving rows:", err)
	}
}
*/
