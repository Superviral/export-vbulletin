package export

import (
	"database/sql"
	"fmt"
	"sync"

	"github.com/cheggaaa/pb"

	_ "github.com/go-sql-driver/mysql"
)

var (
	db         *sql.DB
	dbMaxConns int
)

func GetConnection() {
	var err error
	db, err = sql.Open("mysql", config.DB.ConnectionString)
	HandleErr(err)

	HandleErr(db.QueryRow("SELECT @@max_connections").Scan(&dbMaxConns))

	spareConns := 10
	if dbMaxConns > spareConns {
		dbMaxConns = dbMaxConns - spareConns
	}

	db.SetMaxOpenConns(dbMaxConns)

	fmt.Printf("MySQL MaxConns: %d\n", dbMaxConns)
}

func GetGophers(tasks int) int {
	if tasks > dbMaxConns {
		return dbMaxConns
	} else {
		return tasks
	}
}

func RunDBTasks(ids []int64, task func(int64) error) {

	bar := pb.StartNew(len(ids))

	tasks := make(chan int64, len(ids)+1)
	var errs []error

	var wg sync.WaitGroup
	for i := 0; i < GetGophers(len(ids)); i++ {
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
		HandleErr(err)
	}
}
