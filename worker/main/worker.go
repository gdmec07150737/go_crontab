package main

import (
	"flag"
	"goCrontab/worker"
	"runtime"
	"time"
)

func init() {
	var (
		err error
		fileName string
	)

	//初始化线程数
	runtime.GOMAXPROCS(runtime.NumCPU())

	//命令行参数解析
	flag.StringVar(&fileName, "config", "./worker.json", "指定 worker.json文件路径")
	flag.Parse()

	//加载配置
	if err = worker.InitConfig(fileName); err != nil {
		panic(err)
	}

	//启动服务注册
	if err = worker.InitRegister(); err != nil {
		panic(err)
	}

	//启动日志协程
	if err = worker.InitJobLog(); err != nil {
		panic(err)
	}

	//启动执行器
	if err = worker.InitExecutor(); err != nil {
		panic(err)
	}

	//启动调度器
	if err = worker.InitScheduler(); err != nil {
		panic(err)
	}

	//任务管理器
	if err = worker.InitJobMgr(); err != nil {
		panic(err)
	}
}

func main() {
	for {
		time.Sleep(1 * time.Second)
	}
}
