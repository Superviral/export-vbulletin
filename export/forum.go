package export

import (
	"fmt"
	"strconv"

	f "github.com/microcosm-cc/export-schemas/go/forum"
)

const forumDir = `forums/`

type vbForum struct {
	ForumId      int64
	Title        string
	Description  string
	Options      int64
	DisplayOrder int64
}

func ExportForums() {

	if !FileExists(config.Export.OutputDirectory + forumDir) {
		HandleErr(MkDirAll(config.Export.OutputDirectory + forumDir))
	}

	// vBulletin has the notion of forums that are just hyperlinks to somewhere
	// else. We skip these are they are not forums.
	rows, err := db.Query(`
SELECT forumid
  FROM ` + config.DB.TablePrefix + `forum
 WHERE link = ''
 ORDER BY forumid ASC`)
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

	fmt.Println("Exporting forums")
	RunDBTasks(ids, ExportForum)
}

func ExportForum(id int64) error {

	// Split the filename and ensure the directory exists
	path, name := SplitFilename(strconv.FormatInt(id, 10))
	path = config.Export.OutputDirectory + forumDir + path

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

	vb := vbForum{}
	err := db.QueryRow(`
SELECT forumid
      ,title
      ,description
      ,options
      ,displayorder
  FROM `+config.DB.TablePrefix+`forum
 WHERE forumid = ?`,
		id,
	).Scan(
		&vb.ForumId,
		&vb.Title,
		&vb.Description,
		&vb.Options,
		&vb.DisplayOrder,
	)
	if err != nil {
		return err
	}

	ex := f.Forum{}
	ex.ID = vb.ForumId
	ex.Name = vb.Title
	ex.Text = vb.Description
	ex.DisplayOrder = vb.DisplayOrder

	// From vBulletin includes/xml/bitfield_vbulletin.xml
	// <group name="forumoptions">
	// 	<bitfield name="active">1</bitfield>
	// 	<bitfield name="allowposting">2</bitfield>
	// 	<bitfield name="cancontainthreads">4</bitfield>
	// 	<bitfield name="moderatenewpost">8</bitfield>
	// 	<bitfield name="moderatenewthread">16</bitfield>
	// 	<bitfield name="moderateattach">32</bitfield>
	// 	<bitfield name="allowbbcode">64</bitfield>
	// 	<bitfield name="allowimages">128</bitfield>
	// 	<bitfield name="allowhtml">256</bitfield>
	// 	<bitfield name="allowsmilies">512</bitfield>
	// 	<bitfield name="allowicons">1024</bitfield>
	// 	<bitfield name="allowratings">2048</bitfield>
	// 	<bitfield name="countposts">4096</bitfield>
	// 	<bitfield name="canhavepassword">8192</bitfield>
	// 	<bitfield name="indexposts">16384</bitfield>
	// 	<bitfield name="styleoverride">32768</bitfield>
	// 	<bitfield name="showonforumjump">65536</bitfield>
	// 	<bitfield name="prefixrequired">131072</bitfield>
	// </group>

	ex.Open = vb.Options&2 != 0
	ex.Moderated = vb.Options&16 != 0

	// Forum moderators
	rows, err := db.Query(`
SELECT userid
  FROM `+config.DB.TablePrefix+`moderator
 WHERE forumid = ?`,
		id,
	)
	if err != nil {
		return err
	}
	defer rows.Close()

	mods := []f.ID{}
	for rows.Next() {
		mod := f.ID{}
		err = rows.Scan(&mod.ID)
		if err != nil {
			return err
		}

		mods = append(mods, mod)
	}
	err = rows.Err()
	if err != nil {
		return err
	}
	rows.Close()
	ex.Moderators = mods

	// Forum specific usergroup permissions
	rows, err = db.Query(`
SELECT usergroupid
      ,forumpermissions
  FROM `+config.DB.TablePrefix+`forumpermission
 WHERE forumid = ?`,
		id,
	)
	if err != nil {
		return err
	}
	defer rows.Close()

	usergroups := []f.Usergroup{}
	for rows.Next() {
		var (
			usergroupid      int64
			forumpermissions int64
		)
		err = rows.Scan(
			&usergroupid,
			&forumpermissions,
		)
		if err != nil {
			return err
		}

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
		perms := f.ForumPermissions{}
		perms.View = forumpermissions&1 != 0
		perms.PostNew = forumpermissions&16 != 0
		perms.EditOwn = forumpermissions&128 != 0
		perms.EditOthers = false
		perms.DeleteOwn = forumpermissions&256 != 0
		perms.DeleteOthers = false
		perms.CloseOwn = forumpermissions&1024 != 0
		perms.OpenOwn = forumpermissions&1024 != 0

		ug := f.Usergroup{}
		ug.ID = usergroupid
		ug.ForumPermissions = perms

		usergroups = append(usergroups, ug)
	}
	err = rows.Err()
	if err != nil {
		return err
	}
	rows.Close()
	ex.Usergroups = usergroups

	err = WriteFile(filename, ex)
	if err != nil {
		return err
	}

	return nil
}