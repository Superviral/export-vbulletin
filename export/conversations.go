package export

import (
	"fmt"
	"strconv"
	"time"

	f "github.com/microcosm-cc/export-schemas/go/forum"
)

const conversationDir = `conversations/`

type vbThread struct {
	ThreadID    int64
	Title       string
	Prefix      string
	ForumID     int64
	Open        int64
	DateCreated int64
	Views       int64
	Visible     int64
	Sticky      int64
	PollID      int64
}

func exportConversations() {

	if !fileExists(config.Export.OutputDirectory + conversationDir) {
		handleErr(mkDirAll(config.Export.OutputDirectory + conversationDir))
	}

	rows, err := db.Query(`
SELECT threadid
  FROM ` + config.DB.TablePrefix + `thread
 ORDER BY threadid ASC`)
	handleErr(err)
	defer rows.Close()

	ids := []int64{}
	for rows.Next() {
		var id int64
		handleErr(rows.Scan(&id))
		ids = append(ids, id)
	}
	handleErr(rows.Err())
	rows.Close()

	fmt.Println("Exporting conversations")
	runDBTasks(ids, exportConversation)
}

func exportConversation(id int64) error {

	// Split the filename and ensure the directory exists
	path, name := splitFilename(strconv.FormatInt(id, 10))
	path = config.Export.OutputDirectory + conversationDir + path

	if !fileExists(path) {
		err := mkDirAll(path)
		if err != nil {
			return err
		}
	}

	filename := fmt.Sprintf("%s/%s.json", path, name)

	// Don't export if we've exported already

	if fileExists(filename) {
		return nil
	}

	vb := vbThread{}
	err := db.QueryRow(`
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
  FROM `+config.DB.TablePrefix+`thread t
       LEFT JOIN `+config.DB.TablePrefix+`phrase p ON
            p.varname = CONCAT('prefix_', t.prefixid, '_title_plain')
 WHERE t.threadid = ?`,
		id,
	).Scan(
		&vb.ThreadID,
		&vb.Title,
		&vb.Prefix,
		&vb.ForumID,
		&vb.Open,
		&vb.DateCreated,
		&vb.Views,
		&vb.Visible,
		&vb.Sticky,
		&vb.PollID,
	)
	if err != nil {
		return err
	}

	ex := f.Conversation{}

	ex.ID = vb.ThreadID
	ex.ForumID = vb.ForumID

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

	err = writeFile(filename, ex)
	if err != nil {
		return err
	}

	return nil
}
