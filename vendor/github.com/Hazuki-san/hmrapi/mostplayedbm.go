package hmrapi

import (
	"github.com/osuthailand/api/common"
)

type difficulty struct {
	STD   float64 `json:"std"`
	Taiko float64 `json:"taiko"`
	CTB   float64 `json:"ctb"`
	Mania float64 `json:"mania"`
}

type beatmap struct {
	BeatmapID          int                  `json:"beatmap_id"`
	BeatmapsetID       int                  `json:"beatmapset_id"`
	BeatmapMD5         string               `json:"beatmap_md5"`
	SongName           string               `json:"song_name"`
	AR                 float32              `json:"ar"`
	OD                 float32              `json:"od"`
	Difficulty         float64              `json:"difficulty"`
	Diff2              difficulty           `json:"difficulty2"`
	MaxCombo           int                  `json:"max_combo"`
	HitLength          int                  `json:"hit_length"`
	Ranked             int                  `json:"ranked"`
	RankedStatusFrozen int                  `json:"ranked_status_frozen"`
	LatestUpdate       common.UnixTimestamp `json:"latest_update"`
	PlayCount          int                  `json:"playcount"`
	BPM                int                  `json:"bpm"`
}

type beatmapsResponse struct {
	common.ResponseBase
	Beatmaps []beatmap `json:"beatmaps"`
}

const baseBeatmapSelect = `
SELECT
	beatmap_id, beatmapset_id, beatmap_md5,
	song_name, ar, od, difficulty_std, difficulty_taiko,
	difficulty_ctb, difficulty_mania, max_combo,
	hit_length, ranked, ranked_status_freezed,
	latest_update, playcount, bpm
FROM beatmaps
`

func Beatmaps5GET(md common.MethodData) common.CodeMessager {
	var resp beatmapsResponse
	resp.Code = 200

	rows, err := md.DB.Query(baseBeatmapSelect + " ORDER BY playcount DESC LIMIT 5")
	if err != nil {
		md.Err(err)
		return common.SimpleResponse(500, "Uh oh... Seems like Aoba did something bad to API... Please try again! If it's broken... Please tell me in the Discord!")
	}
	for rows.Next() {
		var b beatmap
		err := rows.Scan(
			&b.BeatmapID, &b.BeatmapsetID, &b.BeatmapMD5,
			&b.SongName, &b.AR, &b.OD, &b.Diff2.STD, &b.Diff2.Taiko,
			&b.Diff2.CTB, &b.Diff2.Mania, &b.MaxCombo,
			&b.HitLength, &b.Ranked, &b.RankedStatusFrozen,
			&b.LatestUpdate, &b.PlayCount, &b.BPM,
		)
		if err != nil {
			md.Err(err)
			continue
		}
		resp.Beatmaps = append(resp.Beatmaps, b)
	}

	return resp
}

//Thank you Kurikku!!
//https://github.com/osukurikku/api/blob/master/vendor/github.com/KotRikD/krapi/mostplayedbm.go
