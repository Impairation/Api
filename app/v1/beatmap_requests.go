package v1

import (
	"database/sql"
	"strconv"
	"time"

	"git.zxq.co/ripple/rippleapi/beatmapget"
	"git.zxq.co/ripple/rippleapi/common"
	"git.zxq.co/ripple/rippleapi/limit"
)

type rankRequestsStatusResponse struct {
	common.ResponseBase
	QueueSize       int        `json:"queue_size"`
	MaxPerUser      int        `json:"max_per_user"`
	Submitted       int        `json:"submitted"`
	SubmittedByUser *int       `json:"submitted_by_user,omitempty"`
	CanSubmit       *bool      `json:"can_submit,omitempty"`
	NextExpiration  *time.Time `json:"next_expiration"`
}

// BeatmapRankRequestsStatusGET gets the current status for beatmap ranking requests.
func BeatmapRankRequestsStatusGET(md common.MethodData) common.CodeMessager {
	c := common.GetConf()
	rows, err := md.DB.Query("SELECT userid, time FROM rank_requests WHERE time > ? ORDER BY id ASC LIMIT "+strconv.Itoa(c.RankQueueSize), time.Now().Add(-time.Hour*24).Unix())
	if err != nil {
		md.Err(err)
		return Err500
	}
	var r rankRequestsStatusResponse
	// if it's not auth-free access and we have got ReadConfidential, we can
	// know if this user can submit beatmaps or not.
	hasConfid := md.ID() != 0 && md.User.TokenPrivileges&common.PrivilegeReadConfidential > 0
	if hasConfid {
		r.SubmittedByUser = new(int)
	}
	isFirst := true
	for rows.Next() {
		var (
			user      int
			timestamp common.UnixTimestamp
		)
		err := rows.Scan(&user, &timestamp)
		if err != nil {
			md.Err(err)
			continue
		}
		// if the user submitted this rank request, increase the number of
		// rank requests submitted by this user
		if user == md.ID() && r.SubmittedByUser != nil {
			(*r.SubmittedByUser)++
		}
		// also, if this is the first result, it means it will be the next to
		// expire.
		if isFirst {
			x := time.Time(timestamp)
			r.NextExpiration = &x
			isFirst = false
		}
		r.Submitted++
	}
	r.QueueSize = c.RankQueueSize
	r.MaxPerUser = c.BeatmapRequestsPerUser
	if hasConfid {
		x := r.Submitted < r.QueueSize && *r.SubmittedByUser < r.MaxPerUser
		r.CanSubmit = &x
	}
	r.Code = 200
	return r
}

type submitRequestData struct {
	ID    int `json:"id"`
	SetID int `json:"set_id"`
}

// BeatmapRankRequestsSubmitPOST submits a new beatmap for ranking approval.
func BeatmapRankRequestsSubmitPOST(md common.MethodData) common.CodeMessager {
	var d submitRequestData
	err := md.RequestData.Unmarshal(&d)
	if err != nil {
		return ErrBadJSON
	}
	// check json data is present
	if d.ID == 0 && d.SetID == 0 {
		return ErrMissingField("id|set_id")
	}

	// you've been rate limited
	if !limit.NonBlockingRequest("rankrequest:u:"+strconv.Itoa(md.ID()), 5) {
		return common.SimpleResponse(429, "You may only try to request 5 beatmaps per minute.")
	}
	if !limit.NonBlockingRequest("rankrequest:ip:"+md.C.ClientIP(), 8) {
		return common.SimpleResponse(429, "You may only try to request 8 beatmaps per minute from the same IP.")
	}

	// find out from BeatmapRankRequestsStatusGET if we can submit beatmaps.
	statusRaw := BeatmapRankRequestsStatusGET(md)
	status, ok := statusRaw.(rankRequestsStatusResponse)
	if !ok {
		// if it's not a rankRequestsStatusResponse, it means it's an error
		return statusRaw
	}
	if !*status.CanSubmit {
		return common.SimpleResponse(403, "It's not possible to do a rank request at this time.")
	}

	if d.SetID == 0 {
		d.SetID, err = beatmapget.Beatmap(d.ID)
	} else {
		err = beatmapget.Set(d.SetID)
	}
	if err == beatmapget.ErrBeatmapNotFound {
		return common.SimpleResponse(404, "That beatmap could not be found anywhere!")
	}
	if err != nil {
		md.Err(err)
		return Err500
	}

	err = md.DB.QueryRow("SELECT 1 FROM rank_requests WHERE bid = ? AND type = ? AND time > ?",
		d.SetID, "s", time.Now().Unix()).Scan(new(int))
	switch err {
	case sql.ErrNoRows:
		break
	case nil:
		// TODO: return beatmap
		// we're returning a success because if the request was already sent in the past 24
		// hours, it's as if the user submitted it.
		return common.SimpleResponse(200, "Your request has been submitted.")
	default:
		md.Err(err)
		return Err500
	}

	_, err = md.DB.Exec(
		"INSERT INTO rank_requests (userid, bid, type, time, blacklisted) VALUES (?, ?, ?, ?, 0)",
		md.ID(), d.SetID, "s", time.Now().Unix())
	if err != nil {
		md.Err(err)
		return Err500
	}

	// TODO: return beatmap
	return common.SimpleResponse(200, "Your request has been submitted.")
}