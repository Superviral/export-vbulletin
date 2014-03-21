package export

import (
	"fmt"
	"strconv"
	"sync"

	"github.com/cheggaaa/pb"

	"github.com/microcosm-cc/export-schemas/go/forum"
)

type vbUserGroup struct {
	UserGroupId      int64
	Title            string
	Description      string
	ForumPermissions int64
}

func ExportUserGroups() {

	exportDir := fmt.Sprintf("%susergroups/", config.OutputDirectory)
	if !FileExists(exportDir) {
		HandleErr(MkDirAll(exportDir))
	}

	stmt, err := db.Prepare(`
SELECT usergroupid
  FROM ` + config.DB.TablePrefix + `usergroup
 ORDER BY usergroupid ASC`)
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

	fmt.Println("Exporting usergroups")

	bar := pb.StartNew(len(ids))

	var wg sync.WaitGroup
	wg.Add(len(ids))

	errs := make(chan error)
	defer close(errs)

	go func(exportDir string) {
		for _, id := range ids {
			errs <- ExportUserGroup(id, exportDir)
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

func ExportUserGroup(id int64, exportDir string) error {

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
SELECT usergroupid
      ,title
      ,description
      ,forumpermissions
  FROM ` + config.DB.TablePrefix + `usergroup
 WHERE usergroupid = ?`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	vb := vbUserGroup{}
	err = stmt.QueryRow(id).Scan(
		&vb.UserGroupId,
		&vb.Title,
		&vb.Description,
		&vb.ForumPermissions,
	)
	if err != nil {
		return err
	}

	ex := forum.Usergroup{}
	ex.Id = vb.UserGroupId
	ex.Name = vb.Title
	ex.Text = vb.Description
	ex.Banned = (vb.ForumPermissions == 0)

	// From vBulletin includes/xml/bitfield_vbulletin.xml
	// <group name="forumpermissions">
	// 	<bitfield name="canview"               group="forum_viewing_permissions"  >1</bitfield>
	// 	<bitfield name="canviewthreads"        group="forum_viewing_permissions"  >524288</bitfield>
	// 	<bitfield name="canviewothers"         group="forum_viewing_permissions"  >2</bitfield>
	// 	<bitfield name="cansearch"             group="forum_searching_permissions">4</bitfield>
	// 	<bitfield name="canemail"              group="forum_viewing_permissions"  >8</bitfield>
	// 	<bitfield name="canpostnew"            group="post_thread_permissions"    >16</bitfield>
	// 	<bitfield name="canreplyown"           group="post_thread_permissions"    >32</bitfield>
	// 	<bitfield name="canreplyothers"        group="post_thread_permissions"    >64</bitfield>
	// 	<bitfield name="caneditpost"           group="post_thread_permissions"    >128</bitfield>
	// 	<bitfield name="candeletepost"         group="post_thread_permissions"    >256</bitfield>
	// 	<bitfield name="candeletethread"       group="post_thread_permissions"    >512</bitfield>
	// 	<bitfield name="canopenclose"          group="post_thread_permissions"    >1024</bitfield>
	// 	<bitfield name="canmove"               group="post_thread_permissions"    >2048</bitfield>
	// 	<bitfield name="cangetattachment"      group="forum_viewing_permissions"  >4096</bitfield>
	// 	<bitfield name="canpostattachment"     group="attachment_permissions"     >8192</bitfield>
	// 	<bitfield name="attachlimit"           group="attachment_permissions"     >1</bitfield>
	// 	<bitfield name="canpostpoll"           group="poll_permissions"           >16384</bitfield>
	// 	<bitfield name="canvote"               group="poll_permissions"           >32768</bitfield>
	// 	<bitfield name="canthreadrate"         group="post_thread_permissions"    >65536</bitfield>
	// 	<bitfield name="followforummoderation" group="post_thread_permissions"    >131072</bitfield>
	// 	<bitfield name="canseedelnotice"       group="forum_viewing_permissions"  >262144</bitfield>
	// 	<bitfield name="cantagown"             group="post_thread_permissions"    >1048576</bitfield>
	// 	<bitfield name="cantagothers"          group="post_thread_permissions"    >2097152</bitfield>
	// 	<bitfield name="candeletetagown"       group="post_thread_permissions"    >4194304</bitfield>
	// 	<bitfield name="canseethumbnails"      group="forum_viewing_permissions"  >8388608</bitfield>
	// </group>
	perms := forum.ForumPermissions{}
	perms.View = vb.ForumPermissions&1 != 0
	perms.PostNew = vb.ForumPermissions&16 != 0
	perms.EditOwn = vb.ForumPermissions&128 != 0
	perms.EditOthers = false
	perms.DeleteOwn = vb.ForumPermissions&256 != 0
	perms.DeleteOthers = false
	perms.CloseOwn = vb.ForumPermissions&1024 != 0
	perms.OpenOwn = vb.ForumPermissions&1024 != 0
	ex.ForumPermissions = perms

	stmt2, err := db.Prepare(`
SELECT userid
  FROM ` + config.DB.TablePrefix + `user
 WHERE usergroupid = ?
    OR find_in_set(?, membergroupids) <> 0`)
	if err != nil {
		return err
	}
	defer stmt2.Close()

	rows, err := stmt2.Query(id, id)
	if err != nil {
		return err
	}
	defer rows.Close()

	ids := []forum.Id{}
	for rows.Next() {
		id := forum.Id{}
		err = rows.Scan(&id.Id)
		if err != nil {
			return err
		}
		ids = append(ids, id)
	}
	HandleErr(rows.Err())
	ex.Users = ids

	err = WriteFile(filename, ex)
	if err != nil {
		return err
	}

	return nil
}
