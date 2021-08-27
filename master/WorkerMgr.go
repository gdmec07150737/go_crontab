package master

import (
	"context"
	"go.etcd.io/etcd/api/v3/mvccpb"
	clientv3 "go.etcd.io/etcd/client/v3"
	"goCrontab/common"
	"time"
)

//服务发现管理
type WorkerMgr struct {
	client *clientv3.Client
	kv clientv3.KV
}

var (
	GWorkerMgr *WorkerMgr
)

//初始化服务发现
func InitWorkerMgr() (err error) {
	var (
		config clientv3.Config
		client *clientv3.Client
		kv clientv3.KV
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
	GWorkerMgr = &WorkerMgr{
		client: client,
		kv:     kv,
	}
	return
}

func (workerMgr *WorkerMgr) WorkersList() (workerIPArr []string, err error) {
	var (
		workerKey string
		getResp *clientv3.GetResponse
		kv *mvccpb.KeyValue
		workerIP string
	)
	workerKey = common.WORKER_REGISTER_DIR
	workerIPArr = make([]string, 0)
	if getResp, err = GWorkerMgr.kv.Get(context.TODO(), workerKey, clientv3.WithPrefix()); err != nil {
		return
	}
	for _, kv = range getResp.Kvs {
		workerIP = common.ExtractWorkerIP(string(kv.Key))
		workerIPArr = append(workerIPArr, workerIP)
	}
	return
}