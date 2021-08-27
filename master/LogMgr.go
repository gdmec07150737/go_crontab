package master

import (
	"context"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"goCrontab/common"
	"time"
)

type LogMgr struct {
	client *mongo.Client
	collection *mongo.Collection
}

var (
	GLogMgr *LogMgr
)

//初始化日志管理
func InitLogMgr() (err error) {
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
	GLogMgr = &LogMgr{
		client:     client,
		collection: client.Database("cron").Collection("log"),
	}
	return
}

//查看日志列表
func (logMgr *LogMgr) ListLog(name string, skip int64, limit int64) (logArr []*common.JobLog, err error) {
	var (
		filter *common.JobLogFilter
		ops *options.FindOptions
		cursor *mongo.Cursor
		jobLog *common.JobLog
	)
	logArr = make([]*common.JobLog, 0)
	filter = &common.JobLogFilter{JobName: name}
	ops = &options.FindOptions{
		Limit: &limit,
		Skip:  &skip,
		Sort:  &common.SortLogByStartTime{SortOrder: -1},
	}
	if cursor, err = logMgr.collection.Find(context.TODO(), filter, ops); err != nil {
		return
	}
	defer cursor.Close(context.TODO())
	for cursor.Next(context.TODO()) {
		jobLog = &common.JobLog{}
		if err = cursor.Decode(jobLog); err != nil {
			continue
		}
		logArr = append(logArr, jobLog)
	}
	return
}