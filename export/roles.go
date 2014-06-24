package export

import (
	"fmt"
	"strconv"
	"strings"

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

	var (
		publicGroup    int64
		genericOptions int64
	)
	vb := vbUserGroup{}
	err := db.QueryRow(`
SELECT usergroupid
      ,title
      ,description
      ,forumpermissions
      ,ispublicgroup
      ,genericoptions
  FROM `+config.DB.TablePrefix+`usergroup
 WHERE usergroupid = ?`,
		id,
	).Scan(
		&vb.UserGroupID,
		&vb.Title,
		&vb.Description,
		&vb.ForumPermissions,
		&publicGroup,
		&genericOptions,
	)
	if err != nil {
		return err
	}

	ex := f.Role{}
	ex.ID = vb.UserGroupID
	ex.Name = vb.Title
	ex.Text = vb.Description

	if publicGroup == 0 {
		ex.DefaultRole = true
	}

	// From vBulletin includes/xml/bitfield_vbulletin.xml
	// <group name="genericoptions">
	//  <bitfield name="showgroup"         >1</bitfield>
	//  <bitfield name="showbirthday"      >2</bitfield>
	//  <bitfield name="showmemberlist"    >4</bitfield>
	//  <bitfield name="showeditedby"      >8</bitfield>
	//  <bitfield name="allowmembergroups" >16</bitfield>
	//  <bitfield name="isnotbannedgroup"  >32</bitfield>
	//  <bitfield name="requirehvcheck"    >64</bitfield>
	// </group>
	ex.Banned = genericOptions&32 == 0

	// From vBulletin includes/xml/bitfield_vbulletin.xml
	// <group name="forumpermissions">
	// 	<bitfield name="canview"               >1</bitfield>
	// 	<bitfield name="canviewthreads"        >524288</bitfield>
	// 	<bitfield name="canviewothers"         >2</bitfield>
	// 	<bitfield name="cansearch"             >4</bitfield>
	// 	<bitfield name="canemail"              >8</bitfield>
	// 	<bitfield name="canpostnew"            >16</bitfield>
	// 	<bitfield name="canreplyown"           >32</bitfield>
	// 	<bitfield name="canreplyothers"        >64</bitfield>
	// 	<bitfield name="caneditpost"           >128</bitfield>
	// 	<bitfield name="candeletepost"         >256</bitfield>
	// 	<bitfield name="candeletethread"       >512</bitfield>
	// 	<bitfield name="canopenclose"          >1024</bitfield>
	// 	<bitfield name="canmove"               >2048</bitfield>
	// 	<bitfield name="cangetattachment"      >4096</bitfield>
	// 	<bitfield name="canpostattachment"     >8192</bitfield>
	// 	<bitfield name="attachlimit"           >1</bitfield>
	// 	<bitfield name="canpostpoll"           >16384</bitfield>
	// 	<bitfield name="canvote"               >32768</bitfield>
	// 	<bitfield name="canthreadrate"         >65536</bitfield>
	// 	<bitfield name="followforummoderation" >131072</bitfield>
	// 	<bitfield name="canseedelnotice"       >262144</bitfield>
	// 	<bitfield name="cantagown"             >1048576</bitfield>
	// 	<bitfield name="cantagothers"          >2097152</bitfield>
	// 	<bitfield name="candeletetagown"       >4194304</bitfield>
	// 	<bitfield name="canseethumbnails"      >8388608</bitfield>
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

	exportedItemsLock.Lock()
	exportedItems.Files = append(exportedItems.Files, f.DirFile{
		ID:   ex.ID,
		Path: strings.Replace(filename, config.Export.OutputDirectory, "", 1),
	})
	exportedItemsLock.Unlock()

	return nil
}
