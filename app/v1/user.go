// Package v1 implements the first version of the Ripple API.
package v1

import (
	"database/sql"
	"strconv"
	"strings"
	"unicode"

	"github.com/jmoiron/sqlx"
	"github.com/osu-datenshi/lib/ocl"
	"github.com/osu-datenshi/api/common"
)

type userData struct {
	ID             int                  `json:"id"`
	Username       string               `json:"username"`
	UsernameAKA    string               `json:"username_aka"`
	RegisteredOn   common.UnixTimestamp `json:"registered_on"`
	Privileges     int64               `json:"privileges"`
	LatestActivity common.UnixTimestamp `json:"latest_activity"`
	Country        string               `json:"country"`
}

const userFields = `SELECT users.id, users.username, register_datetime, users.privileges,
	latest_activity, us.username_aka,
	us.country
FROM users
INNER JOIN users_stats as us ON users.id = us.id
`

// UsersGET is the API handler for GET /users
func UsersGET(md common.MethodData) common.CodeMessager {
	shouldRet, whereClause, param := whereClauseUser(md, "users")
	if shouldRet != nil {
		return userPutsMulti(md)
	}

	query := userFields + `
WHERE ` + whereClause + ` AND ` + md.User.OnlyUserPublic(true) + `
LIMIT 1`
	return userPutsSingle(md, md.DB.QueryRowx(query, param))
}

type userPutsSingleUserData struct {
	common.ResponseBase
	userData
}

func userPutsSingle(md common.MethodData, row *sqlx.Row) common.CodeMessager {
	var err error
	var user userPutsSingleUserData

	err = row.StructScan(&user.userData)
	switch {
	case err == sql.ErrNoRows:
		return common.SimpleResponse(404, "No such user was found!")
	case err != nil:
		md.Err(err)
		return Err500
	}

	user.Code = 200
	return user
}

type userPutsMultiUserData struct {
	common.ResponseBase
	Users []userData `json:"users"`
}

func userPutsMulti(md common.MethodData) common.CodeMessager {
	pm := md.Ctx.Request.URI().QueryArgs().PeekMulti
	// query composition
	wh := common.
		Where("users.username_safe = ?", common.SafeUsername(md.Query("nname"))).
		Where("users.id = ?", md.Query("iid")).
		Where("users.privileges = ?", md.Query("privileges")).
		Where("users.privileges & ? > 0", md.Query("has_privileges")).
		Where("users.privileges & ? = 0", md.Query("has_not_privileges")).
		Where("us.country = ?", md.Query("country")).
		Where("us.username_aka = ?", md.Query("name_aka")).
		Where("pg.name = ?", md.Query("privilege_group")).
		In("users.id", pm("ids")...).
		In("users.username_safe", safeUsernameBulk(pm("names"))...).
		In("us.username_aka", pm("names_aka")...).
		In("us.country", pm("countries")...)

	var extraJoin string
	if md.Query("privilege_group") != "" {
		extraJoin = " LEFT JOIN privileges_groups as pg ON users.privileges & pg.privileges = pg.privileges "
	}

	query := userFields + extraJoin + wh.ClauseSafe() + " AND " + md.User.OnlyUserPublic(true) +
		" " + common.Sort(md, common.SortConfiguration{
		Allowed: []string{
			"id",
			"username",
			"privileges",
			"donor_expire",
			"latest_activity",
			"silence_end",
		},
		Default: "id ASC",
		Table:   "users",
	}) +
		" " + common.Paginate(md.Query("p"), md.Query("l"), 100)

	// query execution
	rows, err := md.DB.Queryx(query, wh.Params...)
	if err != nil {
		md.Err(err)
		return Err500
	}
	var r userPutsMultiUserData
	for rows.Next() {
		var u userData
		err := rows.StructScan(&u)
		if err != nil {
			md.Err(err)
			continue
		}
		r.Users = append(r.Users, u)
	}
	r.Code = 200
	return r
}

