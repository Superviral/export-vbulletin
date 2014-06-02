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

	if config.DB.Connections > 0 {
		return config.DB.Connections
	}

	if tasks > dbMaxConns {
		switch {
		case dbMaxConns < 10:
			return dbMaxConns
		default:
			// Use 3/4 of the available connections
			return int(float64(dbMaxConns) * 0.75)
		}
	} else {
		return tasks
	}
}

func runDBTasks(ids []int64, task func(int64) error) {

	// Progress bar
	bar := pb.StartNew(len(ids))

	// Cancel control
	done := make(chan struct{})
	quit := false

	// IDs to process, sent via channel
	tasks := make(chan int64, len(ids)+1)

	var errs []error
	var wg sync.WaitGroup

	// Only fire up a set number of worker processes
	for i := 0; i < getGophers(len(ids)); i++ {
		wg.Add(1)

		go func() {
			for id := range tasks {
				err := doTask(id, task, done)
				if err != nil {
					if !quit {
						close(done)
						quit = true
					}
					errs = append(
						errs,
						fmt.Errorf("Failed on ID %d : %+v", id, err),
					)
					break
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
	if !quit {
		close(done)
	}

	if len(errs) == 0 {
		bar.Finish()
	}

	for _, err := range errs {
		handleErr(err)
	}
}

func doTask(id int64, task func(int64) error, done <-chan struct{}) error {
	select {
	case <-done:
		return fmt.Errorf("task cancelled")
	default:
		if id == 0 {
			return fmt.Errorf("id zero")
		}
		return task(id)
	}
}
