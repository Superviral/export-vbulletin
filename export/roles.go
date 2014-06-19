package export

import (
	"fmt"
	"strconv"

	f "github.com/microcosm-cc/export-schemas/go/forum"
)

type vbUserGroup struct {
	UserGroupID      int64
	Title            string
	Description      string
	ForumPermissions int64
}

func exportRoles() {

	exportedItems = f.DirIndex{}
	exportedItems.Type = f.RolesPath

	if !fileExists(config.Export.OutputDirectory + f.RolesPath) {
		handleErr(mkDirAll(config.Export.OutputDirectory + f.RolesPath))
	}

	rows, err := db.Query(`
SELECT usergroupid
  FROM ` + config.DB.TablePrefix + `usergroup
 ORDER BY usergroupid ASC`)
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

	fmt.Println("Exporting usergroups")
	runDBTasks(ids, exportRole)

	handleErr(writeFile(
		config.Export.OutputDirectory+f.RolesPath+"index.json",
		exportedItems,
	))
}

func exportRole(id int64) error {

	// Split the filename and ensure the directory exists
	path, name := splitFilename(strconv.FormatInt(id, 10))
	path = config.Export.OutputDirectory + f.RolesPath + path

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

	vb := vbUserGroup{}
	err := db.QueryRow(`
SELECT usergroupid
      ,title
      ,description
      ,forumpermissions
  FROM `+config.DB.TablePrefix+`usergroup
 WHERE usergroupid = ?`,
		id,
	).Scan(
		&vb.UserGroupID,
		&vb.Title,
		&vb.Description,
		&vb.ForumPermissions,
	)
	if err != nil {
		return err
	}

	ex := f.Role{}
	ex.ID = vb.UserGroupID
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
	perms := f.ForumPermissions{}
	perms.View = vb.ForumPermissions&1 != 0
	perms.PostNew = vb.ForumPermissions&16 != 0
	perms.EditOwn = vb.ForumPermissions&128 != 0
	perms.EditOthers = false
	perms.DeleteOwn = vb.ForumPermissions&256 != 0
	perms.DeleteOthers = false
	perms.CloseOwn = vb.ForumPermissions&1024 != 0
	perms.OpenOwn = vb.ForumPermissions&1024 != 0
	ex.ForumPermissions = perms

	// If we are usergroup 1 or 2 then we are the default built-in ones and we
	// can handle that and get out.
	switch id {
	case 1:
		// Guests in vBulletin
		ex.IncludeGuests = true
		err = writeFile(filename, ex)
		return err
	case 2:
		// Registered users in vBulletin
		ex.IncludeRegistered = true
		err = writeFile(filename, ex)
		return err
	case 3, 4:
		// Awaiting email confirmation, covered by group id == 2
		return nil
	case 5, 6, 7:
		ex.Moderator = true
	case 8:
		ex.Banned = true
	}

	rows, err := db.Query(`
SELECT date
      ,posts
      ,strategy
  FROM `+config.DB.TablePrefix+`userpromotion
 WHERE joinusergroupid = ?`,
		id,
	)
	if err != nil {
		return err
	}
	defer rows.Close()

	type Promotion struct {
		Date     int64
		Posts    int64
		Strategy int64
	}
	promotions := []Promotion{}
	hasCriteria := false
	for rows.Next() {

		promotion := Promotion{}
		err = rows.Scan(
			&promotion.Date,
			&promotion.Posts,
			&promotion.Strategy,
		)
		if err != nil {
			return err
		}

		if promotion.Strategy != 16 {
			hasCriteria = true
		}

		promotions = append(promotions, promotion)
	}
	err = rows.Err()
	if err != nil {
		return err
	}
	rows.Close()

	// date,posts,strategy
	// 1,3,3
	// 30,1500,17

	if hasCriteria {
		// Export criteria
		var OrGroup int64
		ex.Criteria = []f.Criterion{}

		// Note that strategy translates to the following:
		//
		// 0 = Posts and Reputation and Date
		// 1 = Posts or Reputation or Date
		// 2 = (Posts and Reputation) or Date
		// 3 = Posts and (Reputation or Date)
		// 4 = (Posts or Reputation) and Date
		// 5 = Posts or (Reputation and Date)
		// 6 = Reputation and (Posts or Date)
		// 7 = Reputation or (Posts and Date)
		// 16 = Reputation
		// 17 = Posts
		// 18 = Join Date
		//
		// Based on the strategy we can ignore certain fields, i.e. strategy 17
		// means we only care about the number of posts
		//
		// We are ignoring reputation as a criteria and so the strategies are
		// equivalent to:
		//   null
		//     16
		//   comments
		//     17
		//   date
		//     18
		//   comments AND date
		//     0, 3, 4, 7
		//   comments OR date
		//     1, 2, 5, 6
		const (
			postsKey string = "comment_count"
			dateKey  string = "days_since_registering"
		)

		for _, promotion := range promotions {

			switch promotion.Strategy {
			case 17:
				// Just posts
				ex.Criteria = append(ex.Criteria, f.Criterion{
					OrGroup:   OrGroup,
					Key:       postsKey,
					Predicate: f.PredicateGreaterThanOrEquals,
					Value:     promotion.Posts,
				})
			case 18:
				// Just days since registering
				ex.Criteria = append(ex.Criteria, f.Criterion{
					OrGroup:   OrGroup,
					Key:       dateKey,
					Predicate: f.PredicateGreaterThanOrEquals,
					Value:     promotion.Date,
				})
			case 0, 3, 4, 7:
				// Posts AND days since registering
				ex.Criteria = append(ex.Criteria, f.Criterion{
					OrGroup:   OrGroup,
					Key:       postsKey,
					Predicate: f.PredicateGreaterThanOrEquals,
					Value:     promotion.Posts,
				})

				ex.Criteria = append(ex.Criteria, f.Criterion{
					OrGroup:   OrGroup,
					Key:       dateKey,
					Predicate: f.PredicateGreaterThanOrEquals,
					Value:     promotion.Date,
				})
			case 1, 2, 5, 6:
				// Posts OR days since registering
				ex.Criteria = append(ex.Criteria, f.Criterion{
					OrGroup:   OrGroup,
					Key:       postsKey,
					Predicate: f.PredicateGreaterThanOrEquals,
					Value:     promotion.Posts,
				})

				OrGroup++

				ex.Criteria = append(ex.Criteria, f.Criterion{
					OrGroup:   OrGroup,
					Key:       dateKey,
					Predicate: f.PredicateGreaterThanOrEquals,
					Value:     promotion.Date,
				})
			}

			OrGroup++
		}

	} else {
		// Export users
		rows, err = db.Query(`
SELECT userid
  FROM `+config.DB.TablePrefix+`user
 WHERE usergroupid = ?
    OR find_in_set(?, membergroupids) <> 0`,
			id,
			id,
		)
		if err != nil {
			return err
		}
		defer rows.Close()

		ids := []f.ID{}
		for rows.Next() {
			id := f.ID{}
			err = rows.Scan(&id.ID)
			if err != nil {
				return err
			}
			ids = append(ids, id)
		}
		err = rows.Err()
		if err != nil {
			return err
		}
		rows.Close()

		ex.Users = ids
	}

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
