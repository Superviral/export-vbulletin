package export

import (
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/cheggaaa/pb"

	"github.com/microcosm-cc/export-schemas/go/forum"
)

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

	exportDir := fmt.Sprintf("%scomments/", config.OutputDirectory)
	if !FileExists(exportDir) {
		HandleErr(MkDirAll(exportDir))
	}

	stmt, err := db.Prepare(`
SELECT postid
  FROM ` + config.DB.TablePrefix + `post
 ORDER BY postid ASC`)
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

	fmt.Println("Exporting comments")
	bar := pb.StartNew(len(ids))

	var wg sync.WaitGroup
	wg.Add(len(ids))

	errs := make(chan error)
	defer close(errs)

	go func(exportDir string) {
		for _, id := range ids {
			errs <- ExportComment(id, exportDir)
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

func ExportComment(id int64, exportDir string) error {

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
SELECT postid
      ,threadid
      ,parentid
      ,userid
      ,title
      ,dateline
      ,pagetext
      ,ipaddress
      ,visible
  FROM ` + config.DB.TablePrefix + `post
 WHERE postid = ?`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	vb := vbPost{}
	err = stmt.QueryRow(id).Scan(
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

	ex := forum.Comment{}
	ex.Id = vb.PostId
	ex.OnType = "conversation"
	ex.OnId = vb.ThreadId
	ex.InReplyTo = vb.ParentId
	ex.Author = vb.UserId
	ex.IpAddress = vb.IpAddress
	ex.DateCreated = time.Unix(vb.DateCreated, 0).UTC()
	ex.Moderated = (vb.Visible == 0)
	ex.Deleted = (vb.Visible == 2)

	version := forum.CommentVersion{}
	version.DateModified = time.Unix(vb.DateCreated, 0).UTC()
	version.Headline = vb.Title
	version.Text = vb.PageText
	version.IpAddress = vb.IpAddress
	version.Editor = vb.UserId
	ex.Versions = append(ex.Versions, version)

	err = WriteFile(filename, ex)
	if err != nil {
		return err
	}

	return nil
}
