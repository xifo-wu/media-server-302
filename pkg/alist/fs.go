package alist

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

func FetchAlistPathApi(alistApiPath, alistFilePath, alistToken string) (string, error) {
	alistRequestBody := map[string]interface{}{
		"path":     alistFilePath,
		"password": "",
	}

	jsonData, err := json.Marshal(alistRequestBody)
	if err != nil {
		return "", fmt.Errorf("error: could not encode request body: %w", err)
	}

	client := &http.Client{}
	req, err := http.NewRequest("POST", alistApiPath, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("error: could not create POST request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json;charset=utf-8")
	req.Header.Set("Authorization", alistToken)

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("error: alist_path_api fetchAlistFiled %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error: could not read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("error: alist_path_api %d %s", resp.StatusCode, http.StatusText(resp.StatusCode))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("error: could not decode response body: %w", err)
	}

	if result["code"].(float64) != float64(200) {
		return "", fmt.Errorf(result["message"].(string))
	}

	data, ok := result["data"].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("error: invalid data format")
	}

	rawUrl, ok := data["raw_url"].(string)
	if ok {
		return rawUrl, nil
	}

	if content, ok := data["content"].([]interface{}); ok {
		var fileNames []string
		for _, item := range content {
			if fileItem, ok := item.(map[string]interface{}); ok {
				if name, ok := fileItem["name"].(string); ok {
					fileNames = append(fileNames, name)
				}
			}
		}
		return strings.Join(fileNames, ","), nil
	}

	return "", fmt.Errorf("error: 未知错误")
}
