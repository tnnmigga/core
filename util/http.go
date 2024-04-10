package util

import (
	"bytes"
	"io"
	"net/http"
	"strings"
)

func HttpPost(url string, data []byte, header ...map[string]string) ([]byte, error) {
	if len(header) == 0 {
		return HttpRequest("POST", url, data, nil)
	}
	return HttpRequest("POST", url, data, header[0])
}

func HttpGet(url string, header ...map[string]string) ([]byte, error) {
	if len(header) == 0 {
		return HttpRequest("GET", url, nil, nil)
	}
	return HttpRequest("GET", url, nil, header[0])
}

func HttpRequest(method, url string, data []byte, header map[string]string) ([]byte, error) {
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		url = "http://" + url
	}
	req, err := http.NewRequest(method, url, bytes.NewBuffer(data))
	if err != nil {
		return nil, err
	}
	// 设置请求头部信息
	for k, v := range header {
		req.Header.Set(k, v)
	}

	// 发起 HTTP 请求
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	// 读取响应体
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return body, nil
}