// UserSelfGET is a shortcut for /users/id/self. (/users/self)
func UserSelfGET(md common.MethodData) common.CodeMessager {
	md.Ctx.Request.URI().SetQueryString("id=self")
	return UsersGET(md)
}

func safeUsernameBulk(us [][]byte) [][]byte {
	for _, u := range us {
		for idx, v := range u {
			if v == ' ' {
				u[idx] = '_'
				continue
			}
			u[idx] = byte(unicode.ToLower(rune(v)))
		}
	}
	return us
}

type whatIDResponse struct {
	common.ResponseBase
	ID int `json:"id"`
}

// UserWhatsTheIDGET is an API request that only returns an user's ID.
func UserWhatsTheIDGET(md common.MethodData) common.CodeMessager {
	var (
		r          whatIDResponse
		privileges int64
	)
	err := md.DB.QueryRow("SELECT id, privileges FROM users WHERE username_safe = ? LIMIT 1", common.SafeUsername(md.Query("name"))).Scan(&r.ID, &privileges)
	if err != nil || ((privileges&int64(common.UserPrivilegePublic)) == 0 &&
		(md.User.UserPrivileges&common.AdminPrivilegeManageUsers == 0)) {
		return common.SimpleResponse(404, "That user could not be found!")
	}
	r.Code = 200
	return r
}

var modesToReadable = [...]string{
	"std",
	"taiko",
	"ctb",
	"mania",
}

type modeData struct {
	RankedScore            int64  `json:"ranked_score"`
	TotalScore             int64  `json:"total_score"`
	PlayCount              int     `json:"playcount"`
	PlayTime               int     `json:"play_time"`
	ReplaysWatched         int     `json:"replays_watched"`
	TotalHits              int     `json:"total_hits"`
	Level                  float64 `json:"level"`
	Accuracy               float64 `json:"accuracy"`
	PP                     int     `json:"pp"`
	GlobalLeaderboardRank  *int    `json:"global_leaderboard_rank"`
	CountryLeaderboardRank *int    `json:"country_leaderboard_rank"`
}
type userFullResponse struct {
	common.ResponseBase
	userData
	STD           modeData              `json:"std"`
	Taiko         modeData              `json:"taiko"`
	CTB           modeData              `json:"ctb"`
	Mania         modeData              `json:"mania"`
	PlayStyle     int                   `json:"play_style"`
	FavouriteMode int                   `json:"favourite_mode"`
	Badges        []singleBadge         `json:"badges"`
	Clan          singleClan            `json:"clan"`
	CustomBadge   *singleBadge          `json:"custom_badge"`
	SilenceInfo   silenceInfo           `json:"silence_info"`
	CMNotes       *string               `json:"cm_notes,omitempty"`
	BanDate       *common.UnixTimestamp `json:"ban_date,omitempty"`
	Email         string                `json:"email,omitempty"`
}
type silenceInfo struct {
	Reason string               `json:"reason"`
	End    common.UnixTimestamp `json:"end"`
}
type userNotFullResponse struct {
	Id             int                  `json:"id"`
	Username       string               `json:"username"`
	UsernameAKA    string               `json:"username_aka"`
	RegisteredOn   common.UnixTimestamp `json:"registered_on"`
	Privileges     int64               `json:"privileges"`
	LatestActivity common.UnixTimestamp `json:"latest_activity"`
	Country        string               `json:"country"`
	UserColor        string               `json:"user_color"`
	RankedScoreStd            int64  `json:"ranked_score_std"`
	TotalScoreStd             int64  `json:"total_score_std"`
	PlaycountStd              int     `json:"playcount_std"`
	ReplaysWatchedStd         int     `json:"replays_watched_std"`
	TotalHitsStd              int     `json:"total_hits_std"`
	PpStd                     int     `json:"pp_std"`
	RankedScoreTaiko            int64  `json:"ranked_score_taiko"`
	TotalScoreTaiko             int64  `json:"total_score_taiko"`
	PlaycountTaiko              int     `json:"playcount_taiko"`
	ReplaysWatchedTaiko         int     `json:"replays_watched_taiko"`
	TotalHitsTaiko              int     `json:"total_hits_taiko"`
	PpTaiko                     int     `json:"pp_taiko"`
	RankedScoreCtb            int64  `json:"ranked_score_ctb"`
	TotalScoreCtb            int64  `json:"total_score_ctb"`
	PlaycountCtb              int     `json:"playcount_ctb"`
	ReplaysWatchedCtb         int     `json:"replays_watched_ctb"`
	TotalHitsCtb              int     `json:"total_hits_ctb"`
	PpCtb                     int     `json:"pp_ctb"`
	RankedScoreMania            int64  `json:"ranked_score_mania"`
	TotalScoreMania             int64  `json:"total_score_mania"`
	PlaycountMania              int     `json:"playcount_mania"`
	ReplaysWatchedMania         int     `json:"replays_watched_mania"`
	TotalHitsMania              int     `json:"total_hits_mania"`
	PpMania                     int     `json:"pp_mania"`
	// STD       clappedModeData  `json:"std"`
	// Taiko     clappedModeData  `json:"taiko"`
	// CTB       clappedModeData  `json:"ctb"`
	// Mania     clappedModeData  `json:"mania"`
}



