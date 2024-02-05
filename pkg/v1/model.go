package v1

import "net/url"

type QueryParam struct {
	Year       string
	Department string
	Code       string
}

func (p *QueryParam) ToReqParam() url.Values {
	return url.Values{
		"act":    {"check"},
		"basic":  {"checkd"},
		"status": {""},
		"year":   {"202401"},
		"dwmc":   {p.Department},
		"gwdm":   {p.Code},
	}
}

type QueryResp struct {
	// html
	Str string `json:"str"`
	// 1：成功
	Code int `json:"code"`
}

func (r QueryResp) OK() bool {
	return r.Code == 1
}

type JobDetail struct {
	Code             string
	City             string
	Department       string
	JobName          string
	RecruitsNumber   int
	Educational      string
	Remark           string
	ApplicantsNumber int
	ratio            float64
}

func (d *JobDetail) GetRatio() float64 {
	if d.RecruitsNumber == 0 {
		return 0
	}
	return float64(d.RecruitsNumber) / float64(d.ApplicantsNumber)
}
