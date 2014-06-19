package export

import (
	"encoding/base64"
	"fmt"
	"mime"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	f "github.com/microcosm-cc/export-schemas/go/forum"
)

type vbAttachment struct {
	AttachmentID int64
	DateCreated  int64
	PostID       int64
	UserID       int64
	FileName     string
	FileSize     int64
	FileData     []byte
	FileHash     string
	Extension    string
	MimeType     string
	Visible      int
}

func exportAttachments() {

	exportedItems = f.DirIndex{}
	exportedItems.Type = f.AttachmentsPath

	if !fileExists(config.Export.OutputDirectory + f.AttachmentsPath) {
		handleErr(mkDirAll(config.Export.OutputDirectory + f.AttachmentsPath))
	}

	rows, err := db.Query(`
SELECT attachmentid
  FROM ` + config.DB.TablePrefix + `attachment
 ORDER BY attachmentid ASC`)
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

	fmt.Println("Exporting attachments")
	runDBTasks(ids, exportAttachment)

	handleErr(writeFile(
		config.Export.OutputDirectory+f.AttachmentsPath+"index.json",
		exportedItems,
	))
}

func exportAttachment(id int64) error {

	// Split the filename and ensure the directory exists
	path, name := splitFilename(strconv.FormatInt(id, 10))
	path = config.Export.OutputDirectory + f.AttachmentsPath + path

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

	vb := vbAttachment{}
	err := db.QueryRow(`
SELECT a.attachmentid
      ,a.dateline
      ,a.postid
      ,a.userid
      ,a.filename
      ,a.filesize
      ,a.filedata
      ,a.filehash
      ,a.extension
      ,t.mimetype
      ,a.visible
  FROM `+config.DB.TablePrefix+`attachment a
  JOIN `+config.DB.TablePrefix+`attachmenttype t ON t.extension = a.extension
 WHERE a.attachmentid = ?`,
		id,
	).Scan(
		&vb.AttachmentID,
		&vb.DateCreated,
		&vb.PostID,
		&vb.UserID,
		&vb.FileName,
		&vb.FileSize,
		&vb.FileData,
		&vb.FileHash,
		&vb.Extension,
		&vb.MimeType,
		&vb.Visible,
	)
	if err != nil {
		return err
	}

	ex := f.Attachment{}
	ex.ID = vb.AttachmentID
	ex.Author = vb.UserID
	ex.DateCreated = time.Unix(vb.DateCreated, 0).UTC()
	ex.Associations = append(ex.Associations, f.Association{
		OnType: "comment",
		OnID:   vb.PostID,
	})
	ex.Name = vb.FileName
	ex.ContentSize = vb.FileSize
	ex.MimeType = getMimeTypeFromFileName("name." + vb.Extension)
	ex.ContentURL = "data:" + ex.MimeType + ";base64," +
		base64.StdEncoding.EncodeToString(vb.FileData)

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

func getMimeTypeFromFileName(m string) string {
	ext := filepath.Ext(m)
	if ext == "" {
		return "unknown/unknown"
	}

	mimetype := mime.TypeByExtension(ext)
	if mimetype == "" {
		return "unknown/unknown"
	}

	return mimetype
}