// RelaxUserFullGET gets all of... bluh.. I'm tired...
func RelaxUserFullGET(md common.MethodData) common.CodeMessager {
	shouldRet, whereClause, param := whereClauseUser(md, "users")
	if shouldRet != nil {
		return *shouldRet
	}

	// Hellest query I've ever done.
	query := `
SELECT
	users.id, users.username, users.register_datetime, users.privileges, users.latest_activity,

	us.username_aka, us.country, us.play_style, us.favourite_mode,

	us.custom_badge_icon, us.custom_badge_name, us.can_custom_badge,
	us.show_custom_badge,

	rs.ranked_score_std, rs.total_score_std, rs.playcount_std,
	us.replays_watched_std, us.total_hits_std,
	rs.avg_accuracy_std, rs.pp_std, rs.playtime_std,

	rs.ranked_score_taiko, rs.total_score_taiko, rs.playcount_taiko,
	us.replays_watched_taiko, us.total_hits_taiko,
	rs.avg_accuracy_taiko, rs.pp_taiko, rs.playtime_taiko,

	rs.ranked_score_ctb, rs.total_score_ctb, rs.playcount_ctb,
	us.replays_watched_ctb, us.total_hits_ctb,
	rs.avg_accuracy_ctb, rs.pp_ctb, rs.playtime_ctb,

	rs.ranked_score_mania, rs.total_score_mania, rs.playcount_mania,
	us.replays_watched_mania, us.total_hits_mania,
	rs.avg_accuracy_mania, rs.pp_mania, rs.playtime_mania,

	users.silence_reason, users.silence_end,
	users.notes, users.ban_datetime, users.email

FROM users
LEFT JOIN users_stats as us ON users.id = us.id
LEFT JOIN rx_stats as rs ON users.id = rs.id
WHERE ` + whereClause + ` AND ` + md.User.OnlyUserPublic(true) + `
LIMIT 1
`
	// Whatever man.
	r := userFullResponse{}
	var (
		b    singleBadge
		can  bool
		show bool
	)
	err := md.DB.QueryRow(query, param).Scan(
		&r.ID, &r.Username, &r.RegisteredOn, &r.Privileges, &r.LatestActivity,

		&r.UsernameAKA, &r.Country,
		&r.PlayStyle, &r.FavouriteMode,

		&b.Icon, &b.Name, &can, &show,

		&r.STD.RankedScore, &r.STD.TotalScore, &r.STD.PlayCount,
		&r.STD.ReplaysWatched, &r.STD.TotalHits,
		&r.STD.Accuracy, &r.STD.PP, &r.STD.PlayTime,

		&r.Taiko.RankedScore, &r.Taiko.TotalScore, &r.Taiko.PlayCount,
		&r.Taiko.ReplaysWatched, &r.Taiko.TotalHits,
		&r.Taiko.Accuracy, &r.Taiko.PP, &r.Taiko.PlayTime,

		&r.CTB.RankedScore, &r.CTB.TotalScore, &r.CTB.PlayCount,
		&r.CTB.ReplaysWatched, &r.CTB.TotalHits,
		&r.CTB.Accuracy, &r.CTB.PP, &r.CTB.PlayTime,

		&r.Mania.RankedScore, &r.Mania.TotalScore, &r.Mania.PlayCount,
		&r.Mania.ReplaysWatched, &r.Mania.TotalHits,
		&r.Mania.Accuracy, &r.Mania.PP, &r.Mania.PlayTime,

		&r.SilenceInfo.Reason, &r.SilenceInfo.End,
		&r.CMNotes, &r.BanDate, &r.Email,
	)
	switch {
	case err == sql.ErrNoRows:
		return common.SimpleResponse(404, "That user could not be found!")
	case err != nil:
		md.Err(err)
		return Err500
	}

	can = can && show && common.UserPrivileges(r.Privileges)&common.UserPrivilegeDonor > 0
	if can && (b.Name != "" || b.Icon != "") {
		r.CustomBadge = &b
	}

	for modeID, m := range [...]*modeData{&r.STD, &r.Taiko, &r.CTB, &r.Mania} {
		m.Level = ocl.GetLevelPrecise(int64(m.TotalScore))

		if i := relaxboardPosition(md.R, modesToReadable[modeID], r.ID); i != nil {
			m.GlobalLeaderboardRank = i
		}
		if i := rxcountryPosition(md.R, modesToReadable[modeID], r.ID, r.Country); i != nil {
			m.CountryLeaderboardRank = i
		}
	}

	rows, err := md.DB.Query("SELECT b.id, b.name, b.icon FROM user_badges ub "+
		"LEFT JOIN badges b ON ub.badge = b.id WHERE user = ?", r.ID)
	if err != nil {
		md.Err(err)
	}

	for rows.Next() {
		var badge singleBadge
		err := rows.Scan(&badge.ID, &badge.Name, &badge.Icon)
		if err != nil {
			md.Err(err)
			continue
		}
		r.Badges = append(r.Badges, badge)
	}

	if md.User.TokenPrivileges&common.PrivilegeManageUser == 0 {
		r.CMNotes = nil
		r.BanDate = nil
		r.Email = ""
	}

	rows, err = md.DB.Query("SELECT c.id, c.name, c.description, c.tag, c.icon FROM user_clans uc "+
		"LEFT JOIN clans c ON uc.clan = c.id WHERE user = ?", r.ID)
	if err != nil {
		md.Err(err)
	}

	for rows.Next() {
		var clan singleClan
		err = rows.Scan(&clan.ID, &clan.Name, &clan.Description, &clan.Tag, &clan.Icon)
		if err != nil {
			md.Err(err)
			continue
		}
		r.Clan = clan
	}

	r.Code = 200
	return r
}

