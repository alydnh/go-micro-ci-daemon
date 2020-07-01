# go-micro-ci-daemon
基于Golang开发的微CI工具（守护程序）

## 2020-07-01 启动go-micro
1. 根据ci.yaml中的registry配置使用相应的registry启动micro service
    ```yaml
    registry:
      type: consul
      address: "{{ .v2 }}-consul"
      port: 8500
      useSSL: false
    ```
2. 启动服务的名称对应对ci.yaml中的 namespace.name
    ```yaml
    name: micro-ci-test
    namespace: test.{{ .v1 }}
    ```
## 2020-07-01 github workflow 新增容器打包
1. docker中CI目录为/micro-ci
2. 默认的ci文件名为ci.yaml 路径为/micro-ci/ci.yaml, 否则启动失败
3. 启动容器:
    ```shell script
    docker run --rm \
    -v /host/micro-ci/path:/micro-ci \
    -v /var/run/docker.sock:/var/run/docker.sock alydnh/go-micro-ci-daemon
    ```