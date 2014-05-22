package export

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	f "github.com/microcosm-cc/export-schemas/go/forum"
)

const userDir = `users/`

type vbUser struct {
	UserId         int64
	UserGroupId    int64
	MemberGroupIds string
	Username       string
	Email          string
	JoinDate       int64
	LastVisit      int64
	LastActivity   int64
	IpAddress      string
	Banned         bool
	Options        int64
}

func ExportUsers() {

	if !FileExists(config.Export.OutputDirectory + userDir) {
		HandleErr(MkDirAll(config.Export.OutputDirectory + userDir))
	}

	rows, err := db.Query(`
SELECT userid
  FROM ` + config.DB.TablePrefix + `user
 ORDER BY userid ASC`)
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

	fmt.Println("Exporting users")
	RunDBTasks(ids, ExportUser)
}

func ExportUser(id int64) error {

	// Split the filename and ensure the directory exists
	path, name := SplitFilename(strconv.FormatInt(id, 10))
	path = config.Export.OutputDirectory + userDir + path

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
		&vb.UserId,
		&vb.UserGroupId,
		&vb.MemberGroupIds,
		&vb.Username,
		&vb.Email,
		&vb.JoinDate,
		&vb.LastVisit,
		&vb.LastActivity,
		&vb.IpAddress,
		&vb.Banned,
		&vb.Options,
	)
	if err != nil {
		return err
	}

	// Map the user into our structure performing any translations needed

	ex := f.User{}
	ex.ID = vb.UserId
	ex.Name = vb.Username
	ex.Email = vb.Email
	ex.DateCreated = time.Unix(vb.JoinDate, 0).UTC()
	ex.LastActive = time.Unix(vb.LastVisit, 0).UTC()
	ex.IPAddress = vb.IpAddress
	ex.Banned = vb.Banned

	usergroups := []f.ID{}
	usergroups = append(usergroups, f.ID{ID: vb.UserGroupId})
	if vb.MemberGroupIds != "" {
		groups := strings.Split(vb.MemberGroupIds, ",")
		for _, group := range groups {
			groupId, err := strconv.ParseInt(strings.Trim(group, " "), 10, 64)
			if err != nil {
				return err
			}
			usergroups = append(usergroups, f.ID{ID: groupId})
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

	// Write the user
	err = WriteFile(filename, ex)
	if err != nil {
		return err
	}

	return nil
}