// UserFullGET gets all of an user's information, with one exception: their userpage.
func UserFullGET(md common.MethodData) common.CodeMessager {
	shouldRet, whereClause, param := whereClauseUser(md, "users")
	if shouldRet != nil {
		return *shouldRet
	}

	// Hellest query I've ever done.
	query := `
SELECT
	users.id, users.username, users.register_datetime, users.privileges, users.latest_activity,

	us.username_aka, us.country, us.play_style, us.favourite_mode,

	us.custom_badge_icon, us.custom_badge_name, us.can_custom_badge,
	us.show_custom_badge,

	us.ranked_score_std, us.total_score_std, us.playcount_std,
	us.replays_watched_std, us.total_hits_std,
	us.avg_accuracy_std, us.pp_std, us.playtime_std,

	us.ranked_score_taiko, us.total_score_taiko, us.playcount_taiko,
	us.replays_watched_taiko, us.total_hits_taiko,
	us.avg_accuracy_taiko, us.pp_taiko, us.playtime_taiko,

	us.ranked_score_ctb, us.total_score_ctb, us.playcount_ctb,
	us.replays_watched_ctb, us.total_hits_ctb,
	us.avg_accuracy_ctb, us.pp_ctb, us.playtime_ctb,

	us.ranked_score_mania, us.total_score_mania, us.playcount_mania,
	us.replays_watched_mania, us.total_hits_mania,
	us.avg_accuracy_mania, us.pp_mania, us.playtime_mania,

	users.silence_reason, users.silence_end,
	users.notes, users.ban_datetime, users.email

FROM users
LEFT JOIN users_stats as us ON users.id = us.id
WHERE ` + whereClause + ` AND ` + md.User.OnlyUserPublic(true) + `
LIMIT 1
`
	// Fuck.
	r := userFullResponse{}
	var (
		b    singleBadge
		can  bool
		show bool
	)
	err := md.DB.QueryRow(query, param).Scan(
		&r.ID, &r.Username, &r.RegisteredOn, &r.Privileges, &r.LatestActivity,

		&r.UsernameAKA, &r.Country,
		&r.PlayStyle, &r.FavouriteMode,

		&b.Icon, &b.Name, &can, &show,

		&r.STD.RankedScore, &r.STD.TotalScore, &r.STD.PlayCount,
		&r.STD.ReplaysWatched, &r.STD.TotalHits,
		&r.STD.Accuracy, &r.STD.PP, &r.STD.PlayTime,

		&r.Taiko.RankedScore, &r.Taiko.TotalScore, &r.Taiko.PlayCount,
		&r.Taiko.ReplaysWatched, &r.Taiko.TotalHits,
		&r.Taiko.Accuracy, &r.Taiko.PP, &r.Taiko.PlayTime,

		&r.CTB.RankedScore, &r.CTB.TotalScore, &r.CTB.PlayCount,
		&r.CTB.ReplaysWatched, &r.CTB.TotalHits,
		&r.CTB.Accuracy, &r.CTB.PP, &r.CTB.PlayTime,

		&r.Mania.RankedScore, &r.Mania.TotalScore, &r.Mania.PlayCount,
		&r.Mania.ReplaysWatched, &r.Mania.TotalHits,
		&r.Mania.Accuracy, &r.Mania.PP, &r.Mania.PlayTime,

		&r.SilenceInfo.Reason, &r.SilenceInfo.End,
		&r.CMNotes, &r.BanDate, &r.Email,
	)
	switch {
	case err == sql.ErrNoRows:
		return common.SimpleResponse(404, "That user could not be found!")
	case err != nil:
		md.Err(err)
		return Err500
	}

	can = can && show && common.UserPrivileges(r.Privileges)&common.UserPrivilegeDonor > 0
	if can && (b.Name != "" || b.Icon != "") {
		r.CustomBadge = &b
	}

	for modeID, m := range [...]*modeData{&r.STD, &r.Taiko, &r.CTB, &r.Mania} {
		m.Level = ocl.GetLevelPrecise(int64(m.TotalScore))

		if i := leaderboardPosition(md.R, modesToReadable[modeID], r.ID); i != nil {
			m.GlobalLeaderboardRank = i
		}
		if i := countryPosition(md.R, modesToReadable[modeID], r.ID, r.Country); i != nil {
			m.CountryLeaderboardRank = i
		}
	}

	rows, err := md.DB.Query("SELECT b.id, b.name, b.icon FROM user_badges ub "+
		"LEFT JOIN badges b ON ub.badge = b.id WHERE user = ?", r.ID)
	if err != nil {
		md.Err(err)
	}

	for rows.Next() {
		var badge singleBadge
		err := rows.Scan(&badge.ID, &badge.Name, &badge.Icon)
		if err != nil {
			md.Err(err)
			continue
		}
		r.Badges = append(r.Badges, badge)
	}

	if md.User.TokenPrivileges&common.PrivilegeManageUser == 0 {
		r.CMNotes = nil
		r.BanDate = nil
		r.Email = ""
	}

	rows, err = md.DB.Query("SELECT c.id, c.name, c.description, c.tag, c.icon FROM user_clans uc "+
		"LEFT JOIN clans c ON uc.clan = c.id WHERE user = ?", r.ID)
	if err != nil {
		md.Err(err)
	}

	for rows.Next() {
		var clan singleClan
		err = rows.Scan(&clan.ID, &clan.Name, &clan.Description, &clan.Tag, &clan.Icon)
		if err != nil {
			md.Err(err)
			continue
		}
		r.Clan = clan
	}

	r.Code = 200
	return r
}

