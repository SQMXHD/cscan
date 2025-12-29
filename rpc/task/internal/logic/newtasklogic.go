package logic

import (
	"context"
	"encoding/json"
	"time"

	"cscan/rpc/task/internal/svc"
	"cscan/rpc/task/pb"
	"cscan/scheduler"

	"github.com/redis/go-redis/v9"
	"github.com/zeromicro/go-zero/core/logx"
)

type NewTaskLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewNewTaskLogic(ctx context.Context, svcCtx *svc.ServiceContext) *NewTaskLogic {
	return &NewTaskLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 创建执行器任务
func (l *NewTaskLogic) NewTask(in *pb.NewTaskReq) (*pb.NewTaskResp, error) {
	taskId := in.TaskId
	if taskId == "" {
		return &pb.NewTaskResp{
			Success: false,
			Message: "TaskId不能为空",
		}, nil
	}

	// 创建任务信息
	taskInfo := scheduler.TaskInfo{
		TaskId:      taskId,
		MainTaskId:  in.MainTaskId,
		TaskName:    in.TaskName,
		Config:      in.Config,
		WorkspaceId: in.WorkspaceId,
		CreateTime:  time.Now().Local().Format("2006-01-02 15:04:05"),
	}

	// 序列化任务信息
	taskJson, err := json.Marshal(taskInfo)
	if err != nil {
		l.Logger.Errorf("NewTask: failed to marshal task: %v", err)
		return &pb.NewTaskResp{
			Success: false,
			Message: "序列化任务失败: " + err.Error(),
		}, nil
	}

	// 添加到任务队列（使用时间戳作为分数，实现FIFO）
	queueKey := "cscan:task:queue"
	score := float64(time.Now().UnixNano())
	err = l.svcCtx.RedisClient.ZAdd(l.ctx, queueKey, redis.Z{
		Score:  score,
		Member: string(taskJson),
	}).Err()
	if err != nil {
		l.Logger.Errorf("NewTask: failed to add task to queue: %v", err)
		return &pb.NewTaskResp{
			Success: false,
			Message: "添加任务到队列失败: " + err.Error(),
		}, nil
	}

	l.Logger.Infof("NewTask: created task %s", taskId)

	return &pb.NewTaskResp{
		Success: true,
		Message: "Task created successfully",
	}, nil
}
