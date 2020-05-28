# golang内网穿透工具V2.0（支持windows与linux）

#### 项目介绍
V2版本重写了内核代码可大幅提高转发速率。

transponder内网穿透工具分为两端：外网服务器端与内网服务器端。

通过该程序内网服务器可以在没有公网IP的情况下借助外网服务器对广域网提供服务（适用于有回调接口的第方应用的开发调试，如：微信公众号、支付宝接口调试）。

该工具支持windows与linux等不同的操作系统。

该项目使用的为P2P中继模式非UDP打洞模式（适用于http协议的内网穿透）

参考链接：https://blog.csdn.net/yunlianglinfeng/article/details/54018113

#### 软件架构
golang


#### 工作原理

1、外网服务器监听端口9090（默认，防火墙要添加入站规则）为内网服务器提供服务，用于内网服务器与外网服务器的通讯并暂存内网服务器反向连接进入连接池

2、外网服务器使用使用unix套接字提供外网服务（或监听端口8080，防火墙要添加入站规则），用于接收nginx转发过来的请求（也可以直接监听端口）

3、连接通路：外网nginx->外网unix套接字->外网9090->内网连接->内部80。这样就借助外网服务器的9090端口成功的连接了外网80到内网80的连接通路。

内网服务器会每秒检测当前连接到外网服务器的待命连接数，少于10个后会再主动发送一批连接并通过ping包维护与外网服务器的连接。

### 工作流程

![工作流程](https://www.stlswm.com/uploads/20190119115122.jpg)

1、内网服务器批量建立到外网服务器的9090端口的连接并授权

2、外网服务unix套接字（或8080端口）接收外网网络请求并从内网到9090的连接池中取出连接，发送转发指令

3、内网服务器收到转发指令并回复转发待命指令

4、外网服务器收到转发待命请求开发送数据，开启数据转发

5、转发完成，内网服务器释放资源关闭连接，外网服务器释放资源关闭连接

#### 安装教程

会golang的大朋友走这里，不会的小朋友可跳过这里

1、 先下载配置解析器（安装到GOPATH的src目录）

git clone https://gitee.com/stlswm/ConfigAdapter.git

2、 下载本项目（安装到GOPATH的src目录）

git clone https://gitee.com/stlswm/transponder.git

3、 o啦


#### 配置说明

1. git clone 本项目或下载可执行文件（文件在bin目录下）

2. 外网服务端

    配置文件：
    
    注：开发环境新建 outer.config.json 与 与outer_server.go处于同一目录即可，生产环境保证 outer.config.json 与outer_server(与outer_server.exe)可执行文件在同一目录，修改配置文件后要重启服务

        
        { 
            "InnerServerAddress": "tcp://0.0.0.0:9090",//内网服务监听地址，内网服务器收到外网服务器通知后，会发起到该端口的连接用于处理客户端的请求
            "OuterServerAddress": "tcp://0.0.0.0:8080",//外部服务监听地址，用于对客户端提供服务
            "OuterServerAddress": "unix:///var/run/transponderouter.sock",//linux unix套接字的网络模式（linux建议使用该模式）
            "AuthKey":"123456"//连接授权码（内外网必须保持一致）
        }

3. 内网服务端

    配置文件：
    
    注：开发环境新建 inner.config.json 与 inner_server.go处于同一目录即可，生产环境保证 inner.config.json 与inner_server(与inner_server.exe)可执行文件在同一目录，修改配置文件后要重启服务
    
        
        {
            "RegisterAddress": "tcp://外网服务器ip:9090",//外网服务器对内网服务器的地址（这里填写外网服务器的InnerServerAddress）
            "ProxyAddress": "tcp://127.0.0.1:80",//本地目标服务
            "AuthKey":"123456",//连接授权码（内外网必须保持一致）
            "MaxFreeConn": 50//最大空闲连接数，并发数高时可适当增大该参数
        }
    
4. 启动

   4.1 先启动外网服务 
   
   保证配置文件outer.config.json与可执行文件在同一目录
   
    
    linux : ./outer_server (后台执行:nohup ./outer_server >> /tmp/transponder_outer.log 2>&1 &)
    
    windows: 通过cmd命令行运行 ./outer_server.exe
        
   4.2 再启动内网服务
   
   保证配置文件inner.config.json与可执行文件在同一目录
   
    linux : ./inner_server (后台执行:nohup ./inner_server >> /tmp/transponder_inner.log 2>&1 &)
    
    windows: 通过cmd命令行运行 ./inner_server.exe
		
#### nginx配置

可以使用nginx配置转发，同时也可以实现多主机配置。

linux服务器推荐使用unix套接字网络模式加快转发效率，并且可以少占用一个端口。

windows只能使用端口转发。

    server {
		listen 80;
		server_name  www.abc.com;
	 
		access_log  /var/log/www.abc.com.access.log;
		error_log  /var/log/www.abc.com.error.log;
		#root   html;
		#index  index.html index.htm index.php;
	 
		#send request to transponderouter.socket
		location / {
			proxy_pass http://unix:/var/run/transponderouter.sock:/;
			
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