package v1

import (
	"fmt"
	"strings"

	"gopkg.in/thehowl/go-osuapi.v1"
	"database/sql"
	"github.com/osu-datenshi/api/common"
	"github.com/osu-datenshi/getrank"
)

type userScore struct {
	Score
	Beatmap beatmap `json:"beatmap"`
}

type userScoresResponse struct {
	common.ResponseBase
	Scores []userScore `json:"scores"`
}

const masterMode = true
const masterScoreSelectBase = `
		SELECT
			s.id, s.beatmap_md5, s.score,
			s.max_combo, s.full_combo, s.mods,
			s.300_count, s.100_count, s.50_count,
			s.gekis_count, s.katus_count, s.misses_count,
			s.time, s.play_mode, s.accuracy, s.pp,
			s.completed,

			b.beatmap_id, b.beatmapset_id, b.beatmap_md5,
			b.song_name, b.ar, b.od, b.difficulty_std,
			b.difficulty_taiko, b.difficulty_ctb, b.difficulty_mania,
			b.max_combo, b.hit_length, b.ranked,
			b.ranked_status_freezed, b.latest_update
		FROM scores_master as s
		INNER JOIN beatmaps as b ON b.beatmap_md5 = s.beatmap_md5
		INNER JOIN users ON users.id = s.userid
		`

const relaxScoreSelectBase = `
		SELECT
			s.id, s.beatmap_md5, s.score,
			s.max_combo, s.full_combo, s.mods,
			s.300_count, s.100_count, s.50_count,
			s.gekis_count, s.katus_count, s.misses_count,
			s.time, s.play_mode, s.accuracy, s.pp,
			s.completed,

			b.beatmap_id, b.beatmapset_id, b.beatmap_md5,
			b.song_name, b.ar, b.od, b.difficulty_std,
			b.difficulty_taiko, b.difficulty_ctb, b.difficulty_mania,
			b.max_combo, b.hit_length, b.ranked,
			b.ranked_status_freezed, b.latest_update
		FROM scores_relax as s
		INNER JOIN beatmaps as b ON b.beatmap_md5 = s.beatmap_md5
		INNER JOIN users ON users.id = s.userid
		`

const userScoreSelectBase = `
		SELECT
			s.id, s.beatmap_md5, s.score,
			s.max_combo, s.full_combo, s.mods,
			s.300_count, s.100_count, s.50_count,
			s.gekis_count, s.katus_count, s.misses_count,
			s.time, s.play_mode, s.accuracy, s.pp,
			s.completed,

			b.beatmap_id, b.beatmapset_id, b.beatmap_md5,
			b.song_name, b.ar, b.od, b.difficulty_std,
			b.difficulty_taiko, b.difficulty_ctb, b.difficulty_mania,
			b.max_combo, b.hit_length, b.ranked,
			b.ranked_status_freezed, b.latest_update
		FROM scores as s
		INNER JOIN beatmaps ON b.beatmap_md5 = s.beatmap_md5
		INNER JOIN users ON users.id = s.userid
		`

// UserScoresBestGET retrieves the best scores of an user, sorted by PP if
// mode is standard and sorted by ranked score otherwise.
func UserScoresBestGET(md common.MethodData) common.CodeMessager {
	cm, wc, param := whereClauseUser(md, "users")
	if cm != nil {
		return *cm
	}
	
	mc := genModeClause(md)
	smode := common.Int(md.Query("smode"))
	if smode < 0 || smode > 2 {
		// enforce to vanilla
		smode = 0
	}
	if !md.HasQuery("smode") && common.Int(md.Query("rx")) > 0 {
		// retain old behavior
		smode = 1
	}
	// For all modes that have PP, we leave out 0 PP scores.

	if masterMode {
		return masterPuts(md, fmt.Sprintf(
			`WHERE
				s.completed = '3'
				AND %s
				%s
				AND s.special_mode = %d
				AND %s
			ORDER BY s.pp DESC, s.score DESC %s`,
			wc, mc, smode, md.User.OnlyUserPublic(true), common.Paginate(md.Query("p"), md.Query("l"), 100),
		), param)
	} else {
		switch smode {
			case 1:
				return relaxPuts(md, fmt.Sprintf(
					`WHERE
						s.completed = '3'
						AND %s
						%s
						AND %s
					ORDER BY s.pp DESC, s.score DESC %s`,
					wc, mc, md.User.OnlyUserPublic(true), common.Paginate(md.Query("p"), md.Query("l"), 100),
				), param)
			default:
				return scoresPuts(md, fmt.Sprintf(
					`WHERE
						s.completed = '3'
						AND %s
						%s
						AND %s
					ORDER BY s.pp DESC, s.score DESC %s`,
					wc, mc, md.User.OnlyUserPublic(true), common.Paginate(md.Query("p"), md.Query("l"), 100),
				), param)
		}
	}
}

