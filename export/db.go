package export

import (
	"database/sql"
	"fmt"
	"sync"

	"github.com/cheggaaa/pb"

	// mysql driver, needed for the connection
	_ "github.com/go-sql-driver/mysql"
)

var (
	db         *sql.DB
	dbMaxConns int
)

func getConnection() {
	var err error
	db, err = sql.Open("mysql", config.DB.ConnectionString)
	handleErr(err)

	handleErr(db.QueryRow("SELECT @@max_connections").Scan(&dbMaxConns))

	db.SetMaxOpenConns(dbMaxConns)

	fmt.Printf("MySQL MaxConns: %d\n", dbMaxConns)
}

func getGophers(tasks int) int {
	if tasks > dbMaxConns {
		switch {
		case dbMaxConns < 10:
			return dbMaxConns
		default:
			return int(float64(dbMaxConns) * 0.75)
		}
	} else {
		return tasks
	}
}

func runDBTasks(ids []int64, task func(int64) error) {

	bar := pb.StartNew(len(ids))

	tasks := make(chan int64, len(ids)+1)
	var errs []error

	var wg sync.WaitGroup
	for i := 0; i < getGophers(len(ids)); i++ {
		wg.Add(1)

		go func() {
			for id := range tasks {
				err := task(id)
				if err != nil {
					errs = append(errs, err)
				}
				bar.Increment()
			}
			wg.Done()
		}()
	}

	for _, id := range ids {
		tasks <- id
	}
	close(tasks)

	wg.Wait()
	bar.Finish()

	for _, err := range errs {
		handleErr(err)
	}
}
