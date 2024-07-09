package emby

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
)

func GetItemPathInfo(c *gin.Context) (itemInfoUri string, itemId string, etag string, mediaSourceId string, apiKey string) {
	embyHost := viper.GetString("emby.url")
	embyApiKey := viper.GetString("emby.apikey")
	regex := regexp.MustCompile("[A-Za-z0-9]+")

	// 从URI中解析itemId，移除"emby"和"Sync"，以及所有连字符"-"。
	pathParts := regex.FindAllString(strings.ReplaceAll(strings.ReplaceAll(c.Request.RequestURI, "emby", ""), "Sync", ""), -1)
	if len(pathParts) > 1 {
		itemId = pathParts[1]
	}

	values := c.Request.URL.Query()
	if values.Get("MediaSourceId") != "" {
		mediaSourceId = values.Get("MediaSourceId")
	} else if values.Get("mediaSourceId") != "" {
		mediaSourceId = values.Get("mediaSourceId")
	}
	etag = values.Get("Tag")
	apiKey = values.Get("X-Emby-Token")
	if apiKey == "" {
		apiKey = values.Get("api_key")
	}
	if apiKey == "" {
		apiKey = embyApiKey
	}

	// Construct the itemInfoUri based on the URI and parameters
	if strings.Contains(c.Request.RequestURI, "JobItems") {
		itemInfoUri = embyHost + "/Sync/JobItems?api_key=" + apiKey
	} else {
		if mediaSourceId != "" {
			newMediaSourceId := mediaSourceId
			if strings.HasPrefix(mediaSourceId, "mediasource_") {
				newMediaSourceId = strings.Replace(mediaSourceId, "mediasource_", "", 1)
			}

			itemInfoUri = embyHost + "/Items?Ids=" + newMediaSourceId + "&Fields=Path,MediaSources&Limit=1&api_key=" + apiKey
		} else {
			itemInfoUri = embyHost + "/Items?Ids=" + itemId + "&Fields=Path,MediaSources&Limit=1&api_key=" + apiKey
		}
	}

	return itemInfoUri, itemId, etag, mediaSourceId, apiKey
}

func GetEmbyItems(itemInfoUri string, itemId string, etag string, mediaSourceId string, apiKey string) (map[string]interface{}, error) {
	rvt := map[string]interface{}{
		"message":  "success",
		"protocol": "File",
		"path":     "",
	}

	client := &http.Client{}
	req, err := http.NewRequest("GET", itemInfoUri, nil)
	if err != nil {
		return nil, fmt.Errorf("error: emby_api create request failed, %v", err)
	}
	req.Header.Set("Content-Type", "application/json;charset=utf-8")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error: emby_api fetch mediaItemInfo failed, %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		var result map[string]interface{}
		err := json.Unmarshal(bodyBytes, &result)
		if err != nil {
			return nil, fmt.Errorf("error: emby_api response json unmarshal failed, %v", err)
		}

		items, ok := result["Items"].([]interface{})
		if !ok {
			return nil, fmt.Errorf("error: emby_api invalid items format")
		}

		if itemInfoUri[len(itemInfoUri)-9:] == "JobItems" {
			for _, item := range items {
				jobItem := item.(map[string]interface{})
				if jobItem["Id"] == itemId && jobItem["MediaSource"] != nil {
					mediaSource := jobItem["MediaSource"].(map[string]interface{})
					rvt["protocol"] = mediaSource["Protocol"]
					rvt["path"] = mediaSource["Path"]
					return rvt, nil
				}
			}
			rvt["message"] = "error: emby_api /Sync/JobItems response is null"
		} else {
			// Handle case where "MediaType": "Photo"...
			if len(items) > 0 {
				item := items[0].(map[string]interface{})
				rvt["path"] = item["Path"].(string)

				// Parse MediaSources if available
				mediaSources, exists := item["MediaSources"].([]interface{})
				if exists && len(mediaSources) > 0 {
					var mediaSource map[string]interface{}
					for _, source := range mediaSources {
						ms := source.(map[string]interface{})
						if etag != "" && ms["etag"].(string) == etag {
							mediaSource = ms
							break
						}
						if mediaSourceId != "" && ms["Id"].(string) == mediaSourceId {
							mediaSource = ms
							break
						}
					}
					if mediaSource == nil {
						mediaSource = mediaSources[0].(map[string]interface{})
					}
					rvt["protocol"] = mediaSource["Protocol"]
					rvt["path"] = mediaSource["Path"]
				}
				// Decode .strm file path if necessary
				if rvt["path"].(string)[len(rvt["path"].(string))-5:] == ".strm" {
					decodedPath, err := url.QueryUnescape(rvt["path"].(string))
					if err == nil {
						rvt["path"] = decodedPath
					}
				}
			} else {
				rvt["message"] = "error: emby_api /Items response is null"
			}
		}
	} else {
		rvt["message"] = fmt.Sprintf("error: emby_api %d %s", resp.StatusCode, http.StatusText(resp.StatusCode))
	}
	return rvt, nil
}
