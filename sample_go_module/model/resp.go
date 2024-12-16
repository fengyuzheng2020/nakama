package model

type GetRankResp struct {
	UserScore UserScore `json:"userScore"`
	RankInfo  string    `json:"rankInfo"`
}
