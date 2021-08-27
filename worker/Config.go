package worker

import (
	"encoding/json"
	"io/ioutil"
)

type ApiConfig struct {
	EtcdEndPoints []string `json:"etcdEndPoints"`
	EtcdDialTimeout int `json:"etcdDialTimeout"`
	MongodbUrl string `json:"mongodbUrl"`
	MongodbConnectTimeout int `json:"mongodbConnectTimeout"`
	JobLogBatchSize int `json:"jobLogBatchSize"`
	JobLogCommitTimeout int `json:"jobLogCommitTimeout"`
}

var (
	//单例
	GConfig *ApiConfig
)

func InitConfig(fileName string) (err error) {
	var (
		content []byte
		config ApiConfig
	)
	if content, err = ioutil.ReadFile(fileName); err != nil {
		return
	}
	if err = json.Unmarshal(content, &config); err != nil {
		return
	}
	GConfig = &config
	return
}