package export

import (
	"database/sql"
	"fmt"

	_ "github.com/go-sql-driver/mysql"
)

var (
	db *sql.DB
)

type lock struct{}

func GetConnection() {
	var err error
	db, err = sql.Open("mysql", config.DB.ConnectionString)
	HandleErr(err)

	var maxConn int
	HandleErr(db.QueryRow("SELECT @@max_connections").Scan(&maxConn))

	fmt.Printf("MySQL MaxConns: %d\n", maxConn)
}
