#Go分布式Crontab

##master
~~编译：go build~~ 

编译后的二进制文件： ./master/main/main

配置文件： ./maser/man/master.json

后台页面静态文件目录： ./master/main/webroot/

执行帮助命令
```shell script    
    ./main -help
```
执行运行master命令
```shell script    
    ./main -config ./master.json
```

[浏览器访问管理后台http://localhost:8070/](
http://localhost:8070/
)

##worker
~~编译：go build~~

编译后的二进制文件： ./worker/main/main

配置文件： ./worker/man/worker.json

执行帮助命令
```shell script    
    ./main -help
```
执行运行master命令
```shell script    
    ./main -config ./worker.json
```