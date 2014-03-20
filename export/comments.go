package export

import (
	"fmt"
)

type vbPost struct {
	PostId int64
}

func ExportComments() error {

	exportDir := fmt.Sprintf("%scomments/", config.OutputDirectory)
	if !FileExists(exportDir) {
		err := MkDirAll(exportDir)
		if err != nil {
			return err
		}
	}

	stmt, err := db.Prepare(`
SELECT postid
  FROM ` + config.DB.TablePrefix + `post
 ORDER BY postid ASC`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	rows, err := stmt.Query()
	if err != nil {
		return err
	}
	defer rows.Close()

	ids := []int64{}
	for rows.Next() {
		var id int64
		err = rows.Scan(&id)
		if err != nil {
			return err
		}
		ids = append(ids, id)
	}

	errs := make(chan error)
	for _, id := range ids {
		go func(id int64) {
			errs <- ExportComment(id)
		}(id)
	}

	err = <-errs
	if err != nil {
		close(errs)
		return err
	}

	return nil
}

func ExportComment(id int64) error {

	filename := fmt.Sprintf("%scomments/%d.json", config.OutputDirectory, id)

	if FileExists(filename) {
		return nil
	}

	stmt, err := db.Prepare(`
SELECT postid
  FROM ` + config.DB.TablePrefix + `post
 WHERE postid = ?`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	u := vbPost{}
	err = stmt.QueryRow(id).Scan(&u.PostId)
	if err != nil {
		return err
	}

	err = WriteFile(filename, u)
	if err != nil {
		return err
	}

	return nil
}
