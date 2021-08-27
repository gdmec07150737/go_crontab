package worker

import (
	"fmt"
	"goCrontab/common"
	"time"
)

//任务调度器
type Scheduler struct {
	JobEventChan chan *common.JobEvent
	JobPlanTable map[string]*common.JobSchedulerPlan	//任务调度计划表
	JobExecutingTable map[string]*common.JobExecuteInfo	//任务执行表
	JobResultChan chan *common.JobExecuteResult	//任务结果队列
}

var (
	Gscheduler *Scheduler
)

//处理任务事件
func (scheduler *Scheduler) handleJobEvent(jobEvent *common.JobEvent)  {
	var (
		jobSchedulerPlan *common.JobSchedulerPlan
		jobExecuteInfo *common.JobExecuteInfo
		jobExecuting bool
		err error
		jobIsExist bool
	)
	switch jobEvent.EventType {
	case common.JOB_EVENT_SAVE:	//保存任务事件
		if jobSchedulerPlan, err = common.BuildSchedulerPlan(jobEvent.Job); err != nil {
			return
		}
		scheduler.JobPlanTable[jobEvent.Job.Name] = jobSchedulerPlan
	case common.JOB_EVENT_DELETE:	//删除任务事件
		if jobSchedulerPlan, jobIsExist = scheduler.JobPlanTable[jobEvent.Job.Name]; jobIsExist {}
		delete(scheduler.JobPlanTable, jobEvent.Job.Name)
	case common.JOB_EVENT_KILL:	//强杀任务事件
		//取消Command执行
		if jobExecuteInfo, jobExecuting = scheduler.JobExecutingTable[jobEvent.Job.Name]; jobExecuting {
			//取消任务command执行
			jobExecuteInfo.CancelFunc()
		}
	}
}

//重新计算任务调度状态
func (scheduler *Scheduler) TrySchedule() (ScheduleAfter time.Duration) {
	var (
		jobPlan *common.JobSchedulerPlan
		now time.Time
		nearTime *time.Time
	)
	//如果没有任务，则自行设置睡眠时间
	if len(scheduler.JobPlanTable) == 0 {
		ScheduleAfter = 1 * time.Second
		return
	}
	now = time.Now()
	//遍历所有任务
	for _, jobPlan = range scheduler.JobPlanTable{
		if jobPlan.NextTime.Before(now) || jobPlan.NextTime.Equal(now){
			//fmt.Println("执行任务：", jobPlan.Job.Name)
			scheduler.TryStartJob(jobPlan)
			//更新下次执行时间
			jobPlan.NextTime = jobPlan.Expr.Next(now)
		}
		//统计最近要过期的任务时间
		if nearTime == nil || jobPlan.NextTime.Before(*nearTime) {
			nearTime = &jobPlan.NextTime
		}
	}
	//下次调度间隔（最近执行的调度时间 - 当前时间）
	ScheduleAfter = (*nearTime).Sub(now)
	return
}

//尝试执行任务
func (scheduler *Scheduler) TryStartJob(jobPlan *common.JobSchedulerPlan)  {
	var (
		jobExecuteInfo *common.JobExecuteInfo
		jobExecExist   bool
	)
	//调度和执行是分开的
	//执行任务可能会很久，假设执行耗时1分钟，任务定时是1秒，则1分钟会调度60次，但只能执行1次，防止并发

	//如果任务正在执行跳过本次调度
	if jobExecuteInfo, jobExecExist = scheduler.JobExecutingTable[jobPlan.Job.Name]; jobExecExist {
		//fmt.Println("任务正在执行，跳过执行任务：", jobExecuteInfo.Job)
		return
	}
	//构建执行状态信息
	jobExecuteInfo = common.BuildJobExecuteInfo(jobPlan)
	//保存执行状态信息
	scheduler.JobExecutingTable[jobPlan.Job.Name] = jobExecuteInfo
	//执行任务
	fmt.Println("执行任务：", jobExecuteInfo.Job.Name, jobExecuteInfo.PlanTime, jobExecuteInfo.RealTime)
	GExecutor.ExecuteJob(jobExecuteInfo)
}

//处理任务结果
func (scheduler *Scheduler) handleJobResult(result *common.JobExecuteResult)  {
	var (
		jobLog *common.JobLog
	)

	//删除执行状态
	delete(scheduler.JobExecutingTable, result.ExecuteInfo.Job.Name)

	if result.Err != common.ERR_LOCK_ALREADY_REQUIRED {
		jobLog = &common.JobLog{
			JobName:      result.ExecuteInfo.Job.Name,
			Command:      result.ExecuteInfo.Job.Command,
			//Err:          "",
			Output:       string(result.OutPut),
			PlanTime:     result.ExecuteInfo.PlanTime.UnixNano() / 1000 / 1000,
			ScheduleTime: result.ExecuteInfo.RealTime.UnixNano() / 1000 /1000,
			StartTime:    result.StartTime.UnixNano() / 1000 / 1000,
			EndTime:      result.EndTime.UnixNano() / 1000 / 1000,
		}
		if result.Err != nil {
			jobLog.Err = result.Err.Error()
		} else {
			jobLog.Err = ""
		}
		GLogSink.Append(jobLog)
	}

	//fmt.Println("任务完成：", result.ExecuteInfo.Job.Name, string(result.OutPut), result.Err)
}

//调度协程(消费任务)
func (scheduler *Scheduler) schedulerLoop()  {
	var (
		jobEvent *common.JobEvent
		scheduleAfter time.Duration
		scheduleTimer *time.Timer
		jobResult *common.JobExecuteResult
	)
	//初始化（默认一秒）
	scheduleAfter = scheduler.TrySchedule()
	//创建定时器
	scheduleTimer = time.NewTimer(scheduleAfter)
	for {
		select {
		case jobEvent = <- scheduler.JobEventChan:
			scheduler.handleJobEvent(jobEvent)
		case <- scheduleTimer.C:	//最近的任务到期了
		case jobResult = <- scheduler.JobResultChan:	//监听任务执行结果
			scheduler.handleJobResult(jobResult)
		}
		//调度一次任务
		scheduleAfter = scheduler.TrySchedule()
		//重置调度间隔
		scheduleTimer.Reset(scheduleAfter)
	}

}

//推送任务
func (scheduler *Scheduler) PushJobEvent(jobEvent *common.JobEvent)  {
	scheduler.JobEventChan <- jobEvent
}

//初始化调度器
func InitScheduler() (err error) {
	Gscheduler = &Scheduler{
		JobEventChan:      make(chan *common.JobEvent, 1000),
		JobPlanTable:      make(map[string]*common.JobSchedulerPlan),
		JobExecutingTable: make(map[string]*common.JobExecuteInfo),
		JobResultChan:     make(chan *common.JobExecuteResult, 1000),
	}
	go Gscheduler.schedulerLoop()
	return
}

//回传任务执行结果
func (scheduler *Scheduler) PushJobResult(result *common.JobExecuteResult)  {
	scheduler.JobResultChan <- result
}