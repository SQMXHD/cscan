package logic

import (
	"context"
	"encoding/json"

	"cscan/rpc/task/internal/svc"
	"cscan/rpc/task/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetPocValidationResultLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetPocValidationResultLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetPocValidationResultLogic {
	return &GetPocValidationResultLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 查询POC验证结果
func (l *GetPocValidationResultLogic) GetPocValidationResult(in *pb.GetPocValidationResultReq) (*pb.GetPocValidationResultResp, error) {
	taskId := in.TaskId
	if taskId == "" {
		return &pb.GetPocValidationResultResp{
			Success: false,
			Message: "TaskId不能为空",
			Status:  "ERROR",
		}, nil
	}

	// 从Redis获取验证结果（与Worker保存的key一致）
	resultKey := "cscan:task:result:" + taskId
	resultData, err := l.svcCtx.RedisClient.Get(l.ctx, resultKey).Result()
	if err != nil {
		// 检查任务是否还在执行中
		taskInfoKey := "cscan:task:info:" + taskId
		taskInfoData, infoErr := l.svcCtx.RedisClient.Get(l.ctx, taskInfoKey).Result()
		if infoErr == nil && taskInfoData != "" {
			var taskInfo map[string]interface{}
			if json.Unmarshal([]byte(taskInfoData), &taskInfo) == nil {
				status, _ := taskInfo["status"].(string)
				if status == "" || status == "PENDING" || status == "STARTED" {
					// 任务还在执行中
					return &pb.GetPocValidationResultResp{
						Success: true,
						Message: "任务执行中",
						Status:  "RUNNING",
					}, nil
				}
			}
		}
		// 未找到结果
		return &pb.GetPocValidationResultResp{
			Success: false,
			Message: "未找到验证结果",
			Status:  "NOT_FOUND",
		}, nil
	}

	// 解析结果数据
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(resultData), &result); err != nil {
		return &pb.GetPocValidationResultResp{
			Success: false,
			Message: "解析结果失败",
			Status:  "ERROR",
		}, nil
	}

	// 获取状态
	status, _ := result["status"].(string)
	if status == "" {
		status = "COMPLETED"
	}

	// 解析验证结果列表
	var pbResults []*pb.PocValidationResult
	if results, ok := result["results"].([]interface{}); ok {
		for _, r := range results {
			if rMap, ok := r.(map[string]interface{}); ok {
				pbResult := &pb.PocValidationResult{
					PocId:      getString(rMap, "pocId"),
					PocName:    getString(rMap, "pocName"),
					TemplateId: getString(rMap, "templateId"),
					Severity:   getString(rMap, "severity"),
					Matched:    getBool(rMap, "matched"),
					MatchedUrl: getString(rMap, "matchedUrl"),
					Details:    getString(rMap, "details"),
					Output:     getString(rMap, "output"),
					PocType:    getString(rMap, "pocType"),
				}
				if tags, ok := rMap["tags"].([]interface{}); ok {
					for _, t := range tags {
						if s, ok := t.(string); ok {
							pbResult.Tags = append(pbResult.Tags, s)
						}
					}
				}
				pbResults = append(pbResults, pbResult)
			}
		}
	}

	// 获取更新时间
	updateTime, _ := result["updateTime"].(string)
	createTime, _ := result["createTime"].(string)

	return &pb.GetPocValidationResultResp{
		Success:        true,
		Message:        "success",
		Status:         status,
		Results:        pbResults,
		CompletedCount: int32(len(pbResults)),
		TotalCount:     int32(len(pbResults)),
		UpdateTime:     updateTime,
		CreateTime:     createTime,
	}, nil
}

// 辅助函数：安全获取字符串
func getString(m map[string]interface{}, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

// 辅助函数：安全获取布尔值
func getBool(m map[string]interface{}, key string) bool {
	if v, ok := m[key].(bool); ok {
		return v
	}
	return false
}
