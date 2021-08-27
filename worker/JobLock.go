package worker

import (
	"context"
	clientv3 "go.etcd.io/etcd/client/v3"
	"goCrontab/common"
)

//分布式锁（利用TXN事务实现）
type JobLock struct {
	kv clientv3.KV
	lease clientv3.Lease
	jobName string
	cancelFunc context.CancelFunc	//用于取消自动续约
	leaseId clientv3.LeaseID	//租约id
	isLock bool	//是否上锁成功
}

func InitJobLock(jobName string, kv clientv3.KV, lease clientv3.Lease) (jobLock *JobLock) {
	jobLock = &JobLock{
		kv:      kv,
		lease:   lease,
		jobName: jobName,
	}
	return
}

//尝试上锁
func (jobLock *JobLock) TryLock() (err error) {
	var (
		leaseGranResp *clientv3.LeaseGrantResponse
		leaseId clientv3.LeaseID
		cancelFunc context.CancelFunc	//用于取消自动续约
		ctx context.Context
		keepRespChan <- chan *clientv3.LeaseKeepAliveResponse
		txn clientv3.Txn
		lockKey string
		txnResp *clientv3.TxnResponse
	)

	//1,创建租约
	if leaseGranResp, err = jobLock.lease.Grant(context.TODO(), 5); err != nil {
		return
	}
	leaseId = leaseGranResp.ID
	ctx, cancelFunc = context.WithCancel(context.TODO())

	//2,自动续租
	if keepRespChan, err = jobLock.lease.KeepAlive(ctx, leaseId); err != nil {
		goto FAIL
	}

	//3,处理租约应答协程
	go func() {
		var (
			keepResp *clientv3.LeaseKeepAliveResponse
		)
		for {
			select {
			case keepResp = <- keepRespChan:
				if keepResp == nil{
					goto END
				}
			}
		}
		END:
	}()

	//4,创建事务
	txn = jobLock.kv.Txn(context.TODO())
	lockKey = common.JOB_LOCK_DIR + jobLock.jobName

	//5,事务抢锁
	txn.If(clientv3.Compare(clientv3.CreateRevision(lockKey), "=", 0)).
		Then(clientv3.OpPut(lockKey, "", clientv3.WithLease(leaseId))).
		Else(clientv3.OpGet(lockKey))
	if txnResp, err = txn.Commit(); err != nil {
		goto FAIL
	}

	//6,成功返回，失败释放租约
	if !txnResp.Succeeded {
		//锁被占用
		err = common.ERR_LOCK_ALREADY_REQUIRED
		goto FAIL
	}

	//抢锁成功
	jobLock.leaseId = leaseId
	jobLock.cancelFunc = cancelFunc
	jobLock.isLock = true
	return
FAIL:
	cancelFunc()
	jobLock.lease.Revoke(context.TODO(), leaseId)
	return
}

//释放锁
func (jobLock *JobLock) Unlock()  {
	if jobLock.isLock {
		//取消自动续约
		jobLock.cancelFunc()
		//取消租约
		jobLock.lease.Revoke(context.TODO(), jobLock.leaseId)
	}
}