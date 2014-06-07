package export

import (
	"database/sql"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
	"time"

	f "github.com/microcosm-cc/export-schemas/go/forum"
)

const userDir = `users/`

type vbUser struct {
	UserID         int64
	UserGroupID    int64
	MemberGroupIDs string
	Username       string
	Email          string
	JoinDate       int64
	LastVisit      int64
	LastActivity   int64
	IPAddress      string
	Banned         bool
	Options        int64
}

type vbAvatar struct {
	DateCreated int64
	FileName    string
	FileSize    int64
	FileData    []byte
	Width       int64
	Height      int64
	Visible     int
}

func exportProfiles() {

	if !fileExists(config.Export.OutputDirectory + userDir) {
		handleErr(mkDirAll(config.Export.OutputDirectory + userDir))
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

	fmt.Println("Exporting profiles")
	runDBTasks(ids, exportProfile)
}

func exportProfile(id int64) error {

	// Split the filename and ensure the directory exists
	path, name := splitFilename(strconv.FormatInt(id, 10))
	path = config.Export.OutputDirectory + userDir + path

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

	// Fetch the user
	vb := vbUser{}
	err := db.QueryRow(`
SELECT u.userid
      ,u.usergroupid
      ,u.membergroupids
      ,u.username
      ,u.email
      ,u.joindate
      ,u.lastvisit
      ,u.lastactivity
      ,u.ipaddress
      ,IF(ub.userid, true, false) AS banned
      ,u.options
  FROM `+config.DB.TablePrefix+`user u
       LEFT JOIN `+config.DB.TablePrefix+`userban ub ON u.userid = ub.userid
 WHERE u.userid = ?`,
		id,
	).Scan(
		&vb.UserID,
		&vb.UserGroupID,
		&vb.MemberGroupIDs,
		&vb.Username,
		&vb.Email,
		&vb.JoinDate,
		&vb.LastVisit,
		&vb.LastActivity,
		&vb.IPAddress,
		&vb.Banned,
		&vb.Options,
	)
	if err != nil {
		return err
	}

	// Map the user into our structure performing any translations needed

	ex := f.Profile{}
	ex.ID = vb.UserID
	ex.Name = vb.Username
	ex.Email = vb.Email
	ex.DateCreated = time.Unix(vb.JoinDate, 0).UTC()
	ex.LastActive = time.Unix(vb.LastVisit, 0).UTC()
	ex.IPAddress = vb.IPAddress
	ex.Banned = vb.Banned

	usergroups := []f.ID{}
	usergroups = append(usergroups, f.ID{ID: vb.UserGroupID})
	if vb.MemberGroupIDs != "" {
		groups := strings.Split(vb.MemberGroupIDs, ",")
		for _, group := range groups {
			groupID, err := strconv.ParseInt(strings.Trim(group, " "), 10, 64)
			if err != nil {
				return err
			}
			usergroups = append(usergroups, f.ID{ID: groupID})
		}
	}
	ex.Usergroups = usergroups

	// From vBulletin includes/xml/bitfield_vbulletin.xml
	// <group name="useroptions">
	// 	<bitfield name="showsignatures">1</bitfield>
	// 	<bitfield name="showavatars">2</bitfield>
	// 	<bitfield name="showimages">4</bitfield>
	// 	<bitfield name="coppauser">8</bitfield>
	// 	<bitfield name="adminemail">16</bitfield>
	// 	<bitfield name="showvcard">32</bitfield>
	// 	<bitfield name="dstauto">64</bitfield>
	// 	<bitfield name="dstonoff">128</bitfield>
	// 	<bitfield name="showemail">256</bitfield>
	// 	<bitfield name="invisible">512</bitfield>
	// 	<bitfield name="showreputation">1024</bitfield>
	// 	<bitfield name="receivepm">2048</bitfield>
	// 	<bitfield name="emailonpm">4096</bitfield>
	// 	<bitfield name="hasaccessmask">8192</bitfield>
	// 	<bitfield name="postorder">32768</bitfield>
	// 	<bitfield name="receivepmbuddies">131072</bitfield>
	// 	<bitfield name="noactivationmails">262144</bitfield>
	// 	<bitfield name="pmboxwarning">524288</bitfield>
	// 	<bitfield name="showusercss">1048576</bitfield>
	// 	<bitfield name="receivefriendemailrequest">2097152</bitfield>
	// 	<bitfield name="vm_enable">8388608</bitfield>
	// 	<bitfield name="vm_contactonly">16777216</bitfield>
	// 	<bitfield name="pmdefaultsavecopy">33554432</bitfield>
	// </group>
	ex.ReceiveEmailFromAdmins = vb.Options&16 != 0
	ex.ReceiveEmailNotifications = vb.Options&4096 != 0

	// Fetch the avatar
	vba := vbAvatar{}
	err = db.QueryRow(`
SELECT dateline
      ,filename
      ,filesize
      ,filedata
      ,width
      ,height
      ,visible
  FROM `+config.DB.TablePrefix+`customavatar
 WHERE userid = ?`,
		id,
	).Scan(
		&vba.DateCreated,
		&vba.FileName,
		&vba.FileSize,
		&vba.FileData,
		&vba.Width,
		&vba.Height,
		&vba.Visible,
	)
	if err != nil {
		if err != sql.ErrNoRows {
			return err
		}
		// No custom avatar
	} else {
		// We have a custom avatar
		exa := f.Attachment{}

		exa.Author = id
		exa.DateCreated = time.Unix(vba.DateCreated, 0).UTC()
		exa.Associations = append(exa.Associations, f.Association{
			OnType: "user",
			OnID:   id,
		})
		exa.Name = vba.FileName
		exa.ContentSize = vba.FileSize
		exa.MimeType = getMimeTypeFromFileName(vba.FileName)
		exa.ContentURL = "data:" + exa.MimeType + ";base64," +
			base64.StdEncoding.EncodeToString(vba.FileData)
		exa.Width = vba.Width
		exa.Height = vba.Height

		ex.Avatar = exa
	}

	// Write the user
	err = writeFile(filename, ex)
	if err != nil {
		return err
	}

	return nil
}
