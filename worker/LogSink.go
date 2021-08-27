package worker

import (
	"context"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"goCrontab/common"
	"time"
)

type LogSink struct {
	client *mongo.Client
	logCollection *mongo.Collection
	logChan chan *common.JobLog
	autoCommitChan chan *common.LogBatch
}

var (
	GLogSink *LogSink
)

//初始化日志服务
func InitJobLog() (err error) {
	var (
		client *mongo.Client
		ctx context.Context
		cancelFunc context.CancelFunc
	)
	ctx, cancelFunc = context.WithTimeout(
		context.TODO(),
		time.Duration(GConfig.MongodbConnectTimeout) * time.Millisecond)
	defer cancelFunc()
	if client, err =  mongo.Connect(ctx,options.Client().ApplyURI(GConfig.MongodbUrl)); err != nil {
		return
	}
	GLogSink = &LogSink{
		client:         client,
		logCollection:  client.Database("cron").Collection("log"),
		logChan:        make(chan *common.JobLog, 1000),
		autoCommitChan: make(chan *common.LogBatch, 1000),
	}
	go GLogSink.writeLoop()
	return
}

//日志存储协程
func (logSink *LogSink) writeLoop()  {
	var (
		jobLog *common.JobLog
		logBatch *common.LogBatch
		commitTimer *time.Timer
		timeoutBatch *common.LogBatch
	)
	for {
		select {
		case jobLog = <- logSink.logChan:
			if logBatch == nil {
				logBatch = &common.LogBatch{}
				//让这个批次超时自动提交
				commitTimer = time.AfterFunc(time.Duration(GConfig.JobLogCommitTimeout) * time.Millisecond,
					func(batch *common.LogBatch) func() {
						return func() {
							logSink.autoCommitChan <- batch
						}
					}(logBatch),
				)
			}
			logBatch.Logs = append(logBatch.Logs, jobLog)
			if len(logBatch.Logs) >= GConfig.JobLogBatchSize {
				//批量插入
				logSink.saveLogs(logBatch)
				//清空logBatch
				logBatch = nil
				//取消定时器
				commitTimer.Stop()
			}
		case timeoutBatch = <- logSink.autoCommitChan://过期的批次
			//判断过期的批次是否仍旧是当前的批次
			if timeoutBatch != logBatch {
				continue//跳过已经被提交的批次
			}
			//批量插入
			logSink.saveLogs(timeoutBatch)
			//清空logBatch
			logBatch = nil
		}
	}
}

//批量插入
func (logSink *LogSink) saveLogs(logBatch *common.LogBatch)  {
	logSink.logCollection.InsertMany(context.TODO(), logBatch.Logs)
}

//发送日志
func (logSink *LogSink) Append(log *common.JobLog)  {
	select {
	case logSink.logChan <- log:
	default:
		//队列满了就丢弃
	}
}