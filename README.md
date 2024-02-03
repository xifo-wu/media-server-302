# Media Server 302

使用 Gin 实现的简单直链服务，依赖于 Alist 302 服务。

## 配置

```yaml
server:
  # 替换成自己的挂载路径
  mount-path: /data/cloud/CloudDrive

alist:
  url: http://172.0.0.1:5244
  token: alist-xxxxx

emby:
  url: http://172.0.0.1:8096
  apikey: xxxxxx

```

`mount-path` 说明：用于替换路径前缀。例如 Emby Docker 内的路径是 /data/cloud/CloudDrive/ali-open，alist 挂载的路径 `/ali-open`， 那么此处填写 `/data/cloud/CloudDrive` 即可。

## 使用

### Dockerfile

```bash
docker run -d --name meida-server-1 -p 9096:9096 -v ./config.yaml:/config.yaml -v ./logs:/logs xifowu/meida-server-302:latest
```

### Docker Compose
只需安装 config.yaml.example 文件，修改配置即可。

```yml
version: '3'

services:
  web:
    image: "xifowu/meida-server-302:latest"
    container_name: "meida-server"
    ports:
      - "9096:9096"
    volumes:
      - ./config.yaml:/config.yaml
      - ./logs:/logs
```

## 致谢
参考 https://blog.738888.xyz/posts/emby_jellyfin_to_alist_directlink