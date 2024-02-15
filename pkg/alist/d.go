package alist

import (
	"fmt"
	"net/http"

	"github.com/spf13/viper"
)

// getRedirectURL尝试获取指定路径的重定向URL。
// 如果状态码是302，则返回重定向的URL，否则返回空字符串和错误。
func GetRedirectURL(modifiedPath string, originalHeaders map[string]string) (string, error) {
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse // 仅获取重定向URL，不跟随
		},
	}

	req, err := http.NewRequest("GET", modifiedPath, nil)
	if err != nil {
		return "", err // 创建请求失败
	}

	// 设置请求头
	for key, value := range originalHeaders {
		req.Header.Add(key, value)
	}

	req.Header.Add("Authorization", viper.GetString("alist.token"))

	resp, err := client.Do(req)
	if err != nil {
		return "", err // 发送请求失败
	}
	defer resp.Body.Close()

	// 检查是否是302状态码
	if resp.StatusCode == http.StatusFound { // 302
		// 获取重定向地址
		redirectedURL, err := resp.Location()
		if err != nil {
			return "", err // 获取重定向URL失败
		}
		return redirectedURL.String(), nil
	}

	return "", fmt.Errorf("no redirect or not a 302 status code") // 非302状态码
}
