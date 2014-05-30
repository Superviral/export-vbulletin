package export

import (
	"fmt"
	"strconv"
	"time"

	f "github.com/microcosm-cc/export-schemas/go/forum"
)

const messageDir = `messages/`

type vbPM struct {
	PMTextID    int64
	FromUserID  int64
	Title       string
	Message     string
	DateCreated int64
}

type vbPMReceipient struct {
	UserID      int64
	FolderID    int64
	MessageRead int64
}

func exportMessages() {

	if !fileExists(config.Export.OutputDirectory + messageDir) {
		handleErr(mkDirAll(config.Export.OutputDirectory + messageDir))
	}

	rows, err := db.Query(`
SELECT pmtextid
  FROM ` + config.DB.TablePrefix + `pmtext
 ORDER BY pmtextid ASC`)
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

	fmt.Println("Exporting messages")
	runDBTasks(ids, exportMessage)
}

func exportMessage(id int64) error {

	// Split the filename and ensure the directory exists
	path, name := splitFilename(strconv.FormatInt(id, 10))
	path = config.Export.OutputDirectory + messageDir + path

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

	vb := vbPM{}
	err := db.QueryRow(`
SELECT pmtextid
      ,fromuserid
      ,title
      ,message
      ,dateline
  FROM `+config.DB.TablePrefix+`pmtext
 WHERE pmtextid = ?`,
		id,
	).Scan(
		&vb.PMTextID,
		&vb.FromUserID,
		&vb.Title,
		&vb.Message,
		&vb.DateCreated,
	)
	if err != nil {
		return err
	}

	// -- every PM that hasn't been deleted creates a row in vb_pm
	// SELECT userid
	//       ,folderid
	//       ,messageread
	//   FROM vb_pm
	//  WHERE pmtextid = 1458652;

	// -- folderid = -1 = sent
	// -- folderid >= 0 = received
	// -- messageread = 0 = unread
	// -- messageread = 0 = read

	c := f.CommentVersion{}
	c.DateModified = time.Unix(vb.DateCreated, 0).UTC()
	c.Editor = vb.FromUserID
	c.Headline = vb.Title
	c.Text = vb.Message

	ex := f.Message{}

	ex.ID = vb.PMTextID
	ex.Author = vb.FromUserID
	ex.DateCreated = time.Unix(vb.DateCreated, 0).UTC()
	ex.Versions = append(ex.Versions, c)

	// ex.OnType = "conversation"
	// ex.OnID = vb.ThreadID
	// ex.InReplyTo = vb.ParentID
	// ex.Author = vb.UserID
	// ex.IPAddress = vb.IPAddress
	// ex.DateCreated = time.Unix(vb.DateCreated, 0).UTC()
	// ex.Moderated = (vb.Visible == 0)
	// ex.Deleted = (vb.Visible == 2)

	// 	version := f.CommentVersion{}
	// 	version.DateModified = time.Unix(vb.DateCreated, 0).UTC()
	// 	version.Headline = vb.Title
	// 	version.Text = vb.PageText
	// 	version.IPAddress = vb.IPAddress
	// 	version.Editor = vb.UserID
	// 	ex.Versions = append(ex.Versions, version)

	err = writeFile(filename, ex)
	if err != nil {
		return err
	}

	return nil
}
