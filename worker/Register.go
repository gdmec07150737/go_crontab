package worker

import (
	"context"
	clientv3 "go.etcd.io/etcd/client/v3"
	"goCrontab/common"
	"net"
	"time"
)

type Register struct {
	client *clientv3.Client
	kv clientv3.KV
	lease clientv3.Lease
	localIP string
}

var (
	GRegister *Register
)

//初始化worker服务注册
func InitRegister() (err error) {
	var (
		config  clientv3.Config
		client  *clientv3.Client
		kv      clientv3.KV
		lease   clientv3.Lease
		localIP string
	)
	//初始化配置项
	config = clientv3.Config{
		Endpoints:   GConfig.EtcdEndPoints,	//集群地址
		DialTimeout: time.Duration(GConfig.EtcdDialTimeout) * time.Millisecond,	//超时时间
	}
	if localIP, err = getLocalIP(); err != nil {
		return
	}
	//建立链接
	if client, err = clientv3.New(config); err != nil {
		return
	}
	//获得KV和Lease的API子集
	kv = clientv3.NewKV(client)
	lease = clientv3.NewLease(client)
	GRegister = &Register{
		client:  client,
		kv:      kv,
		lease:   lease,
		localIP: localIP,
	}
	go GRegister.keepOnLine()
	return
}

//获取worker所在的机器的网卡地址
func getLocalIP() (ipv4 string, err error) {
	var (
		adds    []net.Addr
		add     net.Addr
		ipNet   *net.IPNet
		isIPNet bool
	)
	if adds, err = net.InterfaceAddrs(); err != nil {
		return
	}
	for _, add = range adds {
		//取第一个非Loopback的网卡ip
		if ipNet, isIPNet = add.(*net.IPNet); isIPNet && !ipNet.IP.IsLoopback() {
			//这个网络地址是IP地址：ipv4，ipv6
			//跳过ipv6
			if ipNet.IP.To4() != nil {
				ipv4 = ipNet.IP.String() //例如：192.168.0.1
				return
			}
		}
	}
	err = common.ERR_NO_LOCAL_IP_FOUND
	return
}

//注册到/cron/worker/ip ,并自动续租
func (register *Register) keepOnLine() () {
	var (
		err error
		leaseGranResp *clientv3.LeaseGrantResponse
		leaseId clientv3.LeaseID
		cancelCtx context.Context
		cancelFunc context.CancelFunc
		workerKey string
		leaseKeepRespChan <- chan *clientv3.LeaseKeepAliveResponse
		leaseKeepResp *clientv3.LeaseKeepAliveResponse
	)
	for {
		cancelCtx, cancelFunc = context.WithCancel(context.TODO())
		if leaseGranResp, err = register.lease.Grant(context.TODO(), 10); err != nil {
			goto RETRY
		}
		leaseId = leaseGranResp.ID
		if leaseKeepRespChan , err = register.lease.KeepAlive(cancelCtx, leaseId); err != nil {
			goto RETRY
		}
		workerKey = common.WORKER_REGISTER_DIR + register.localIP
		if _, err = register.kv.Put(context.TODO(), workerKey, "", clientv3.WithLease(leaseId)); err != nil {
			goto RETRY
		}
		for {
			select {
			case leaseKeepResp = <- leaseKeepRespChan:
				if leaseKeepResp == nil {//续租失败
					goto RETRY
				}
			}
		}
	RETRY:
		time.Sleep(1 * time.Second)
		cancelFunc()
	}

}