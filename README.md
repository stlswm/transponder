# golang内网穿透

#### 项目介绍
transponder分为两端：外网服务器端与内网服务器端。通过该程序内网服务器可以在没有公网IP的情况下借助外网服务器对广域网提供服务。

#### 软件架构
golang


#### 实现原理
todo

#### 安装教程

会golang的大朋友走这里

1、 先下载配置解析器 

go get https://gitee.com/stlswm/ConfigAdapter.git

2、 下载本项目

go get https://gitee.com/stlswm/transponder.git

3、 o啦


#### 使用说明

1. git clone 本项目或下载可执行文件（文件在bin目录下）

2. 外网服务端

    配置文件：
    
    注：开发环境新建 config.json 与 main.go处于同一目录即可，生产环境保证 config.json 与可执行文件在同一目录，修改配置文件后要重启服务

        
        { 
            "CommunicateServerAddress": "0.0.0.0:9090",//通讯服务监听地址，内网服务器会发起一个到该端口的连接用于与外网服务器互通有无
            "InnerServerAddress": "0.0.0.0:9091",//内网服务监听地址，内网服务器收到外网服务器通知后，会发起到该端口的连接用于处理客户端的请求
            "OuterServerAddress": "0.0.0.0:8080"//外部服务监听地址
        }

3. 内网服务端

    配置文件：
    
    注：开发环境新建 config.json 与 main.go处于同一目录即可，生产环境保证 config.json 与可执行文件在同一目录，修改配置文件后要重启服务
    
        
        {
            "CommunicateAddress": "localhost:9090",//外网服务器通讯地址（这里填写外网服务器的CommunicateServerAddress）
            "ServerAddress": "localhost:9091",//外网服务器对内网服务器的地址（这里填写外网服务器的InnerServerAddress）
            "ProxyAddress": "localhost:80"//本地目标服务
        }
    
4. 启动

   4.1 先启动外网服务 
   
   保证配置文件config.json与可执行文件在同一目录
   
    
    linux : ./bin/outer/outer (后台执行:nohup ./bin/outer/outer >> /tmp/transponder_outer.log 2>&1 &)
    
    windows: 通过cmd命令行运行 /bin/outer/outer.exe
        
   4.2 再启动内网服务 ./inner/main (或inner/main.exe)
   
   保证配置文件config.json与可执行文件在同一目录
   
    linux : ./bin/inner/inner (后台执行:nohup ./bin/inner/inner >> /tmp/transponder_inner.log 2>&1 &)
    
    windows: 通过cmd命令行运行 /bin/inner/inner.exe
		
3. nginx配置

    为了不暴露outer所监听的外部地址，可以使用nginx配置转发，同时也可以实现多主机配置。
    
    todo

#### 参与贡献

1. Fork 本项目
2. 新建 Feat_xxx 分支
3. 提交代码
4. 新建 Pull Request