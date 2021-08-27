package master

import (
	"context"
	"encoding/json"
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
}

var (
	//单例
	GJobMgr *JobMgr
)

func InitJobMgr() (err error) {
	var (
		config clientv3.Config
		client *clientv3.Client
		kv clientv3.KV
		lease clientv3.Lease
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
	//赋值单例
	GJobMgr = &JobMgr{
		client: client,
		kv:     kv,
		lease:  lease,
	}
	return
}

//保存任务到etcd
func (jobMgr *JobMgr) SaveJob(job *common.Job) (oldJob *common.Job, err error) {
	//把任务保存到/cron/jobs/任务名 -> jobJson
	var (
		jobKey string
		jobValue []byte
		rep *clientv3.PutResponse
	)
	jobKey = common.JOB_SAVE_DIR + job.Name
	if jobValue, err  = json.Marshal(job); err != nil {
		return
	}
	//保存到etcd
	if rep, err = jobMgr.kv.Put(context.TODO(), jobKey, string(jobValue), clientv3.WithPrevKV()); err != nil {
		return
	}
	//如果是更新，则返回旧值
	if rep.PrevKv != nil {
		//对旧值反序列化
		if err = json.Unmarshal(rep.PrevKv.Value, &oldJob); err != nil {
			err = nil
			return
		}
	}
	return
}

func (jobMgr *JobMgr) DeleteJob(jobName string) (oldJob *common.Job, err error) {
	var (
		jobKey string
		resp *clientv3.DeleteResponse
	)
	jobKey = common.JOB_SAVE_DIR + jobName
	if resp, err = GJobMgr.kv.Delete(context.TODO(), jobKey, clientv3.WithPrevKV()); err != nil {
		return
	}
	//如果存在，则返回被删除的Job
	if len(resp.PrevKvs) != 0 {
		if err = json.Unmarshal(resp.PrevKvs[0].Value, &oldJob); err != nil {
			err = nil
			return
		}
	}
	return
}

func (jobMgr *JobMgr) ListJobs() (jobList []*common.Job, err error) {
	var (
		jobKey = common.JOB_SAVE_DIR
		resp *clientv3.GetResponse
		job *common.Job
		keyValue *mvccpb.KeyValue
	)
	jobList = make([]*common.Job, 0)
	if resp, err = GJobMgr.kv.Get(context.TODO(), jobKey, clientv3.WithPrefix()); err != nil {
		return
	}
	for _, keyValue = range resp.Kvs {
		job = &common.Job{}
		if err = json.Unmarshal(keyValue.Value, job); err != nil {
			err = nil
			continue
		}
		jobList = append(jobList, job)
	}
	return
}

func (jobMgr *JobMgr) KillJob(name string) (err error) {
	var (
		killerKey string
		leaseResp *clientv3.LeaseGrantResponse
		leaseId clientv3.LeaseID
	)
	//让worker监听到一次put操作，通知worker杀死对应任务（在worker里面实现）

	killerKey = common.JOB_KILLER_DIR + name
	//创建一个一秒钟的租约
	if leaseResp, err = GJobMgr.lease.Grant(context.TODO(), 1); err != nil {
		return
	}
	leaseId = leaseResp.ID
	if _, err = GJobMgr.kv.Put(context.TODO(), killerKey, "", clientv3.WithLease(leaseId)); err != nil {
		return
	}
	return
}