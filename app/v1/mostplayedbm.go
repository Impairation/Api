package v1

import (
	"zxq.co/ripple/rippleapi/common"
)

type difficultys struct {
	STD   float64 `json:"std"`
	Taiko float64 `json:"taiko"`
	CTB   float64 `json:"ctb"`
	Mania float64 `json:"mania"`
}

type beatmaps struct {
	BeatmapID          int                  `json:"beatmap_id"`
	BeatmapsetID       int                  `json:"beatmapset_id"`
	BeatmapMD5         string               `json:"beatmap_md5"`
	SongName           string               `json:"song_name"`
	AR                 float32              `json:"ar"`
	OD                 float32              `json:"od"`
	Difficulty         float64              `json:"difficulty"`
	Diff2              difficulty           `json:"difficulty2"` // fuck nyo
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
	Beatmaps []beatmaps `json:"beatmaps"`
}

const baseBeatmapmpSelect = `
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

	rows, err := md.DB.Query(baseBeatmapmpSelect + " ORDER BY playcount DESC LIMIT 5")
	if err != nil {
		md.Err(err)
		return common.SimpleResponse(500, "An error occurred. Trying again may work. If it doesn't, yell at this Kotorikku instance admin and tell them to fix the API.")
	}
	for rows.Next() {
		var b beatmaps
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
