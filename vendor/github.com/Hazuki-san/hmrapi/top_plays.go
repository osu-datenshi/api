package hmrapi

import (
	"fmt"
	"strings"

	"gopkg.in/thehowl/go-osuapi.v1"
	"github.com/osuthailand/api/common"
	"zxq.co/x/getrank"
)

type miniUser struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
}

type Score2 struct {
	ID         int                  `json:"id"`
	BeatmapMD5 string               `json:"beatmap_md5"`
	Score      int64                `json:"score"`
	MaxCombo   int                  `json:"max_combo"`
	FullCombo  bool                 `json:"full_combo"`
	Mods       int                  `json:"mods"`
	Count300   int                  `json:"count_300"`
	Count100   int                  `json:"count_100"`
	Count50    int                  `json:"count_50"`
	CountGeki  int                  `json:"count_geki"`
	CountKatu  int                  `json:"count_katu"`
	CountMiss  int                  `json:"count_miss"`
	Time       common.UnixTimestamp `json:"time"`
	PlayMode   int                  `json:"play_mode"`
	Accuracy   float64              `json:"accuracy"`
	PP         float32              `json:"pp"`
	Rank       string               `json:"rank"`
	Completed  int                  `json:"completed"`
}

type MixedBeatmap struct {
	Score2
	Beatmap beatmap  `json:"beatmap"`
	User    miniUser `json:"user"`
}

type ScoresResponse struct {
	common.ResponseBase
	Scores []MixedBeatmap `json:"scores"`
}

const topPlaysQuery = `
SELECT
	scores.id, scores.beatmap_md5, scores.score,
	scores.max_combo, scores.full_combo, scores.mods,
	scores.300_count, scores.100_count, scores.50_count,
	scores.gekis_count, scores.katus_count, scores.misses_count,
	scores.time, scores.play_mode, scores.accuracy, scores.pp,
	scores.completed,

	beatmaps.beatmap_id, beatmaps.beatmapset_id, beatmaps.beatmap_md5,
	beatmaps.song_name, beatmaps.ar, beatmaps.od, beatmaps.difficulty_std,
	beatmaps.difficulty_taiko, beatmaps.difficulty_ctb, beatmaps.difficulty_mania,
	beatmaps.max_combo, beatmaps.hit_length, beatmaps.ranked,
	beatmaps.ranked_status_freezed, beatmaps.latest_update,

	users.id, users.username
FROM scores
INNER JOIN beatmaps ON beatmaps.beatmap_md5 = scores.beatmap_md5
INNER JOIN users ON users.id = scores.userid
WHERE scores.pp > 0 AND scores.completed = '3' AND users.privileges & 1 > 0 AND scores.play_mode = %s
ORDER BY scores.pp DESC
`

func TopPlaysGET(md common.MethodData) common.CodeMessager {
	limit := md.HasQuery("l")
	limitQuery := " LIMIT 50"
	if limit {
		limitQuery = " LIMIT " + md.Query("l")
	}
	mode := md.Query("mode")

	rows, err := md.DB.Query(fmt.Sprintf(topPlaysQuery, mode) + limitQuery)
	if err != nil {
		md.Err(err)
		return common.SimpleResponse(500, "Uh oh... Seems like Aoba did something bad to API... Please try again! If it's broken... Please tell me in the Discord!")
	}
	var scores []MixedBeatmap
	for rows.Next() {
		var (
			us MixedBeatmap
			b  beatmap
		)
		err = rows.Scan(
			&us.ID, &us.BeatmapMD5, &us.Score2.Score,
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

			&us.User.ID, &us.User.Username,
		)
		if err != nil {
			md.Err(err)
			return common.SimpleResponse(500, "Uh oh... Seems like Aoba did something bad to API... Please try again! If it's broken... Please tell me in the Discord!")
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
	r := ScoresResponse{}
	r.Code = 200
	r.Scores = scores
	return r

}

//Thank you Kurikku!!
//https://github.com/osukurikku/api/blob/master/vendor/github.com/KotRikD/krapi/top_plays.go
