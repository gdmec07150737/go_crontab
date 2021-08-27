package master

import (
	"encoding/json"
	"goCrontab/common"
	"net"
	"net/http"
	"strconv"
	"time"
)

//任务的http接口
type ApiServer struct {
	httpServer *http.Server
}

var (
	//单例对象
	GApiServer *ApiServer
)

//保存任务接口
//POST job={"name":"job1", "command":"echo hello", "cronExpr":"* * * * * *"}
func HandleJobSave(resp http.ResponseWriter, req *http.Request) {
	var (
		err       error
		postJob   string
		job       common.Job
		oldJob    *common.Job
		respValue []byte
	)

	//1，解析POST表单
	if err = req.ParseForm(); err != nil {
		goto ERR
	}
	//2，取表单中的job字段
	postJob = req.PostForm.Get("job")
	//3，将job反序列化
	if err = json.Unmarshal([]byte(postJob), &job); err != nil {
		goto ERR
	}
	//4，将Job任务保存到etcd中
	if oldJob, err = GJobMgr.SaveJob(&job); err != nil {
		goto ERR
	}
	//5，返回正常应答（{"errno":0, "msg":"...", "data":{....}）
	if respValue, err = common.BuildResponse(0, "success", oldJob); err == nil {
		resp.Write(respValue)
	}
	return
ERR:
	//5，返回异常应答
	if respValue, err = common.BuildResponse(-1, err.Error(), nil); err == nil {
		resp.Write(respValue)
	}
}

func HandleJobDelete(resp http.ResponseWriter, req *http.Request) {
	var (
		err       error
		jobName   string
		oldJob    *common.Job
		respValue []byte
	)
	if err = req.ParseForm(); err != nil {
		goto ERR
	}
	jobName = req.PostForm.Get("jobName")
	//删除etcd中的任务
	if oldJob, err = GJobMgr.DeleteJob(jobName); err != nil {
		goto ERR
	}
	//正常返回
	if respValue, err = common.BuildResponse(0, "success", oldJob); err == nil {
		resp.Write(respValue)
	}
	return
ERR:
	//异常返回
	if respValue, err = common.BuildResponse(-1, err.Error(), nil); err == nil {
		resp.Write(respValue)
	}
}

func HandleJobList(resp http.ResponseWriter, req *http.Request) {
	var (
		err error
		jobsList []*common.Job
		respValue []byte
	)
	if jobsList, err = GJobMgr.ListJobs(); err != nil {
		goto ERR
	}
	//正常返回
	if respValue, err = common.BuildResponse(0, "success", jobsList); err == nil {
		resp.Write(respValue)
	}
	return
ERR:
	//异常返回
	if respValue, err = common.BuildResponse(-1, err.Error(), nil); err == nil {
		resp.Write(respValue)
	}
}

func HandleJobKill(resp http.ResponseWriter, req *http.Request) {
	var (
		err error
		name string
		respValue []byte
	)
	if err = req.ParseForm(); err != nil {
		goto ERR
	}
	name = req.PostForm.Get("name")
	if err = GJobMgr.KillJob(name); err != nil {
		goto ERR
	}
	//正常返回
	if respValue, err = common.BuildResponse(0, "success", nil); err == nil {
		resp.Write(respValue)
	}
	return
ERR:
	//异常返回
	if respValue, err = common.BuildResponse(-1, err.Error(), nil); err == nil {
		resp.Write(respValue)
	}
}

func HandleJobLog(resp http.ResponseWriter, req *http.Request) {
	var (
		err error
		jobName string
		skipForm string
		limitForm string
		skip int
		limit int
		logArr []*common.JobLog
		respValue []byte
	)
	if err = req.ParseForm(); err != nil {
		goto ERR
	}
	jobName = req.Form.Get("name")
	skipForm = req.Form.Get("skip")
	limitForm = req.Form.Get("limit")
	if skip, err = strconv.Atoi(skipForm); err  != nil {
		skip = 0
	}
	if limit, err = strconv.Atoi(limitForm); err  != nil {
		limit = 20
	}
	if logArr, err =GLogMgr.ListLog(jobName, int64(skip), int64(limit)); err != nil {
		goto ERR
	}
	//正常返回
	if respValue, err = common.BuildResponse(0, "success", logArr); err == nil {
		resp.Write(respValue)
	}
	return
ERR:
	//异常返回
	if respValue, err = common.BuildResponse(-1, err.Error(), nil); err == nil {
		resp.Write(respValue)
	}
}

func HandleWorkerList(resp http.ResponseWriter, req *http.Request) {
	var (
		workerIPArr []string
		err error
		respValue []byte
	)
	if workerIPArr, err =GWorkerMgr.WorkersList(); err != nil {
		goto ERR
	}
	//正常返回
	if respValue, err = common.BuildResponse(0, "success", workerIPArr); err == nil {
		resp.Write(respValue)
	}
	return
ERR:
	//异常返回
	if respValue, err = common.BuildResponse(-1, err.Error(), nil); err == nil {
		resp.Write(respValue)
	}
}

//初始化服务
func InitApiServer() (err error) {
	var (
		mux        *http.ServeMux
		listen     net.Listener
		httpServer *http.Server
		staticDir http.Dir
		staticHandle http.Handler
	)

	//配置路由
	mux = http.NewServeMux()
	mux.HandleFunc("/job/save", HandleJobSave)
	mux.HandleFunc("/job/delete", HandleJobDelete)
	mux.HandleFunc("/job/list", HandleJobList)
	mux.HandleFunc("/job/kill", HandleJobKill)
	mux.HandleFunc("/job/log", HandleJobLog)
	mux.HandleFunc("/worker/list", HandleWorkerList)

	//静态文件路由
	staticDir = http.Dir(GConfig.WebRoot)
	staticHandle = http.FileServer(staticDir)
	mux.Handle("/", http.StripPrefix("/", staticHandle))

	//启动TCP监听
	if listen, err = net.Listen("tcp", ":"+strconv.Itoa(GConfig.ApiPort)); err != nil {
		return
	}

	//创建一个HTTP服务
	httpServer = &http.Server{
		Handler:      mux,
		ReadTimeout:  time.Duration(GConfig.ApiReadTimeOut) * time.Millisecond,
		WriteTimeout: time.Duration(GConfig.ApiWriteTimeOut) * time.Millisecond,
	}

	//赋值单例
	GApiServer = &ApiServer{httpServer: httpServer}

	//启动服务端
	go httpServer.Serve(listen)

	return
}
