package master

import (
	"encoding/json"
	"io/ioutil"
)

type ApiConfig struct {
	ApiPort int `json:"apiPort"`
	ApiReadTimeOut int `json:"apiReadTimeOut"`
	ApiWriteTimeOut int `json:"apiWriteTimeOut"`
	EtcdEndPoints []string `json:"etcdEndPoints"`
	EtcdDialTimeout int `json:"etcdDialTimeout"`
	WebRoot string `json:"webroot"`
	MongodbUrl string `json:"mongodbUrl"`
	MongodbConnectTimeout int `json:"mongodbConnectTimeout"`
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