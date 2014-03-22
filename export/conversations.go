package export

import (
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/cheggaaa/pb"

	"github.com/microcosm-cc/export-schemas/go/forum"
)

type vbThread struct {
	ThreadId    int64
	Title       string
	Prefix      string
	ForumId     int64
	Open        int64
	DateCreated int64
	Views       int64
	Visible     int64
	Sticky      int64
	PollId      int64
}

func ExportConversations() {

	exportDir := fmt.Sprintf("%sconversations/", config.OutputDirectory)
	if !FileExists(exportDir) {
		HandleErr(MkDirAll(exportDir))
	}

	stmt, err := db.Prepare(`
SELECT threadid
  FROM ` + config.DB.TablePrefix + `thread
 ORDER BY threadid ASC`)
	HandleErr(err)
	defer stmt.Close()

	rows, err := stmt.Query()
	HandleErr(err)
	defer rows.Close()

	ids := []int64{}
	for rows.Next() {
		var id int64
		HandleErr(rows.Scan(&id))
		ids = append(ids, id)
	}
	HandleErr(rows.Err())

	fmt.Println("Exporting conversations")
	bar := pb.StartNew(len(ids))

	var wg sync.WaitGroup
	wg.Add(len(ids))

	errs := make(chan error)
	defer close(errs)

	go func(exportDir string) {
		for _, id := range ids {
			errs <- ExportConversation(id, exportDir)
			wg.Done()
		}
	}(exportDir)

	for i := 0; i < len(ids); i++ {
		err := <-errs
		HandleErr(err)

		bar.Increment()
	}
	bar.Finish()

	wg.Wait()
}

func ExportConversation(id int64, exportDir string) error {

	// Split the filename and ensure the directory exists
	path, name := SplitFilename(strconv.FormatInt(id, 10))
	path = exportDir + path

	if !FileExists(path) {
		err := MkDirAll(path)
		if err != nil {
			return err
		}
	}

	filename := fmt.Sprintf("%s/%s.json", path, name)

	// Don't export if we've exported already

	if FileExists(filename) {
		return nil
	}

	stmt, err := db.Prepare(`
SELECT t.threadid
      ,t.title
      ,COALESCE(p.text, '') AS prefix
      ,t.forumid
      ,t.open
      ,t.dateline
      ,t.views
      ,t.visible
      ,t.sticky
      ,t.pollid
  FROM ` + config.DB.TablePrefix + `thread t
       LEFT JOIN ` + config.DB.TablePrefix + `phrase p ON
            p.varname = CONCAT('prefix_', t.prefixid, '_title_plain')
 WHERE t.threadid = ?`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	vb := vbThread{}
	err = stmt.QueryRow(id).Scan(
		&vb.ThreadId,
		&vb.Title,
		&vb.Prefix,
		&vb.ForumId,
		&vb.Open,
		&vb.DateCreated,
		&vb.Views,
		&vb.Visible,
		&vb.Sticky,
		&vb.PollId,
	)
	if err != nil {
		return err
	}

	ex := forum.Conversation{}

	ex.Id = vb.ThreadId
	ex.ForumId = vb.ForumId

	if vb.Prefix == "" {
		ex.Name = vb.Title
	} else {
		ex.Name = vb.Prefix + " " + vb.Title
	}

	ex.Open = (vb.Open == 1 || vb.Open == 10)
	ex.DateCreated = time.Unix(vb.DateCreated, 0).UTC()
	ex.ViewCount = vb.Views
	ex.Moderated = (vb.Visible == 0)
	ex.Deleted = (vb.Visible == 2)
	ex.Sticky = (vb.Sticky == 1)

	err = WriteFile(filename, ex)
	if err != nil {
		return err
	}

	return nil
}
