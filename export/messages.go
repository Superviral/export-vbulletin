package export

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	f "github.com/microcosm-cc/export-schemas/go/forum"
)

type vbPM struct {
	PMTextID    int64
	FromUserID  int64
	Title       string
	Message     string
	DateCreated int64
}

type vbPMRecipient struct {
	UserID      int64
	FolderID    int64
	MessageRead int64
}

func exportMessages() {

	exportedItems = f.DirIndex{}
	exportedItems.Type = f.MessagesPath

	if !fileExists(config.Export.OutputDirectory + f.MessagesPath) {
		handleErr(mkDirAll(config.Export.OutputDirectory + f.MessagesPath))
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

	handleErr(writeFile(
		config.Export.OutputDirectory+f.MessagesPath+"index.json",
		exportedItems,
	))
}

func exportMessage(id int64) error {

	// Split the filename and ensure the directory exists
	path, name := splitFilename(strconv.FormatInt(id, 10))
	path = config.Export.OutputDirectory + f.MessagesPath + path

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

	ex := f.Message{}
	ex.ID = vb.PMTextID
	ex.Author = vb.FromUserID
	ex.DateCreated = time.Unix(vb.DateCreated, 0).UTC()

	ex.Versions = append(ex.Versions, f.CommentVersion{
		DateModified: time.Unix(vb.DateCreated, 0).UTC(),
		Editor:       vb.FromUserID,
		Headline:     vb.Title,
		Text:         vb.Message,
	})

	// -- every PM that hasn't been deleted creates a row in vb_pm
	rows, err := db.Query(`
SELECT userid
      ,folderid
      ,messageread
  FROM vb_pm
 WHERE pmtextid = ?`,
		id,
	)
	if err != nil {
		return err
	}
	defer rows.Close()

	rs := []vbPMRecipient{}
	for rows.Next() {
		r := vbPMRecipient{}
		err = rows.Scan(
			&r.UserID,
			&r.FolderID,
			&r.MessageRead,
		)
		if err != nil {
			return err
		}

		rs = append(rs, r)
	}
	err = rows.Err()
	if err != nil {
		return err
	}
	rows.Close()

	for _, r := range rs {
		recipient := f.MessageRecipient{}
		recipient.ID = r.UserID

		// -- messageread = 0 = unread
		// -- messageread = 1 = read
		recipient.Read = (r.MessageRead > 0)

		ex.To = append(ex.To, recipient)
	}

	err = writeFile(filename, ex)
	if err != nil {
		return err
	}

	exportedItemsLock.Lock()
	exportedItems.Files = append(exportedItems.Files, f.DirFile{
		ID:   ex.ID,
		Path: strings.Replace(filename, config.Export.OutputDirectory, "", 1),
	})
	exportedItemsLock.Unlock()

	return nil
}
