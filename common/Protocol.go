package common

import (
	"context"
	"encoding/json"
	"github.com/gorhill/cronexpr"
	"strings"
	"time"
)

//定时任务
type Job struct {
	Name string `json:"name"`	//表达式
	Command	string `json:"command"`	//shell命令
	CronExpr string	`json:"cronExpr"`	//cron表达式
}

//任务执行计划
type JobSchedulerPlan struct {
	Job *Job
	Expr *cronexpr.Expression	//解析好的 crontab 的时间表达式
	NextTime time.Time	//下一次调度的时间
}

//任务执行状态
type JobExecuteInfo struct {
	Job *Job
	PlanTime time.Time	//计划执行时间
	RealTime time.Time	//实际执行时间
	CancelCtx context.Context	//用于取消任务的上下文
	CancelFunc context.CancelFunc	//用于取消任务的func
}

//任务执行结果
type JobExecuteResult struct {
	ExecuteInfo *JobExecuteInfo	//执行状态
	OutPut []byte	//脚本输出
	Err error	//脚本出错内容
	StartTime time.Time	//开始时间
	EndTime time.Time	//结束时间
}

//任务执行日志
type JobLog struct {
	JobName string `json:"jobName" bson:"jobName"`	//任务名字
	Command string `json:"command" bson:"command"`	//任务脚本命令
	Err string `json:"err" bson:"err"`	//错误原因
	Output	string `json:"output" bson:"output"`	//脚本输出
	PlanTime int64 `json:"planTime" bson:"planTime"`	//计划开始时间
	ScheduleTime int64 `json:"scheduleTime" bson:"scheduleTime"`	//实际调度时间
	StartTime int64 `json:"startTime" bson:"startTime"`	//任务开始时间
	EndTime int64 `json:"endTime" bson:"endTime"`	//任务结束时间
}

//日志批次
type LogBatch struct {
	Logs []interface{}	//多条日志
}

//任务日志过滤条件
type JobLogFilter struct {
	JobName string `bson:"jobName"`
}

//任务日志排序规则
type SortLogByStartTime struct {
	SortOrder int `bson:"startTime"`	//{startTime:-1}
}


//HTTP接口应答
type Response struct {
	Errno int `json:"errno"`
	Msg string `json:"msg"`
	Data interface{} `json:"data"`
}

type JobEvent struct {
	EventType int	`json:"eventType"`
	Job *Job	`json:"job"`
}

//应答方法
func BuildResponse(errno int, msg string, data interface{}) (resp []byte, err error) {
	var (
		response Response
	)
	response.Errno = errno
	response.Msg = msg
	response.Data = data
	resp, err = json.Marshal(response)
	return
}

//反序列化Job
func UnpackJob(value []byte) (ret *Job, err error) {
	var (
		job *Job
	)

	job = &Job{}
	if err = json.Unmarshal(value, job); err != nil {
		return
	}
	ret = job
	return
}

//发生Job任务变化事件
func BuildJobEvent(eventType int, job *Job) (jobEvent *JobEvent) {
	return &JobEvent{
		EventType: eventType,
		Job:     job,
	}
}

//根据 jobKey 获取任务名
func ExtractJobName(jobKey string) string {
	return strings.TrimPrefix(jobKey, JOB_SAVE_DIR)
}

//根据 killerKey 获取任务名
func ExtractKillerName(killerKey string) string {
	return strings.TrimPrefix(killerKey, JOB_KILLER_DIR)
}

//根据 resKey 获取IP名
func ExtractWorkerIP(resKey string) string {
	return strings.TrimPrefix(resKey, WORKER_REGISTER_DIR)
}

//构造执行计划
func BuildSchedulerPlan(job *Job) (jobSchedulerPlan *JobSchedulerPlan, err error) {
	var (
		expr     *cronexpr.Expression
	)
	if expr, err = cronexpr.Parse(job.CronExpr); err != nil {
		return
	}
	jobSchedulerPlan = &JobSchedulerPlan{
		Job:      job,
		Expr:     expr,
		NextTime: expr.Next(time.Now()),
	}
	return
}

//构造执行状态信息
func BuildJobExecuteInfo(jobSchedulerPlan *JobSchedulerPlan) (jobExecuteInfo *JobExecuteInfo) {
	jobExecuteInfo = &JobExecuteInfo{
		Job:      jobSchedulerPlan.Job,
		PlanTime: jobSchedulerPlan.NextTime,
		RealTime: time.Now(),
	}
	jobExecuteInfo.CancelCtx, jobExecuteInfo.CancelFunc = context.WithCancel(context.TODO())
	return
}