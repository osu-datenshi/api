package mitsuha

import (
	"database/sql"
	"strconv"
	"github.com/osuthailand/api/common"
)

type userData1 struct {
	ID             int                  `json:"id"`
	Username       string               `json:"username"`
	UsernameAKA    string               `json:"username_aka"`
	Privileges     uint64               `json:"privileges"`
	RegisteredOn   common.UnixTimestamp `json:"registered_on"`
	LatestActivity common.UnixTimestamp `json:"latest_activity"`
	Country        string               `json:"country"`
}

type friendData struct {
	userData1
	IsSubbed bool `json:"is_subbed"`
}

type friendsGETResponse struct {
	common.ResponseBase
	Friends []friendData `json:"subs"`
}

type subsCountGetResponse struct {
	common.ResponseBase
	SubsCount string `json:"subscount"`
}

func FollowersGET(md common.MethodData) common.CodeMessager {
	var HasDonor bool
	HasDonor = md.User.UserPrivileges&common.UserPrivilegeDonor > 0
	if !HasDonor {
		return common.SimpleResponse(400, "non-donor")
	}

	var myFrienders []int
	myFriendersRaw, err := md.DB.Query("SELECT user1 FROM users_relationships WHERE user2 = ?", md.ID())
	if err != nil {
		md.Err(err)
		return common.SimpleResponse(500, "An error occurred. Trying again may work. If it doesn't, yell at this Mitsuha instance admin and tell them to fix the API.")
	}
	defer myFriendersRaw.Close()
	for myFriendersRaw.Next() {
		var i int
		err := myFriendersRaw.Scan(&i)
		if err != nil {
			md.Err(err)
			continue
		}
		myFrienders = append(myFrienders, i)
	}
	if err := myFriendersRaw.Err(); err != nil {
		md.Err(err)
	}

	myFriendsQuery := `
SELECT             
	users.id, users.username, users.register_datetime, users.privileges, users.latest_activity,
	users_stats.username_aka,
	users_stats.country
FROM users_relationships
LEFT JOIN users
ON users_relationships.user1 = users.id
LEFT JOIN users_stats
ON users_relationships.user1=users_stats.id
WHERE users_relationships.user2=? AND NOT EXISTS (SELECT * FROM users_relationships WHERE users_relationships.user1=? AND users_relationships.user2=users.id)
`

	myFriendsQuery += common.Sort(md, common.SortConfiguration{
		Allowed: []string{
			"id",
			"username",
			"latest_activity",
		},
		Default: "users.id asc",
		Table:   "users",
	}) + "\n"

	results, err := md.DB.Query(myFriendsQuery+common.Paginate(md.Query("p"), md.Query("l"), 100), md.ID(), md.ID())
	if err != nil {
		md.Err(err)
		return common.SimpleResponse(500, "An error occurred. Trying again may work. If it doesn't, yell at this Mitsuha instance admin and tell them to fix the API.")
	}

	var myFriends []friendData

	defer results.Close()
	for results.Next() {
		newFriend := friendPuts(md, results)
		for range myFrienders {
			newFriend.IsSubbed = true
			break
		}
		myFriends = append(myFriends, newFriend)
	}
	if err := results.Err(); err != nil {
		md.Err(err)
	}

	r := friendsGETResponse{}
	r.Code = 200
	r.Friends = myFriends
	return r
}

func FollowersGetResponse(md common.MethodData) common.CodeMessager {
	userid, err := strconv.Atoi(md.Query("userid"))
	if err != nil {
		return common.SimpleResponse(500, "An error occurred. Trying again may work. If it doesn't, yell at this Mitsuha instance admin and tell them to fix the API.")
	}

	myFriendsQuery := `
SELECT             
	users.id, users.username, users.register_datetime, users.privileges, users.latest_activity,
	users_stats.username_aka,
	users_stats.country
FROM users_relationships
LEFT JOIN users
ON users_relationships.user1 = users.id
LEFT JOIN users_stats
ON users_relationships.user1=users_stats.id
WHERE users_relationships.user2=? AND NOT EXISTS (SELECT * FROM users_relationships WHERE users_relationships.user1=? AND users_relationships.user2=users.id)`

	r := subsCountGetResponse{}
	results, err := md.DB.Query(myFriendsQuery, userid, userid)
	if err != nil {
		md.Err(err)
		return common.SimpleResponse(500, "An error occurred. Trying again may work. If it doesn't, yell at this Mitsuha instance admin and tell them to fix the API.")
	}

	var count int = 0
	defer results.Close()
	for results.Next() {
		count+=1
	}

	if err := results.Err(); err != nil {
		md.Err(err)
	}
	r.SubsCount = strconv.Itoa(count)
	r.Code = 200
	return r
}

func friendPuts(md common.MethodData, row *sql.Rows) (user friendData) {
	var err error

	err = row.Scan(&user.ID, &user.Username, &user.RegisteredOn, &user.Privileges, &user.LatestActivity, &user.UsernameAKA, &user.Country)
	if err != nil {
		md.Err(err)
		return
	}

	return
}
