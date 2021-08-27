package worker

import (
	"goCrontab/common"
	"math/rand"
	"os/exec"
	"time"
)

//任务执行器
type Executor struct {
	
}

var (
	GExecutor *Executor
)

//执行一个任务
func (executor *Executor) ExecuteJob(info *common.JobExecuteInfo)  {
	go func() {
		var (
			cmd *exec.Cmd
			outPut []byte
			err error
			result *common.JobExecuteResult
			jobLock *JobLock
		)
		result = &common.JobExecuteResult{
			ExecuteInfo: info,
			OutPut:      make([]byte, 0),
		}
		//初始化分布式锁
		jobLock = GJobMgr.CreateJobLock(info.Job.Name)
		result.StartTime = time.Now()
		//上锁
		//随机睡眠（0-1s）
		time.Sleep(time.Duration(rand.Intn(1000)) * time.Millisecond)
		err = jobLock.TryLock()
		defer jobLock.Unlock()
		if err != nil {//上锁失败
			result.Err = err
			result.EndTime = time.Now()
		} else {
			result.StartTime = time.Now()
			//执行shell命令
			cmd = exec.CommandContext(info.CancelCtx, "/bin/bash", "-c", info.Job.Command)
			outPut, err = cmd.CombinedOutput()
			result.EndTime = time.Now()
			result.OutPut = outPut
			result.Err = err
		}
		//任务完成后，把执行结果返回给 Scheduler，Scheduler 会从 ExecutingTable 中删除掉对应的执行记录
		Gscheduler.PushJobResult(result)
	}()
}

//初始化一个执行器
func InitExecutor() (err error) {
	GExecutor = &Executor{}
	return
}