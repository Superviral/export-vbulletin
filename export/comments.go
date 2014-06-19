package export

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	f "github.com/microcosm-cc/export-schemas/go/forum"
)

type vbPost struct {
	PostID      int64
	ThreadID    int64
	ParentID    int64
	UserID      int64
	Title       string
	DateCreated int64
	PageText    string
	IPAddress   string
	Visible     int64
}

func exportComments() {

	exportedItems = f.DirIndex{}
	exportedItems.Type = f.CommentsPath

	if !fileExists(config.Export.OutputDirectory + f.CommentsPath) {
		handleErr(mkDirAll(config.Export.OutputDirectory + f.CommentsPath))
	}

	rows, err := db.Query(`
SELECT postid
  FROM ` + config.DB.TablePrefix + `post
 ORDER BY postid ASC`)
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

	fmt.Println("Exporting comments")
	runDBTasks(ids, exportComment)

	handleErr(writeFile(
		config.Export.OutputDirectory+f.CommentsPath+"index.json",
		exportedItems,
	))
}

func exportComment(id int64) error {

	// Split the filename and ensure the directory exists
	path, name := splitFilename(strconv.FormatInt(id, 10))
	path = config.Export.OutputDirectory + f.CommentsPath + path

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

	vb := vbPost{}
	err := db.QueryRow(`
SELECT postid
      ,threadid
      ,parentid
      ,userid
      ,title
      ,dateline
      ,pagetext
      ,ipaddress
      ,visible
  FROM `+config.DB.TablePrefix+`post
 WHERE postid = ?`,
		id,
	).Scan(
		&vb.PostID,
		&vb.ThreadID,
		&vb.ParentID,
		&vb.UserID,
		&vb.Title,
		&vb.DateCreated,
		&vb.PageText,
		&vb.IPAddress,
		&vb.Visible,
	)
	if err != nil {
		return err
	}

	ex := f.Comment{}
	ex.ID = vb.PostID
	ex.OnType = "conversation"
	ex.OnID = vb.ThreadID
	ex.InReplyTo = vb.ParentID
	ex.Author = vb.UserID
	ex.IPAddress = vb.IPAddress
	ex.DateCreated = time.Unix(vb.DateCreated, 0).UTC()
	ex.Moderated = (vb.Visible == 0)
	ex.Deleted = (vb.Visible == 2)

	version := f.CommentVersion{}
	version.DateModified = time.Unix(vb.DateCreated, 0).UTC()
	version.Headline = vb.Title
	version.Text = vb.PageText
	version.IPAddress = vb.IPAddress
	version.Editor = vb.UserID
	ex.Versions = append(ex.Versions, version)

	err = writeFile(filename, ex)
	if err != nil {
		return err
	}

	exportedItems.Files = append(exportedItems.Files, f.DirFile{
		ID:   ex.ID,
		Path: strings.Replace(filename, config.Export.OutputDirectory, "", 1),
	})

	return nil
}
