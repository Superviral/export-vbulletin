package export

import (
	"fmt"
	"strconv"
	"strings"

	f "github.com/microcosm-cc/export-schemas/go/forum"
)

func exportFollows() {

	exportedItems = f.DirIndex{}
	exportedItems.Type = f.FollowsPath

	if !fileExists(config.Export.OutputDirectory + f.FollowsPath) {
		handleErr(mkDirAll(config.Export.OutputDirectory + f.FollowsPath))
	}

	rows, err := db.Query(`
SELECT userid
  FROM ` + config.DB.TablePrefix + `user
 ORDER BY userid ASC`)
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

	fmt.Println("Exporting follows")
	runDBTasks(ids, exportFollow)

	handleErr(writeFile(
		config.Export.OutputDirectory+f.FollowsPath+"index.json",
		exportedItems,
	))
}

func exportFollow(id int64) error {

	// Split the filename and ensure the directory exists
	path, name := splitFilename(strconv.FormatInt(id, 10))
	path = config.Export.OutputDirectory + f.FollowsPath + path

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

	ex := f.Follow{}
	ex.Author = id

	// Forums
	rows, err := db.Query(`
SELECT forumid
      ,emailupdate
  FROM vb_subscribeforum
 WHERE userid = ?
 ORDER BY 2`,
		id,
	)
	if err != nil {
		return err
	}
	defer rows.Close()

	fns := []f.FollowNotify{}
	for rows.Next() {
		var i, e int64
		err := rows.Scan(&i, &e)
		if err != nil {
			return err
		}

		fn := f.FollowNotify{}
		fn.ID = i
		fn.Notify = (e >= 1)

		fns = append(fns, fn)
	}
	err = rows.Err()
	if err != nil {
		return err
	}
	rows.Close()
	ex.Forums = fns

	// Threads
	rows, err = db.Query(`
SELECT threadid
      ,emailupdate
  FROM vb_subscribethread
 WHERE userid = ?
 ORDER BY 2`,
		id,
	)
	if err != nil {
		return err
	}
	defer rows.Close()

	fns = []f.FollowNotify{}
	for rows.Next() {
		var i, e int64
		err := rows.Scan(&i, &e)
		if err != nil {
			return err
		}

		fn := f.FollowNotify{}
		fn.ID = i
		fn.Notify = (e >= 1)

		fns = append(fns, fn)
	}
	err = rows.Err()
	if err != nil {
		return err
	}
	rows.Close()
	ex.Conversations = fns

	// People
	rows, err = db.Query(`
SELECT userid
  FROM vb_user
 WHERE IF(
           FIND_IN_SET(
                userid,
                (
                     SELECT REPLACE(buddylist, ' ', ',')
                       FROM vb_usertextfield
                      WHERE userid = ?
                )
           )
           >=1,
           1,
           0
       )
 ORDER BY 1`,
		id,
	)
	if err != nil {
		return err
	}
	defer rows.Close()

	fns = []f.FollowNotify{}
	for rows.Next() {
		var i int64
		err := rows.Scan(&i)
		if err != nil {
			return err
		}

		fn := f.FollowNotify{}
		fn.ID = i

		fns = append(fns, fn)
	}
	err = rows.Err()
	if err != nil {
		return err
	}
	rows.Close()
	ex.Users = fns

	// Ignored people
	rows, err = db.Query(`
SELECT userid
  FROM vb_user
 WHERE IF(
           FIND_IN_SET(
                userid,
                (
                     SELECT REPLACE(ignorelist, ' ', ',')
                       FROM vb_usertextfield
                      WHERE userid = ?
                )
           )
           >=1,
           1,
           0
       )
 ORDER BY 1`,
		id,
	)
	if err != nil {
		return err
	}
	defer rows.Close()

	is := []int64{}
	for rows.Next() {
		var i int64
		err := rows.Scan(&i)
		if err != nil {
			return err
		}

		is = append(is, i)
	}
	err = rows.Err()
	if err != nil {
		return err
	}
	rows.Close()
	ex.UsersIgnored = is

	// -- ignored threads and forums
	// -- This table may not exist, it's a vBulletin.org hack
	rows, err = db.Query(`
SELECT threadid
      ,ig_type
  FROM vb_cis_thread_ignore
 WHERE userid = ?`,
		id,
	)
	if err != nil {
		return err
	}
	defer rows.Close()

	fi := []int64{}
	ci := []int64{}
	for rows.Next() {
		var (
			i int64
			t string
		)

		err := rows.Scan(&i, &t)
		if err != nil {
			return err
		}

		if t == "forum" {
			fi = append(fi, i)
		} else {
			ci = append(ci, i)
		}
	}
	err = rows.Err()
	if err != nil {
		return err
	}
	rows.Close()
	ex.ForumsIgnored = fi
	ex.ConversationsIgnored = ci

	err = writeFile(filename, ex)
	if err != nil {
		return err
	}

	exportedItemsLock.Lock()
	exportedItems.Files = append(exportedItems.Files, f.DirFile{
		ID:   id,
		Path: strings.Replace(filename, config.Export.OutputDirectory, "", 1),
	})
	exportedItemsLock.Unlock()

	return nil
}