type userpageResponse struct {
	common.ResponseBase
	Userpage *string `json:"userpage"`
}

// UserUserpageGET gets an user's userpage, as in the customisable thing.
func UserUserpageGET(md common.MethodData) common.CodeMessager {
	shouldRet, whereClause, param := whereClauseUser(md, "us")
	if shouldRet != nil {
		return *shouldRet
	}
	var r userpageResponse
	err := md.DB.QueryRow("SELECT userpage_content FROM users_stats as us WHERE "+whereClause+" LIMIT 1", param).Scan(&r.Userpage)
	switch {
	case err == sql.ErrNoRows:
		return common.SimpleResponse(404, "No such user!")
	case err != nil:
		md.Err(err)
		return Err500
	}
	if r.Userpage == nil {
		r.Userpage = new(string)
	}
	r.Code = 200
	return r
}

// UserSelfUserpagePOST allows to change the current user's userpage.
func UserSelfUserpagePOST(md common.MethodData) common.CodeMessager {
	var d struct {
		Data *string `json:"data"`
	}
	md.Unmarshal(&d)
	if d.Data == nil {
		return ErrMissingField("data")
	}
	cont := common.SanitiseString(*d.Data)
	_, err := md.DB.Exec("UPDATE users_stats SET userpage_content = ? WHERE id = ? LIMIT 1", cont, md.ID())
	if err != nil {
		md.Err(err)
	}
	md.Ctx.URI().SetQueryString("id=self")
	return UserUserpageGET(md)
}

