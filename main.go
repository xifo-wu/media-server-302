package main

import (
	"fmt"
	"media-server-302/pkg/alist"
	"media-server-302/pkg/config"
	"media-server-302/pkg/emby"
	"media-server-302/pkg/logger"
	"net/http"
	"net/http/httputil"
	"net/url"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
)

func extractIDFromPath(path string) (string, error) {
	// 编译正则表达式
	re := regexp.MustCompile(`/[Vv]ideos/(\S+)/(stream|original|master)`)
	// 执行匹配操作
	matches := re.FindStringSubmatch(path)

	// 如果找到匹配项，第一个分组就是我们想要的视频ID
	if len(matches) >= 2 {
		return matches[1], nil
	}

	// 如果没有匹配项，返回错误
	return "", fmt.Errorf("no match found")
}

func main() {
	config.Init()
	log := logger.Init()
	r := gin.Default()

	embyURL := viper.GetString("emby.url")
	url, _ := url.Parse(embyURL)

	proxy := httputil.NewSingleHostReverseProxy(url)

	r.Any("/*actions", func(c *gin.Context) {

		currentURI := c.Request.RequestURI
		videoID, err := extractIDFromPath(currentURI)
		if err != nil {
			proxy.ServeHTTP(c.Writer, c.Request)
			return
		}

		mediaSourceID := c.Query("MediaSourceId")
		if mediaSourceID == "" {
			mediaSourceID = c.Query("mediaSourceId")
		}

		if videoID == "" || mediaSourceID == "" {
			proxy.ServeHTTP(c.Writer, c.Request)
			return
		}

		itemInfoUri, itemId, etag, mediaSourceId, apiKey := emby.GetItemPathInfo(c)
		embyRes, err := emby.GetEmbyItems(itemInfoUri, itemId, etag, mediaSourceId, apiKey)

		if err != nil {
			log.Error(fmt.Sprintf("获取 Emby 失败。错误信息: %v", err))
			proxy.ServeHTTP(c.Writer, c.Request)
			return
		}

		alistPath := strings.Replace(embyRes["path"].(string), viper.GetString("server.mount-path"), "", 1)
		// url, err := alist.FetchAlistPathApi(viper.GetString("alist.url")+"/api/fs/get", alistPath, viper.GetString("alist.token"))
		// if err != nil {
		// 	log.Error(fmt.Sprintf("获取 Alist 地址失败。错误信息: %v", err))
		// 	proxy.ServeHTTP(c.Writer, c.Request)
		// 	return
		// }

		// 从Gin的请求上下文中获取请求头
		originalHeaders := make(map[string]string)
		for key, value := range c.Request.Header {
			if len(value) > 0 {
				originalHeaders[key] = value[0]
			}
		}

		url, err := alist.GetRedirectURL(viper.GetString("alist.url")+"/d"+alistPath, originalHeaders)

		log.Info("获取重定向链接： ")
		log.Info(url)

		c.Redirect(http.StatusFound, url)
	})

	if err := r.Run(":9096"); err != nil {
		panic(err)
	}
}
