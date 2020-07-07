# go-micro-ci-daemon
基于Golang开发的微CI工具（守护程序）

## 2020-07-07 创建docker network后结束运行
1. 如果DOCKER网络被创建则：docker run的时候需使用--network=network参数进行重启
2. 第一次启动daemon容器时，不要带入--network参数，否则可能因为不在一个虚拟网络中导致daemon无法注册的问题

## 2020-07-07 自动部署thirdServices
1. 统一服务注册类型(当前新增支持consul)
    ```yaml
    registry:
      type: consul
      address: "micro-ci-test-consul"
      port: 8500
      useSSL: false
    ```
2. 启动容器使用的环境变量: commonEnv <- serviceEnv 并加上registry的环境变量
    ```yaml
    commonEnvs:
      key: value
      key2: value2
    thirdServices:
      micro-ci-api:
        dependsOn:
          - consul
        image:
          name: alydnh/micro:latest
        exposedPorts:
          8089/tcp:
            hostIP: 0.0.0.0
            hostPort: 7089
        env: 
          serviceKey1: value1
        args:
          - --api_address=0.0.0.0:8089
          - api
          - --handler=rpc
          - --namespace={{ .namespace }}.ci
        assertions:
          successes:
            - HTTP API Listening on
          fails:
            - Error listing endpoints
            - micro - A microservice toolkit
            - Incorrect Usage.
    ``` 
3. Registry 环境变量
MICRO_REGISTRY 微服务注册表类型如:consul
MICRO_REGISTRY_ADDRESS 注册表地址
CONSUL_HTTP_SSL consul类型不使用SSL
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