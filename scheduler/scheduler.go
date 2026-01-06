package scheduler

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/robfig/cron/v3"
)

// 任务状态常量
const (
	TaskStatusCreated = "CREATED"
	TaskStatusPending = "PENDING"
	TaskStatusStarted = "STARTED"
	TaskStatusSuccess = "SUCCESS"
	TaskStatusFailure = "FAILURE"
	TaskStatusRevoked = "REVOKED"
)

// TaskInfo 任务信息
type TaskInfo struct {
	TaskId      string   `json:"taskId"`
	MainTaskId  string   `json:"mainTaskId"`
	WorkspaceId string   `json:"workspaceId"`
	TaskName    string   `json:"taskName"`
	Config      string   `json:"config"`
	Priority    int      `json:"priority"`
	CreateTime  string   `json:"createTime"`
	Workers     []string `json:"workers,omitempty"` // 指定执行任务的 Worker 列表，为空表示任意 Worker
}

// Scheduler 任务调度器
type Scheduler struct {
	rdb         *redis.Client
	cron        *cron.Cron
	queueKey    string
	processingKey string
	mu          sync.Mutex
	handlers    map[string]TaskHandler
}

// TaskHandler 任务处理函数
type TaskHandler func(ctx context.Context, task *TaskInfo) error

// NewScheduler 创建调度器
func NewScheduler(rdb *redis.Client) *Scheduler {
	return &Scheduler{
		rdb:           rdb,
		cron:          cron.New(cron.WithSeconds()),
		queueKey:      "cscan:task:queue",
		processingKey: "cscan:task:processing",
		handlers:      make(map[string]TaskHandler),
	}
}

// RegisterHandler 注册任务处理器
func (s *Scheduler) RegisterHandler(taskName string, handler TaskHandler) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.handlers[taskName] = handler
}

// Start 启动调度器
func (s *Scheduler) Start() {
	s.cron.Start()
}

// Stop 停止调度器
func (s *Scheduler) Stop() {
	s.cron.Stop()
}

// AddCronTask 添加定时任务
func (s *Scheduler) AddCronTask(spec string, taskFunc func()) (cron.EntryID, error) {
	return s.cron.AddFunc(spec, taskFunc)
}

// RemoveCronTask 移除定时任务
func (s *Scheduler) RemoveCronTask(id cron.EntryID) {
	s.cron.Remove(id)
}

// GetWorkerQueueKey 获取 Worker 专属队列的 Key
func (s *Scheduler) GetWorkerQueueKey(workerName string) string {
	return fmt.Sprintf("cscan:task:queue:worker:%s", strings.ToLower(workerName))
}

// PushTask 推送任务到队列
// 如果任务指定了 Workers，则推送到每个 Worker 的专属队列
// 否则推送到公共队列
func (s *Scheduler) PushTask(ctx context.Context, task *TaskInfo) error {
	if task.TaskId == "" {
		task.TaskId = uuid.New().String()
	}
	task.CreateTime = time.Now().Local().Format("2006-01-02 15:04:05")

	data, err := json.Marshal(task)
	if err != nil {
		return err
	}

	// 使用优先级队列，分数越小优先级越高
	score := float64(time.Now().Unix()) - float64(task.Priority*1000)

	// 如果指定了 Workers，推送到每个 Worker 的专属队列
	if len(task.Workers) > 0 {
		pipe := s.rdb.Pipeline()
		for _, workerName := range task.Workers {
			workerQueueKey := s.GetWorkerQueueKey(workerName)
			pipe.ZAdd(ctx, workerQueueKey, redis.Z{
				Score:  score,
				Member: data,
			})
		}
		_, err = pipe.Exec(ctx)
		return err
	}

	// 没有指定 Worker，推送到公共队列
	return s.rdb.ZAdd(ctx, s.queueKey, redis.Z{
		Score:  score,
		Member: data,
	}).Err()
}

