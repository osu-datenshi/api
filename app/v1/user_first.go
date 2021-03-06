// Code taken from osu!Akatsuki

package v1

import (
	"database/sql"
	"strings"

	"gopkg.in/thehowl/go-osuapi.v1"
	"github.com/osu-datenshi/api/common"
	"github.com/osu-datenshi/lib/getrank"
)

// Score is a score done.
type tuser struct {
	common.ResponseBase
	ID			int			`json:"id"`
	Username	string		`json:"username"`
	Country     string		`json:"country"`
	Scores		[]userScore `json:"scores"`
}

func UserFirstGET(md common.MethodData) common.CodeMessager {
	id := common.Int(md.Query("id"))
	if id == 0 {
		return ErrMissingField("id")
	}
	mode := 0
	m := common.Int(md.Query("mode"))
	if m != 0 {
		mode = m
	}
	var (
		r    tuser
		rows *sql.Rows
		err  error
	)
	
	// Fetch all score from users
	rows, err = md.DB.Query("SELECT s.id, s.beatmap_md5, s.score, s.max_combo, s.full_combo, s.mods, s.300_count, s.100_count, s.50_count, s.katus_count, s.gekis_count, s.misses_count, s.time, s.play_mode, s.accuracy, s.pp, s.completed, b.beatmap_id, b.beatmapset_id, b.beatmap_md5, b.song_name, b.ar, b.od, b.difficulty_std, b.difficulty_std, b.difficulty_taiko, b.difficulty_ctb, b.difficulty_mania, b.max_combo, b.hit_length, b.ranked, b.ranked_status_freezed, b.latest_update FROM scores_first as sf, scores_master as s, beatmaps as b WHERE sf.scoreid=s.id AND s.beatmap_md5=b.beatmap_md5 AND sf.userid = ? AND s.play_mode = ? " + common.Paginate(md.Query("p"), md.Query("l"), 50), id, mode)
	if err != nil {
		md.Err(err)
		return Err500
	}
	defer rows.Close()
	for rows.Next() {
		nc := userScore{}
		err = rows.Scan(&nc.Score.ID, &nc.Score.BeatmapMD5, &nc.Score.Score, &nc.Score.MaxCombo, &nc.Score.FullCombo, &nc.Score.Mods, &nc.Score.Count300, &nc.Score.Count100, &nc.Score.Count50, &nc.Score.CountKatu, &nc.Score.CountGeki, &nc.Score.CountMiss, &nc.Score.Time, &nc.Score.PlayMode, &nc.Score.Accuracy, &nc.Score.PP, &nc.Score.Completed, &nc.Beatmap.BeatmapID, &nc.Beatmap.BeatmapsetID, &nc.Beatmap.BeatmapMD5, &nc.Beatmap.SongName, &nc.Beatmap.AR, &nc.Beatmap.OD, &nc.Beatmap.Difficulty, &nc.Beatmap.Diff2.STD, &nc.Beatmap.Diff2.Taiko, &nc.Beatmap.Diff2.CTB, &nc.Beatmap.Diff2.Mania, &nc.Beatmap.MaxCombo, &nc.Beatmap.HitLength, &nc.Beatmap.Ranked, &nc.Beatmap.RankedStatusFrozen, &nc.Beatmap.LatestUpdate)
		if err != nil {
			md.Err(err)
		}
		nc.Rank = strings.ToUpper(getrank.GetRank(
			osuapi.Mode(nc.PlayMode),
			osuapi.Mods(nc.Mods),
			nc.Accuracy,
			nc.Count300,
			nc.Count100,
			nc.Count50,
			nc.CountMiss,
		))
		
		if err != nil {
			md.Err(err)
		}
		
		r.Scores = append(r.Scores, nc)
	}
	
	r.ResponseBase.Code = 200
	return r
}