// UserScoresRecentGET retrieves an user's latest scores.
func UserScoresRecentGET(md common.MethodData) common.CodeMessager {
	cm, wc, param := whereClauseUser(md, "users")
	if cm != nil {
		return *cm
	}
	mc := genModeClause(md)
	smode := common.Int(md.Query("smode"))
	if smode < 0 || smode > 2 {
		// enforce to vanilla
		smode = 0
	}
	if !md.HasQuery("smode") && common.Int(md.Query("rx")) > 0 {
		// retain old behavior
		smode = 1
	}
	if masterMode {
		return masterPuts(md, fmt.Sprintf(
			`WHERE
				%s
				%s
				AND s.special_mode = %d
				AND %s
			ORDER BY s.id DESC %s`,
			wc, mc, smode, md.User.OnlyUserPublic(true), common.Paginate(md.Query("p"), md.Query("l"), 100),
		), param)
	} else {
		switch smode {
			case 1:
				return relaxPuts(md, fmt.Sprintf(
					`WHERE
						%s
						%s
						AND `+md.User.OnlyUserPublic(true)+`
					ORDER BY s.id DESC %s`,
					wc, mc, common.Paginate(md.Query("p"), md.Query("l"), 100),
				), param)
		  default:
				return scoresPuts(md, fmt.Sprintf(
					`WHERE
						%s
						%s
						AND `+md.User.OnlyUserPublic(true)+`
					ORDER BY s.id DESC %s`,
					wc, mc, common.Paginate(md.Query("p"), md.Query("l"), 100),
				), param)
		}
	}
}

func genericPuts(rows sql.Rows, md common.MethodData) common.CodeMessager {
	err := nil
	var scores []userScore
	for rows.Next() {
		var (
			us userScore
			b  beatmap
		)
		err = rows.Scan(
			&us.ID, &us.BeatmapMD5, &us.Score.Score,
			&us.MaxCombo, &us.FullCombo, &us.Mods,
			&us.Count300, &us.Count100, &us.Count50,
			&us.CountGeki, &us.CountKatu, &us.CountMiss,
			&us.Time, &us.PlayMode, &us.Accuracy, &us.PP,
			&us.Completed,

			&b.BeatmapID, &b.BeatmapsetID, &b.BeatmapMD5,
			&b.SongName, &b.AR, &b.OD, &b.Diff2.STD,
			&b.Diff2.Taiko, &b.Diff2.CTB, &b.Diff2.Mania,
			&b.MaxCombo, &b.HitLength, &b.Ranked,
			&b.RankedStatusFrozen, &b.LatestUpdate,
		)
		if err != nil {
			md.Err(err)
			return Err500
		}
		b.Difficulty = b.Diff2.STD
		us.Beatmap = b
		us.Rank = strings.ToUpper(getrank.GetRank(
			osuapi.Mode(us.PlayMode),
			osuapi.Mods(us.Mods),
			us.Accuracy,
			us.Count300,
			us.Count100,
			us.Count50,
			us.CountMiss,
		))
		scores = append(scores, us)
	}
	r := userScoresResponse{}
	r.Code = 200
	r.Scores = scores
	return r
}

func masterPuts(md common.MethodData, whereClause string, params ...interface{}) common.CodeMessager {
	rows, err := md.DB.Query(masterScoreSelectBase+whereClause, params...)
	if err != nil {
		md.Err(err)
		return Err500
	}
	return genericPuts(rows, md)
}

func scoresPuts(md common.MethodData, whereClause string, params ...interface{}) common.CodeMessager {
	rows, err := md.DB.Query(userScoreSelectBase+whereClause, params...)
	if err != nil {
		md.Err(err)
		return Err500
	}
	return genericPuts(rows, md)
}

func relaxPuts(md common.MethodData, whereClause string, params ...interface{}) common.CodeMessager {
	rows, err := md.DB.Query(relaxScoreSelectBase+whereClause, params...)
	if err != nil {
		md.Err(err)
		return Err500
	}
	return genericPuts(rows, md)
}