// PushTaskBatch 批量推送任务到队列（使用 Pipeline 提高性能）
// 如果任务指定了 Workers，则推送到每个 Worker 的专属队列
// 否则推送到公共队列
func (s *Scheduler) PushTaskBatch(ctx context.Context, tasks []*TaskInfo) error {
	if len(tasks) == 0 {
		return nil
	}

	pipe := s.rdb.Pipeline()
	baseTime := time.Now()

	for i, task := range tasks {
		if task.TaskId == "" {
			task.TaskId = uuid.New().String()
		}
		task.CreateTime = baseTime.Local().Format("2006-01-02 15:04:05")

		data, err := json.Marshal(task)
		if err != nil {
			continue
		}

		// 使用优先级队列，分数越小优先级越高
		// 同一批次的任务按顺序递增分数，保持顺序
		score := float64(baseTime.Unix()) - float64(task.Priority*1000) + float64(i)*0.001

		// 如果指定了 Workers，推送到每个 Worker 的专属队列
		if len(task.Workers) > 0 {
			for _, workerName := range task.Workers {
				workerQueueKey := s.GetWorkerQueueKey(workerName)
				pipe.ZAdd(ctx, workerQueueKey, redis.Z{
					Score:  score,
					Member: data,
				})
			}
		} else {
			// 没有指定 Worker，推送到公共队列
			pipe.ZAdd(ctx, s.queueKey, redis.Z{
				Score:  score,
				Member: data,
			})
		}
	}

	_, err := pipe.Exec(ctx)
	return err
}

// PopTask 从队列获取任务
func (s *Scheduler) PopTask(ctx context.Context) (*TaskInfo, error) {
	// 获取优先级最高的任务
	results, err := s.rdb.ZPopMin(ctx, s.queueKey, 1).Result()
	if err != nil {
		return nil, err
	}
	if len(results) == 0 {
		return nil, nil
	}

	var task TaskInfo
	if err := json.Unmarshal([]byte(results[0].Member.(string)), &task); err != nil {
		return nil, err
	}

	// 添加到处理中集合
	s.rdb.SAdd(ctx, s.processingKey, task.TaskId)

	return &task, nil
}

// CompleteTask 完成任务
func (s *Scheduler) CompleteTask(ctx context.Context, taskId string) error {
	return s.rdb.SRem(ctx, s.processingKey, taskId).Err()
}

// GetQueueLength 获取队列长度
func (s *Scheduler) GetQueueLength(ctx context.Context) (int64, error) {
	return s.rdb.ZCard(ctx, s.queueKey).Result()
}

// GetProcessingCount 获取处理中任务数
func (s *Scheduler) GetProcessingCount(ctx context.Context) (int64, error) {
	return s.rdb.SCard(ctx, s.processingKey).Result()
}

// TaskConfig 任务配置
type TaskConfig struct {
	PortScan     *PortScanConfig     `json:"portscan,omitempty"`
	PortIdentify *PortIdentifyConfig `json:"portidentify,omitempty"` // 端口识别（Nmap服务识别）
	DomainScan   *DomainScanConfig   `json:"domainscan,omitempty"`
	Fingerprint  *FingerprintConfig  `json:"fingerprint,omitempty"`
	PocScan      *PocScanConfig      `json:"pocscan,omitempty"`
}

type PortScanConfig struct {
	Enable            bool   `json:"enable"`
	Tool              string `json:"tool"`              // tcp, masscan, naabu
	Ports             string `json:"ports"`
	Rate              int    `json:"rate"`
	Timeout           int    `json:"timeout"`           // 端口扫描超时时间(秒)，默认5秒
	PortThreshold     int    `json:"portThreshold"`     // 开放端口数量阈值，超过则过滤该主机
	ScanType          string `json:"scanType"`          // s=SYN, c=CONNECT，默认 c
	SkipHostDiscovery bool   `json:"skipHostDiscovery"` // 跳过主机发现 (-Pn)
}

// PortIdentifyConfig 端口识别配置（Nmap服务识别）
type PortIdentifyConfig struct {
	Enable  bool   `json:"enable"`
	Timeout int    `json:"timeout"` // 单个主机超时时间(秒)，默认30秒
	Args    string `json:"args"`    // Nmap额外参数，如 "-sV --version-intensity 5"
}

