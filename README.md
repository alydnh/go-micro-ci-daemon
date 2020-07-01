# go-micro-ci-daemon
基于Golang开发的微CI工具（守护程序）

## 2020-07-01 github workflow 新增容器打包
1. docker中CI目录为/micro-ci
2. 默认的ci文件名为ci.yaml 路径为/micro-ci/ci.yaml, 否则启动失败
3. 启动容器:
```shell script
docker run --rm \
-v /host/micro-ci/path:/micro-ci \
-v /var/run/docker.sock:/var/run/docker.sock alydnh/go-micro-ci-daemon
```
## 2020-06-30 初始提交