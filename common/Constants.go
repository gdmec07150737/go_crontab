package common

const (
	//任务保存目录
	JOB_SAVE_DIR = "/cron/jobs/"

	//任务强杀目录
	JOB_KILLER_DIR = "/cron/killer/"

	//任务锁目录
	JOB_LOCK_DIR = "/cron/lock/"

	//worker服务注册的目录
	WORKER_REGISTER_DIR = "/cron/workers/"

	//Job事件PUT类型
	JOB_EVENT_SAVE = 1

	//Job事件DELETE类型
	JOB_EVENT_DELETE = 2

	//Job事件KILL类型
	JOB_EVENT_KILL = 3
)