func whereClauseUser(md common.MethodData, tableName string) (*common.CodeMessager, string, interface{}) {
	switch {
	case md.Query("id") == "self":
		return nil, tableName + ".id = ?", md.ID()
	case md.Query("id") != "":
		id, err := strconv.Atoi(md.Query("id"))
		if err != nil {
			a := common.SimpleResponse(400, "please pass a valid user ID")
			return &a, "", nil
		}
		return nil, tableName + ".id = ?", id
	case md.Query("name") != "":
		return nil, tableName + ".username_safe = ?", common.SafeUsername(md.Query("name"))
	}
	a := common.SimpleResponse(400, "you need to pass either querystring parameters name or id")
	return &a, "", nil
}

type userLookupResponse struct {
	common.ResponseBase
	Users []lookupUser `json:"users"`
}
type lookupUser struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
}

// UserLookupGET does a quick lookup of users beginning with the passed
// querystring value name.
func UserLookupGET(md common.MethodData) common.CodeMessager {
	name := common.SafeUsername(md.Query("name"))
	name = strings.NewReplacer(
		"%", "\\%",
		"_", "\\_",
		"\\", "\\\\",
	).Replace(name)
	if name == "" {
		return common.SimpleResponse(400, "please provide an username to start searching")
	}
	name = "%" + name + "%"

	var email string
	if md.User.TokenPrivileges&common.PrivilegeManageUser != 0 &&
		strings.Contains(md.Query("name"), "@") {
		email = md.Query("name")
	}

	rows, err := md.DB.Query("SELECT users.id, users.username FROM users WHERE "+
		"(username_safe LIKE ? OR email = ?) AND "+
		md.User.OnlyUserPublic(true)+" LIMIT 25", name, email)
	if err != nil {
		md.Err(err)
		return Err500
	}

	var r userLookupResponse
	for rows.Next() {
		var l lookupUser
		err := rows.Scan(&l.ID, &l.Username)
		if err != nil {
			continue // can't be bothered to handle properly
		}
		r.Users = append(r.Users, l)
	}

	r.Code = 200
	return r
}
