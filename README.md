# Media Server 302

使用 Gin 实现的简单直链服务，依赖于 Alist 302 服务。

## 如何使用

### 前置条件

- 已经安装了 Alist，Emby 以及挂载路径（如 Rclone Mount 或 Clouddrive2）


### 配置文件
新建一个 config.yaml 文件，内容如下：

```yaml
server:
  # 替换成自己的挂载路径
  # 如果你的 Emby 运行在 Windows 下，可以向下面这样填 mount-page: "F:" (大概是这样吧)
  mount-path: /data/cloud/CloudDrive

alist:
  url: http://172.0.0.1:5244
  # 替换成你的 Alist 公网可访问地址, infuse, emby ios 客户端需要。
  # 填公网地址可修复 infuse 进度条问题。
  pubilc-url: https://alist.xxxx.xxx
  token: alist-xxxxx

emby:
  url: http://172.0.0.1:8096
  apikey: xxxxxx

```

`mount-path` 说明：用于替换路径前缀。例如 Emby Docker 内的路径是 /data/cloud/CloudDrive/ali-open，alist 挂载的路径 `/ali-open`， 那么此处填写 `/data/cloud/CloudDrive` 即可。


### Dockerfile

```bash
docker run -d --name media-server-1 -p 9096:9096 -v ./config.yaml:/config.yaml -v ./logs:/logs xifowu/media-server-302:latest
```

### Docker Compose
只需安装 config.yaml.example 文件，修改配置即可。

```yml
version: '3'

services:
  web:
    image: "xifowu/media-server-302:latest"
    container_name: "media-server"
    ports:
      - "9096:9096"
    volumes:
      - ./config.yaml:/config.yaml
      - ./logs:/logs
```

## FAQ

##### 为什么我的 /d/xxxxxx/xxxx.mkv 返回 401?
> Alist 3.30.0 之后默认开了 Sign 签名全部，本程序虽然适配了生成 Sign，但仍有可能失效。实在不行你可以去 Alist 全局配置里关掉

## 致谢
参考 https://blog.738888.xyz/posts/emby_jellyfin_to_alist_directlink