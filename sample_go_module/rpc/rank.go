package rpc

import (
	"context"
	"database/sql"
	"encoding/json"
	"github.com/heroiclabs/nakama-common/api"
	"github.com/heroiclabs/nakama-common/runtime"
	"github.com/heroiclabs/nakama/v3/sample_go_module/cache"
	"github.com/heroiclabs/nakama/v3/sample_go_module/define"
	"github.com/heroiclabs/nakama/v3/sample_go_module/model"
	"github.com/heroiclabs/nakama/v3/sample_go_module/utils"
	sort "sort"
)

type PlayerData struct {
	UserID            string `json:"userId"`
	UserName          string `json:"userName"`
	Attack            int    `json:"attack"`            // 战力
	CurrentPoints     int    `json:"currentPoints"`     // 获得总积分
	RankPoints        int    `json:"rankPoints"`        // 战力排序积分
	ParticipateInRank int    `json:"participateInRank"` // 是否参与排名
}

func GetPlayersStorageData(ctx context.Context, nk runtime.NakamaModule, userIDs []string, key string) (map[string]string, error) {
	// 构建批量读取请求
	readObjects := make([]*runtime.StorageRead, len(userIDs))
	for i, userID := range userIDs {
		readObjects[i] = &runtime.StorageRead{
			Collection: "player_data", // 与客户端写入的 Collection 保持一致
			Key:        key,
			UserID:     userID, // 提供用户 ID
		}
	}

	// 调用 Nakama 的读取存储对象 API
	objects, err := nk.StorageRead(ctx, readObjects)
	if err != nil {
		return nil, err
	}

	// 处理返回的数据，将结果存储到 map 中
	result := make(map[string]string)
	for _, obj := range objects {
		if obj.Key == key {
			result[obj.GetUserId()] = obj.Value
		}
	}

	// 返回结果
	return result, nil
}

// 获取排行榜数据
func GetFinalLeaderboard(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
	logger.Info("Precomputing final leaderboard...")
	req := &model.CommonReq{}
	err := utils.ParseJsonParam(payload, req)
	if err != nil {
		logger.Error("Failed to parse json param: %v", err)
		return "", err
	}
	// 从缓存中读取最终排行榜
	objects, err := nk.StorageRead(ctx, []*runtime.StorageRead{
		{
			Collection: "leaderboards",
			Key:        "final_rankings",
		},
	})
	if err != nil || len(objects) == 0 {
		logger.Error("Failed to fetch cached leaderboard: %v", err)
		return "", err
	}

	resp := &model.GetRankResp{
		UserScore: model.UserScore{},
		RankInfo:  objects[0].Value,
	}
	v, ok := cache.UserCacheInstance.Get(req.UserId)
	if ok {
		resp.UserScore = v.UserScore
	}
	r, _ := json.Marshal(resp)
	// 返回 JSON 数据
	return string(r), nil
}

func PrecomputeLeaderboard(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule) error {
	cursor := ""
	limit := 1000
	finalLeaderboard := []model.UserScore{}

	for {
		records, nextCursor, err := FetchPaginatedPowerRankings(ctx, logger, nk, limit, cursor)
		if err != nil {
			return err
		}
		// 计算战力积分
		powerScores := CalculatePowerScores(records)
		userIds := make([]string, 0, len(powerScores))
		for userId, _ := range powerScores {
			userIds = append(userIds, userId)
		}
		// 获取玩家已有积分数据
		playerResources, err := GetPlayersStorageData(ctx, nk, userIds, "user_resource")
		if err != nil {
			return err
		}
		// 解析
		playerScores := make(map[string]int)
		for playerId, userRes := range playerResources {
			var res = make(map[int]int)
			err = json.Unmarshal([]byte(userRes), &res)
			if err != nil {
				logger.Error("Failed to unmarshal player data: %v", err)
				return err
			}
			playerScores[playerId] = res[define.ScoreId]
		}

		// 计算总积分
		batchTotalScores := CalculateTotalScores(powerScores, playerScores)
		finalLeaderboard = append(finalLeaderboard, batchTotalScores...)

		if nextCursor == "" {
			break
		}
		cursor = nextCursor
	}

	// 对总积分排序
	finalLeaderboard = SortByTotalScores(finalLeaderboard)
	for _, score := range finalLeaderboard {
		// 缓存
		if v, ok := cache.UserCacheInstance.Get(score.UserId); ok {
			v.UserScore = score
			cache.UserCacheInstance.Set(score.UserId, v)
		} else {
			cache.UserCacheInstance.Set(score.UserId, model.UserCacheData{
				UserId:    score.UserId,
				UserScore: score,
			})
		}
	}
	finalLeaderboardBytes, _ := json.Marshal(map[string]interface{}{
		"finalRankings": finalLeaderboard,
	})
	// 存储到 Nakama
	_, err := nk.StorageWrite(ctx, []*runtime.StorageWrite{
		{
			Collection:      "leaderboards",
			Key:             "final_rankings",
			Value:           string(finalLeaderboardBytes), // JSON序列化后的结果
			PermissionRead:  2,
			PermissionWrite: 0,
		},
	})
	if err != nil {
		logger.Error("Failed to store leaderboard data: %v", err)
		return err
	}

	return nil
}

func SortByTotalScores(totalScores []model.UserScore) []model.UserScore {
	sort.Slice(totalScores, func(i, j int) bool {
		return totalScores[i].TotalScore > totalScores[j].TotalScore
	})
	return totalScores
}

func CalculateTotalScores(powerScores, playerScores map[string]int) []model.UserScore {
	totalScores := make([]model.UserScore, 0)
	for userID, powerScore := range powerScores {
		existingScore := playerScores[userID]
		totalScore := powerScore + existingScore
		totalScores = append(totalScores, model.UserScore{UserId: userID, PowerScore: powerScore, ExistingScore: existingScore, TotalScore: totalScore})
	}
	return totalScores
}

func CalculatePowerScores(records []*api.LeaderboardRecord) map[string]int {
	powerScores := make(map[string]int)
	n := len(records)
	for i, record := range records {
		// 倒数第 i 名的积分 = i + 1
		powerScores[record.OwnerId] = n - i
	}
	return powerScores
}

func FetchPaginatedPowerRankings(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, limit int, cursor string) ([]*api.LeaderboardRecord, string, error) {
	records, _, nextCursor, _, err := nk.LeaderboardRecordsList(ctx, "global_attack_rank", nil, limit, cursor, 0)
	if err != nil {
		logger.Error("Failed to fetch leaderboard page: %v", err)
		return nil, "", err
	}
	return records, nextCursor, nil
}

func FetchPlayerScores(ctx context.Context, db *sql.DB) (map[string]int, error) {
	query := `SELECT user_id, score FROM players_table;`
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	playerScores := make(map[string]int)
	for rows.Next() {
		var userID string
		var score int
		if err := rows.Scan(&userID, &score); err != nil {
			return nil, err
		}
		playerScores[userID] = score
	}
	return playerScores, nil
}

//func RefreshRankPoints(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule) error {
//
//}

//func InitModule(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, initializer runtime.Initializer) error {
//	if err := initializer.RegisterRpc("refresh_rank_points", RefreshRankPoints); err != nil {
//		logger.Error("Failed to register RPC: %v", err)
//		return err
//	}
//	logger.Info("Module initialized successfully.")
//	return nil
//}
