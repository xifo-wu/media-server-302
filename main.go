package main

import (
	"encoding/json"
	"fmt"
	"log"
	"media-server-302/pkg/alist"
	"media-server-302/pkg/config"
	"media-server-302/pkg/emby"
	"media-server-302/pkg/logger"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/patrickmn/go-cache"
	"github.com/spf13/viper"
)

func convertToLinuxPath(windowsPath string) string {
	// 将所有的反斜杠转换成正斜杠
	linuxPath := strings.ReplaceAll(windowsPath, "\\", "/")
	return linuxPath
}

func ensureLeadingSlash(alistPath string) string {
	if !strings.HasPrefix(alistPath, "/") {
		alistPath = "/" + alistPath // 不是以 / 开头，加上 /
	}

	alistPath = convertToLinuxPath(alistPath)
	return alistPath
}

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
	log.Info("MEDIA-SERVER-302")

	goCache := cache.New(1*time.Minute, 3*time.Minute)

	embyURL := viper.GetString("emby.url")
	url, _ := url.Parse(embyURL)

	proxy := httputil.NewSingleHostReverseProxy(url)

	r.Any("/*actions", func(c *gin.Context) {
		response, skip := ProxyPlaybackInfo(c, proxy)
		if !skip {
			c.JSON(http.StatusOK, response)
			return
		}

		currentURI := c.Request.RequestURI
		userAgent := strings.ToLower(c.Request.Header.Get("User-Agent"))
		cacheKey := RemoveQueryParams(currentURI) + userAgent

		if cacheLink, found := goCache.Get(cacheKey); found {
			log.Info("命中缓存")
			c.Redirect(302, cacheLink.(string))
			return
		}

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

		if !strings.HasPrefix(embyRes["path"].(string), viper.GetString("server.mount-path")) {
			proxy.ServeHTTP(c.Writer, c.Request)
			return
		}

		log.Info("Emby 原地址：" + embyRes["path"].(string))
		alistPath := strings.Replace(embyRes["path"].(string), viper.GetString("server.mount-path"), "", 1)
		alistPath = ensureLeadingSlash(alistPath)

		sign := alist.Sign(alistPath, 0)

		fullPath := "/d" + alistPath + "?sign=" + sign
		alistFullUrl := viper.GetString("alist.url") + fullPath

		// 如果是 infuse 走 alist 公网地址
		if strings.Contains(userAgent, "infuse") {
			// 设置公网地址
			if viper.GetString("alist.public-url") != "" {
				alistFullUrl = viper.GetString("alist.public-url") + fullPath
				log.Info("Alist 链接：" + alistFullUrl)
			}

			goCache.Set(cacheKey, alistFullUrl, cache.DefaultExpiration)
			c.Redirect(http.StatusFound, alistFullUrl)
			return
		}

		log.Info("Alist 链接：" + alistFullUrl)
		// 其他客户端继续走老的

		// 从Gin的请求上下文中获取请求头
		originalHeaders := make(map[string]string)
		for key, value := range c.Request.Header {
			if len(value) > 0 {
				originalHeaders[key] = value[0]
			}
		}

		url, err := alist.GetRedirectURL(alistFullUrl, originalHeaders)
		if err != nil {
			log.Error(fmt.Sprintf("获取 Alist 地址失败。错误信息: %v", err))
			proxy.ServeHTTP(c.Writer, c.Request)
			return
		}

		goCache.Set(cacheKey, url, cache.DefaultExpiration)
		c.Redirect(http.StatusFound, url)
	})

	if err := r.Run(":9096"); err != nil {
		panic(err)
	}
}

func RemoveQueryParams(originalURL string) string {
	parsedURL, err := url.Parse(originalURL)
	if err != nil {
		return originalURL
	}
	parsedURL.RawQuery = ""
	return parsedURL.String()
}

func ProxyPlaybackInfo(c *gin.Context, proxy *httputil.ReverseProxy) (response map[string]any, skip bool) {
	currentURI := c.Request.RequestURI

	re := regexp.MustCompile(`/[Ii]tems/(\S+)/PlaybackInfo`)
	matches := re.FindStringSubmatch(currentURI)
	if len(matches) < 1 {
		return nil, true
	}

	// 创建记录器来存储响应内容
	recorder := httptest.NewRecorder()

	// 代理请求
	proxy.ServeHTTP(recorder, c.Request)

	// 处理代理返回结果
	err := json.Unmarshal(recorder.Body.Bytes(), &response)
	if err != nil {
		return response, true
	}

	// 获取代理返回的响应头
	for key, values := range recorder.Header() {
		for _, value := range values {
			if key == "Content-Length" {
				continue
			}
			c.Writer.Header().Set(key, value)
		}
	}

	mediaSources := response["MediaSources"].([]interface{})
	for _, mediaSource := range mediaSources {
		ms := mediaSource.(map[string]interface{})

		isCloud := hitReplacePath(ms["Path"].(string))
		if !isCloud {
			log.Println("跳过：不是云盘文件")
			continue
		}

		// DEBUG 用
		ms["XOriginDirectStreamUrl"] = ms["DirectStreamUrl"]
		ms["SupportsDirectPlay"] = true
		ms["SupportsTranscoding"] = false
		ms["SupportsDirectStream"] = true

		delete(ms, "TranscodingUrl")
		delete(ms, "TranscodingSubProtocol")
		delete(ms, "TranscodingContainer")

		isInfiniteStream := ms["IsInfiniteStream"].(bool)
		localtionPath := "stream"
		if isInfiniteStream {
			localtionPath = "master"
		}

		fileExt := ms["Container"]
		if isInfiniteStream && (ms["Container"] == "" || ms["Container"] == "hls") {
			fileExt = "m3u8"
		}

		streamPart := fmt.Sprintf("%s.%s", localtionPath, fileExt)

		replacePath := strings.ReplaceAll(replaceIgnoreCase(currentURI, "/items", "/videos"), "PlaybackInfo", streamPart)

		parsedURL, _ := url.Parse(replacePath)
		params := parsedURL.Query()
		params.Set("MediaSourceId", ms["Id"].(string))
		params.Set("PlaySessionId", response["PlaySessionId"].(string))
		params.Set("Static", "true")
		parsedURL.RawQuery = params.Encode()
		ms["DirectStreamUrl"] = parsedURL.String()
	}

	response["302"] = "true"

	return response, false
}

func hitReplacePath(path string) bool {
	p := viper.GetString("server.mount-path")

	return strings.HasPrefix(path, p)
}

func replaceIgnoreCase(input string, old string, new string) string {
	re := regexp.MustCompile("(?i)" + regexp.QuoteMeta(old))
	return re.ReplaceAllString(input, new)
}
