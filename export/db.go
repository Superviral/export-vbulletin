package export

import (
	"database/sql"

	_ "github.com/go-sql-driver/mysql"
)

var db *sql.DB

func GetConnection() (*sql.DB, error) {
	return sql.Open("mysql", config.DB.ConnectionString)
}
