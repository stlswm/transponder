# golang内网穿透工具V1.0（支持windows与linux）

#### 项目介绍
transponder内网穿透工具分为两端：外网服务器端与内网服务器端。

通过该程序内网服务器可以在没有公网IP的情况下借助外网服务器对广域网提供服务。

该工具支持windows与linux等不同的操作系统。

#### 软件架构
golang


#### 工作原理

1、外网服务器监听端口9090（默认，防火墙要添加入站规则）为内网服务器提供服务，用于内网服务器与外网服务器的通讯

2、外网服务器监听端口9091（默认，防火墙要添加入站规则）为内网服务器提供服务，用于接收内网服务器连接，暂存内网服务器连接进入连接池，将来会使用连接池中的连接为外网连接提供服务

3、外网服务器使用使用unix套接字提供外网服务（或监听端口8080，防火墙要添加入站规则），用于接收nginx转发过来的请求

4、内网服务器连接9090端口并保持心跳，保证与外网服务器的通讯


### 工作流程

1、外网服务unix套接字（或8080端口）接收外网网络请求

2、发送通知给内网服务器（内网服务器通过端口9090保持与外网服务器的通讯，有自动重连机制）

3、内网服务器收到外网服务器通知立即建立一个到外网服务器9091的新连接，同时建立与内部转发服务器的TCP连接

4、外网服务器从连接池取出内网连接（来原于9091端口）等待与第1步中的外网连接进行数据交换

5、外网服务器将第1步的外网连接和第4步的内网连接进行数据交换

6、内网服务器将外网服务器连接和内部转发连接进行数据交换

7、转发完成，内网服务器释放资源关闭连接，外网服务器释放资源关闭连接

#### 安装教程

会golang的大朋友走这里，不会的小朋友可跳过这里

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
            "CommunicateServerAddress": "tcp://0.0.0.0:9090",//通讯服务监听地址，内网服务器会发起一个到该端口的连接用于与外网服务器互通有无
            "InnerServerAddress": "tcp://0.0.0.0:9091",//内网服务监听地址，内网服务器收到外网服务器通知后，会发起到该端口的连接用于处理客户端的请求
            "OuterServerAddress": "tcp://0.0.0.0:8080"//外部服务监听地址
            //"OuterServerAddress": "unix:///var/run/transponderouter"//linux unix套接字的网络模式（linux建议使用该模式）
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
        
   4.2 再启动内网服务
   
   保证配置文件config.json与可执行文件在同一目录
   
    linux : ./bin/inner/inner (后台执行:nohup ./bin/inner/inner >> /tmp/transponder_inner.log 2>&1 &)
    
    windows: 通过cmd命令行运行 /bin/inner/inner.exe
		
#### nginx配置

可以使用nginx配置转发，同时也可以实现多主机配置。

linux服务器推荐使用unix套接字网络模式加快转发效率，并且可以少占用一个端口（）。

windows只能使用端口转发。

    server {
		listen 80;
		server_name  www.abc.com;
	 
		access_log  /var/log/www.abc.com.access.log  main;
		error_log  /var/log/www.abc.com.error.log;
		#root   html;
		#index  index.html index.htm index.php;
	 
		#send request to transponderouter.socket
		location / {
			proxy_pass http://unix:/var/run/transponderouter.socket:/;
			
			#proxy settings
			proxy_redirect     off;
			proxy_set_header   Host             $host;
			proxy_set_header   X-Real-IP        $remote_addr;
			proxy_set_header   X-Forwarded-For  $proxy_add_x_forwarded_for;
			proxy_next_upstream error timeout invalid_header http_500 http_502 http_503 http_504;
			proxy_max_temp_file_size 0;
			proxy_connect_timeout      90;
			proxy_send_timeout         90;
			proxy_read_timeout         90;
			proxy_buffer_size          4k;
			proxy_buffers              4 32k;
			proxy_busy_buffers_size    64k;
			proxy_temp_file_write_size 64k;
	   }
	}

#### 参与贡献

1. Fork 本项目
2. 新建 Feat_xxx 分支
3. 提交代码
4. 新建 Pull Request