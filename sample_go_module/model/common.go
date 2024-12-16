package model

type UserCacheData struct {
	UserId    string    `json:"userId"`
	UserScore UserScore `json:"userScore"`
}

// 定义结构体
type UserScore struct {
	UserId        string `json:"userId"`
	PowerScore    int    `json:"powerScore"`
	ExistingScore int    `json:"existingScore"`
	TotalScore    int    `json:"totalScore"`
}
