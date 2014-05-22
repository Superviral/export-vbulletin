package export

import (
	"fmt"
	"strconv"
	"time"

	f "github.com/microcosm-cc/export-schemas/go/forum"
)

const commentDir = `comments/`

type vbPost struct {
	PostId      int64
	ThreadId    int64
	ParentId    int64
	UserId      int64
	Title       string
	DateCreated int64
	PageText    string
	IpAddress   string
	Visible     int64
}

func ExportComments() {

	if !FileExists(config.Export.OutputDirectory + commentDir) {
		HandleErr(MkDirAll(config.Export.OutputDirectory + commentDir))
	}

	rows, err := db.Query(`
SELECT postid
  FROM ` + config.DB.TablePrefix + `post
 ORDER BY postid ASC`)
	HandleErr(err)
	defer rows.Close()

	ids := []int64{}
	for rows.Next() {
		var id int64
		HandleErr(rows.Scan(&id))
		ids = append(ids, id)
	}
	HandleErr(rows.Err())
	rows.Close()

	fmt.Println("Exporting comments")
	RunDBTasks(ids, ExportComment)
}

func ExportComment(id int64) error {

	// Split the filename and ensure the directory exists
	path, name := SplitFilename(strconv.FormatInt(id, 10))
	path = config.Export.OutputDirectory + commentDir + path

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
		&vb.PostId,
		&vb.ThreadId,
		&vb.ParentId,
		&vb.UserId,
		&vb.Title,
		&vb.DateCreated,
		&vb.PageText,
		&vb.IpAddress,
		&vb.Visible,
	)
	if err != nil {
		return err
	}

	ex := f.Comment{}
	ex.ID = vb.PostId
	ex.OnType = "conversation"
	ex.OnID = vb.ThreadId
	ex.InReplyTo = vb.ParentId
	ex.Author = vb.UserId
	ex.IPAddress = vb.IpAddress
	ex.DateCreated = time.Unix(vb.DateCreated, 0).UTC()
	ex.Moderated = (vb.Visible == 0)
	ex.Deleted = (vb.Visible == 2)

	version := f.CommentVersion{}
	version.DateModified = time.Unix(vb.DateCreated, 0).UTC()
	version.Headline = vb.Title
	version.Text = vb.PageText
	version.IPAddress = vb.IpAddress
	version.Editor = vb.UserId
	ex.Versions = append(ex.Versions, version)

	err = WriteFile(filename, ex)
	if err != nil {
		return err
	}

	return nil
}
