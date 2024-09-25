package common

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

func RpcRequest(rpcUrl, path string, options ...any) ([]byte, http.Header, error) {
	maxRetries := 1
	if len(options) > 0 {
		maxRetries = options[0].(int)
	}
	if maxRetries <= 0 {
		maxRetries = 1
	}

	var isRpc = true
	if len(options) > 1 {
		isRpc = options[1].(bool)
	}
	for i := 0; i < maxRetries; i++ {
		p, err := url.JoinPath(rpcUrl, path)
		if err != nil {
			return nil, nil, err
		}
		req, err := http.NewRequest("GET", p, nil)
		if err != nil {
			return nil, nil, err
		}
		if isRpc {
			req.Header.Set("Accept", "application/json")
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			if i == maxRetries-1 {
				return nil, nil, fmt.Errorf("RpcRequest: failed to fetch data after %d retries", maxRetries)
			}
			time.Sleep(time.Second)
			continue
		}
		defer resp.Body.Close()
		if resp.StatusCode != 200 {
			msgData, err := io.ReadAll(resp.Body)
			if err != nil {
				return nil, nil, fmt.Errorf("RpcRequest: failed to fetch data, statusCode: %v, error: %s", resp.StatusCode, err.Error())
			}
			return nil, nil, fmt.Errorf("RpcRequest: failed to fetch data, statusCode: %v, error: %s", resp.StatusCode, string(msgData))
		}
		data, err := io.ReadAll(resp.Body)
		if err != nil {
			if i == maxRetries-1 {
				return nil, nil, err
			}
			continue
		}
		return data, resp.Header, nil
	}
	return nil, nil, fmt.Errorf("RpcRequest: failed to fetch data after %d retries", maxRetries)
}

func HtmlRequest(htmlUrl, path string, retries ...int) ([]byte, error) {
	maxRetries := 1
	if len(retries) > 0 {
		maxRetries = retries[0]
	}
	if maxRetries <= 0 {
		maxRetries = 1
	}
	for i := 0; i < maxRetries; i++ {
		p, err := url.JoinPath(htmlUrl, path)
		if err != nil {
			return nil, err
		}
		req, err := http.NewRequest("GET", p, nil)
		if err != nil {
			return nil, err
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			if i == maxRetries-1 {
				return nil, fmt.Errorf("failed to fetch data after %d retries", maxRetries)
			}
			time.Sleep(time.Second)
			continue
		}
		defer resp.Body.Close()
		if resp.StatusCode != 200 {
			return nil, fmt.Errorf("failed to fetch data")
		}
		data, err := io.ReadAll(resp.Body)
		if err != nil {
			if i == maxRetries-1 {
				return nil, err
			}
			continue
		}
		return data, nil
	}
	return nil, fmt.Errorf("failed to fetch data after %d retries", maxRetries)
}
