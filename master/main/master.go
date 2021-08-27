package main

import (
	"flag"
	"goCrontab/master"
	"runtime"
	"time"
)

/*var (
	FileName string
)*/

func init() {
	var (
		err error
		FileName string
	)

	//初始化线程数
	runtime.GOMAXPROCS(runtime.NumCPU())

	//命令行参数解析
	flag.StringVar(&FileName, "config", "./master.json", "指定master.json文件路径")
	flag.Parse()

	//加载配置
	if err = master.InitConfig(FileName); err != nil {
		panic(err)
	}

	//启动服务发现
	if err = master.InitWorkerMgr(); err != nil {
		panic(err)
	}


	//日志管理器
	if err = master.InitLogMgr(); err != nil {
		panic(err)
	}

	//任务管理器
	if err = master.InitJobMgr(); err != nil {
		panic(err)
	}

	//启动API HTTP服务
	if err =  master.InitApiServer(); err != nil {
		panic(err)
	}
}

func main() {
	/*var (
		err error
	)

	//加载配置
	if err = master.InitConfig(FileName); err != nil {
		goto ERR
	}

	//任务管理器
	if err = master.InitJobMgr(); err != nil {
		goto ERR
	}

	//启动API HTTP服务
	if err =  master.InitApiServer(); err != nil {
		goto ERR
	}*/

	for {
		time.Sleep(1 * time.Second)
	}
	//正常退出
	//return
/*
ERR:
	fmt.Println(err)*/
}