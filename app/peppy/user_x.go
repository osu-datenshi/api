package peppy

import (
	"fmt"
	"strings"

	"github.com/jmoiron/sqlx"
	"github.com/valyala/fasthttp"
	"gopkg.in/thehowl/go-osuapi.v1"
	"github.com/osu-datenshi/api/common"
	"github.com/osu-datenshi/getrank"
)

// GetUserRecent retrieves an user's recent scores.
func GetUserRecent(c *fasthttp.RequestCtx, db *sqlx.DB) {
	getUserX(c, db, "ORDER BY s.time DESC", common.InString(1, query(c, "limit"), 50, 10))
}

// GetUserBest retrieves an user's best scores.
func GetUserBest(c *fasthttp.RequestCtx, db *sqlx.DB) {
	var sb string
	if rankable(query(c, "m")) {
		sb = "s.pp"
	} else {
		sb = "s.score"
	}
	getUserX(c, db, "AND completed = '3' ORDER BY "+sb+" DESC", common.InString(1, query(c, "limit"), 100, 10))
}

func getUserX(c *fasthttp.RequestCtx, db *sqlx.DB, orderBy string, limit int) {
	whereClause, p := genUser(c, db)
	sqlQuery := fmt.Sprintf(
		`SELECT
			b.beatmap_id, s.score, s.max_combo,
			s.300_count, s.100_count, s.50_count,
			s.gekis_count, s.katus_count, s.misses_count,
			s.full_combo, s.mods, users.id, s.time,
			s.pp, s.accuracy
		FROM scores_master as s
		LEFT JOIN b ON using(beatmap_md5)
		LEFT JOIN u as u ON s.userid = u.id
		WHERE %s AND s.play_mode = ? AND u.privileges & 1 > 0
		%s
		LIMIT %d`, whereClause, orderBy, limit,
	)
	scores := make([]osuapi.GUSScore, 0, limit)
	m := genmodei(query(c, "m"))
	rows, err := db.Query(sqlQuery, p, m)
	if err != nil {
		json(c, 200, defaultResponse)
		common.Err(c, err)
		return
	}
	for rows.Next() {
		var (
			curscore osuapi.GUSScore
			rawTime  common.UnixTimestamp
			acc      float64
			fc       bool
			mods     int
			bid      *int
		)
		err := rows.Scan(
			&bid, &curscore.Score.Score, &curscore.MaxCombo,
			&curscore.Count300, &curscore.Count100, &curscore.Count50,
			&curscore.CountGeki, &curscore.CountKatu, &curscore.CountMiss,
			&fc, &mods, &curscore.UserID, &rawTime,
			&curscore.PP, &acc,
		)
		if err != nil {
			json(c, 200, defaultResponse)
			common.Err(c, err)
			return
		}
		if bid == nil {
			curscore.BeatmapID = 0
		} else {
			curscore.BeatmapID = *bid
		}
		curscore.FullCombo = osuapi.OsuBool(fc)
		curscore.Mods = osuapi.Mods(mods)
		curscore.Date = osuapi.MySQLDate(rawTime)
		curscore.Rank = strings.ToUpper(getrank.GetRank(
			osuapi.Mode(m),
			curscore.Mods,
			acc,
			curscore.Count300,
			curscore.Count100,
			curscore.Count50,
			curscore.CountMiss,
		))
		scores = append(scores, curscore)
	}
	json(c, 200, scores)
}
