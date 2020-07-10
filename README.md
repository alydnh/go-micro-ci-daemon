# go-micro-ci-daemon
基于Golang开发的微CI工具（守护程序）

## 2020-07-10 完成 `保存部署清单文件`
1. 启动容器前扫描配置是否变更，按需启动
2. 启动完成后在micro-ci/.ci/deployments/{service}文件中保存相应配置
    ```yaml
    args:
      - --api_address=0.0.0.0:8089
      - api
      - --handler=rpc
      - --namespace=test.micro-ci.ci
    env:
        CONSUL_HTTP_SSL: "0"
        MICRO_REGISTER_INTERVAL: "30"
        MICRO_REGISTER_TTL: "60"
        MICRO_REGISTRY: consul
        MICRO_REGISTRY_ADDRESS: micro-ci-test-consul:8500
        MICRO_SERVER_ID: default
        MICRO_SERVER_VERSION: v0.0.1
    serviceName: micro-ci-api
    containerName: micro-ci-test-micro-ci-api
    containerID: 2b3e02dbdbe24a7ead01b086ce7af5c4aa29cf18c010b921d58d4cc2681dfc7b
    dockerImageID: sha256:581acc38dfb061640d8cb46f1d6088da87fa1e2cb1a7e95674565e3ecf069381
    exposedPorts:
        8089/tcp:
            hostIP: 0.0.0.0
            hostPort: 7089
    mounts: {}
    ```
3. 更新ci-common update vendor

## 2020-07-08 daemon RPC 服务
1. 修改go.yml ci文件，打包时支持LDFLAGS注入版本信息
2. ci/service.go 新增version接口，获取当前版本信息
3. 启动日志加入version打印
4. go mod vendor

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