type DomainScanConfig struct {
	Enable             bool     `json:"enable"`
	Subfinder          bool     `json:"subfinder"`          // 使用Subfinder
	Timeout            int      `json:"timeout"`            // 超时时间(秒)
	MaxEnumerationTime int      `json:"maxEnumerationTime"` // 最大枚举时间(分钟)
	Threads            int      `json:"threads"`            // 并发线程数
	RateLimit          int      `json:"rateLimit"`          // 速率限制
	Sources            []string `json:"sources"`            // 指定数据源
	ExcludeSources     []string `json:"excludeSources"`     // 排除数据源
	All                bool     `json:"all"`                // 使用所有数据源(慢)
	Recursive          bool     `json:"recursive"`          // 只使用递归数据源
	RemoveWildcard     bool     `json:"removeWildcard"`     // 移除泛解析域名
	ResolveDNS         bool     `json:"resolveDNS"`         // 是否解析DNS
	Concurrent         int      `json:"concurrent"`         // DNS解析并发数
}

type FingerprintConfig struct {
	Enable        bool   `json:"enable"`
	Tool          string `json:"tool"`          // 探测工具: httpx, builtin (wappalyzer)
	Httpx         bool   `json:"httpx"`         // 已废弃，使用Tool字段
	IconHash      bool   `json:"iconHash"`
	Wappalyzer    bool   `json:"wappalyzer"`    // 已废弃，builtin模式自动启用
	CustomEngine  bool   `json:"customEngine"`  // 使用自定义指纹引擎（ARL格式）
	Screenshot    bool   `json:"screenshot"`
	Timeout       int    `json:"timeout"`       // 总超时时间(秒)，默认300秒
	TargetTimeout int    `json:"targetTimeout"` // 单个目标超时时间(秒)，默认30秒
	Concurrency   int    `json:"concurrency"`   // 指纹识别并发数，默认10
}

type PocScanConfig struct {
	Enable            bool                `json:"enable"`
	PocTypes          []string            `json:"pocTypes"`          // nuclei, builtin
	PocFiles          []string            `json:"pocFiles"`          // 自定义POC文件
	UseNuclei         bool                `json:"useNuclei"`         // 使用Nuclei扫描
	AutoScan          bool                `json:"autoScan"`          // 基于自定义标签映射自动扫描
	AutomaticScan     bool                `json:"automaticScan"`     // 基于Wappalyzer内置映射自动扫描（类似nuclei -as）
	Severity          string              `json:"severity"`          // 严重级别过滤
	Tags              []string            `json:"tags"`              // 手动指定标签
	ExcludeTags       []string            `json:"excludeTags"`       // 排除标签
	RateLimit         int                 `json:"rateLimit"`         // 速率限制
	Concurrency       int                 `json:"concurrency"`       // 并发数
	TargetTimeout     int                 `json:"targetTimeout"`     // 单个目标超时时间(秒)，默认600秒
	CustomPocOnly     bool                `json:"customPocOnly"`     // 只使用自定义POC
	CustomTemplates   []string            `json:"customTemplates"`   // 自定义POC模板内容(YAML) - 已废弃
	NucleiTemplates   []string            `json:"nucleiTemplates"`   // 从数据库获取的模板内容(YAML) - 已废弃
	NucleiTemplateIds []string            `json:"nucleiTemplateIds"` // Nuclei模板ID列表（新）
	CustomPocIds      []string            `json:"customPocIds"`      // 自定义POC ID列表（新）
	TagMappings       map[string][]string `json:"tagMappings"`       // 应用名称到Nuclei标签的映射
}

// ParseTaskConfig 解析任务配置
func ParseTaskConfig(configStr string) (*TaskConfig, error) {
	var config TaskConfig
	if err := json.Unmarshal([]byte(configStr), &config); err != nil {
		return nil, err
	}
	return &config, nil
}

// BuildTaskConfig 构建任务配置
func BuildTaskConfig(config *TaskConfig) (string, error) {
	data, err := json.Marshal(config)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// TaskResult 任务结果
type TaskResult struct {
	TaskId     string `json:"taskId"`
	Status     string `json:"status"`
	Message    string `json:"message"`
	AssetCount int    `json:"assetCount"`
	VulCount   int    `json:"vulCount"`
	Duration   int64  `json:"duration"`
}

// FormatResult 格式化结果
func (r *TaskResult) FormatResult() string {
	return fmt.Sprintf("状态:%s 资产:%d 漏洞:%d 耗时:%ds",
		r.Status, r.AssetCount, r.VulCount, r.Duration)
}
