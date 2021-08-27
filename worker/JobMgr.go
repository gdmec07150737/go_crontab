package worker

import (
	"context"
	"go.etcd.io/etcd/api/v3/mvccpb"
	clientv3 "go.etcd.io/etcd/client/v3"
	"goCrontab/common"
	"time"
)

//任务管理器
type JobMgr struct {
	client *clientv3.Client
	kv clientv3.KV
	lease clientv3.Lease
	watcher clientv3.Watcher
}

var (
	//单例
	GJobMgr *JobMgr
)

func InitJobMgr() (err error) {
	var (
		config  clientv3.Config
		client  *clientv3.Client
		kv      clientv3.KV
		lease   clientv3.Lease
		watcher clientv3.Watcher
	)
	//初始化配置项
	config = clientv3.Config{
		Endpoints:   GConfig.EtcdEndPoints,	//集群地址
		DialTimeout: time.Duration(GConfig.EtcdDialTimeout) * time.Millisecond,	//超时时间
	}
	//建立链接
	if client, err = clientv3.New(config); err != nil {
		return
	}
	//获得KV和Lease的API子集
	kv = clientv3.NewKV(client)
	lease = clientv3.NewLease(client)
	watcher = clientv3.NewWatcher(client)
	//赋值单例
	GJobMgr = &JobMgr{
		client:  client,
		kv:      kv,
		lease:   lease,
		watcher: watcher,
	}

	//启动任务监听
	GJobMgr.watchJobs()

	//启动监听killer
	GJobMgr.watchKiller()

	return
}

//监听任务
func (jobMgr *JobMgr) watchJobs() (err error) {
	var (
		getResp *clientv3.GetResponse
		kvPair *mvccpb.KeyValue
		job *common.Job
		jobEvent *common.JobEvent
		startRevision int64
		watchChan clientv3.WatchChan
		watchResp clientv3.WatchResponse
		event *clientv3.Event
		jobName string
	)
	//获取当前任务，拿到当前 Revision
	if getResp, err = GJobMgr.kv.Get(context.TODO(), common.JOB_SAVE_DIR, clientv3.WithPrefix()); err != nil {
		return
	}
	for _, kvPair = range getResp.Kvs{
		if job, err = common.UnpackJob(kvPair.Value); err == nil {
			jobEvent = common.BuildJobEvent(common.JOB_EVENT_SAVE, job)
			//推送给 scheduler
			Gscheduler.PushJobEvent(jobEvent)
		}
	}

	//从当前 Revision 开始向后监听任务变化
	go func() {
		startRevision = getResp.Header.Revision + 1
		watchChan = GJobMgr.watcher.Watch(context.TODO(), common.JOB_SAVE_DIR, clientv3.WithRev(startRevision), clientv3.WithPrefix())
		for watchResp = range watchChan{
			for _, event = range watchResp.Events{
				switch event.Type {
				case mvccpb.PUT:
					if job, err = common.UnpackJob(event.Kv.Value); err != nil {
						continue
					}
					jobEvent = common.BuildJobEvent(common.JOB_EVENT_SAVE, job)
				case mvccpb.DELETE:
					jobName = common.ExtractJobName(string(event.Kv.Key))
					job = &common.Job{Name: jobName}
					jobEvent = common.BuildJobEvent(common.JOB_EVENT_DELETE, job)
				}
				//推送给 scheduler
				Gscheduler.PushJobEvent(jobEvent)
			}
		}
	}()

	return
}

//创建任务执行锁
func (jobMgr *JobMgr) CreateJobLock(jobName string) (jobLock *JobLock) {
	jobLock = InitJobLock(jobName, jobMgr.kv, jobMgr.lease)
	return
}

//监听强杀任务通知
func (jobMgr *JobMgr) watchKiller()  {
	var (
		watchChan clientv3.WatchChan
		watchResp clientv3.WatchResponse
		event *clientv3.Event
		jobEvent *common.JobEvent
		jobName string
		job *common.Job
	)

	//监听/cron/killer目录
	go func() {
		watchChan = GJobMgr.watcher.Watch(context.TODO(), common.JOB_KILLER_DIR, clientv3.WithPrefix())
		for watchResp = range watchChan{
			for _, event = range watchResp.Events{
				switch event.Type {
				case mvccpb.PUT:
					jobName = common.ExtractKillerName(string(event.Kv.Key))
					job = &common.Job{Name: jobName}
					jobEvent = common.BuildJobEvent(common.JOB_EVENT_KILL, job)
					//推送给 scheduler
					Gscheduler.PushJobEvent(jobEvent)
				case mvccpb.DELETE:
				}
			}
		}
	}()
}