package common

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"
)

func PostUrl(url string, data interface{}, headers map[string]string) (string, error) {
	bodyByte, err := json.Marshal(data)
	if err != nil {
		return "", nil
	}
	reader := bytes.NewReader(bodyByte)
	request, err := http.NewRequest("POST", url, reader)
	if err != nil {
		return "", nil
	}

	request.Header.Set("Content-type", "application/json;charset=UTF-8")
	if headers != nil && len(headers) != 0 {
		for key := range headers {
			request.Header.Set(key, headers[key])
		}
	}
	client := http.Client{}
	response, err := client.Do(request)
	if err != nil {
		return "", err
	}

	result, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return "", nil
	}
	defer response.Body.Close()
	return string(result), nil
}

func PostUrlAndCode(url string, data interface{}, headers map[string]string) (string, int, error) {
	bodyByte, err := json.Marshal(data)
	if err != nil {
		return "", 500, nil
	}
	//fmt.Println("ThirdPart Post:", string(bodyByte))
	reader := bytes.NewReader(bodyByte)
	request, err := http.NewRequest("POST", url, reader)
	if err != nil {
		return "", 500, nil
	}

	request.Header.Set("Content-type", "application/json;charset=UTF-8")
	//request.Header.Set("Content-type", "application/json")
	if headers != nil && len(headers) != 0 {
		for key := range headers {
			request.Header.Set(key, headers[key])
		}
	}
	client := http.Client{}
	response, err := client.Do(request)
	if err != nil {
		return "", 500, err
	}

	result, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return "", response.StatusCode, err
	}
	defer response.Body.Close()
	return string(result), response.StatusCode, nil
}

func GetUrlForSingle(url string) (string, error) {
	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", nil
	}

	request.Header.Set("Content-type", "application/json;charset=UTF-8")
	client := http.Client{}
	response, err := client.Do(request)
	if err != nil {
		fmt.Println("client.Do Err:", err)
		return "", nil
	}
	defer response.Body.Close()

	result, err := ioutil.ReadAll(response.Body)
	if err != nil {
		fmt.Println("ReadAll Err:", err)
		return "", nil
	}

	return string(result), nil
}

func GetUrl(domain string, query, headers map[string]string) (string, error) {
	client := &http.Client{
		Timeout: time.Second * 8,
	}
	req, err := http.NewRequest("GET", domain, nil)
	if err != nil {
		return "", err
	}

	q := url.Values{}
	if query != nil && len(query) != 0 {
		for key := range query {
			q.Add(key, query[key])
		}
	}

	req.Header.Set("Content-type", "application/json;charset=UTF-8")
	req.Header.Set("Accepts", "application/json")
	if headers != nil && len(headers) != 0 {
		for key := range headers {
			req.Header.Set(key, headers[key])
		}
	}
	req.URL.RawQuery = q.Encode()
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	respBody, _ := ioutil.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return "", errors.New(string(respBody))
	}

	return string(respBody), nil
}

func GetUrlAndCode(domain string, query, headers map[string]string) (string, int, error) {
	client := &http.Client{
		Timeout: time.Second * 10,
	}
	req, err := http.NewRequest("GET", domain, nil)
	if err != nil {
		return "", 500, err
	}

	q := url.Values{}
	if query != nil && len(query) != 0 {
		for key := range query {
			q.Add(key, query[key])
		}
	}

	req.Header.Set("Content-type", "application/json;charset=UTF-8")
	req.Header.Set("Accepts", "application/json")
	if headers != nil && len(headers) != 0 {
		for key := range headers {
			req.Header.Set(key, headers[key])
		}
	}
	req.URL.RawQuery = q.Encode()
	resp, err := client.Do(req)
	if err != nil {
		return "", 500, err
	}
	defer resp.Body.Close()

	respBody, _ := ioutil.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return "", resp.StatusCode, errors.New(string(respBody))
	}

	return string(respBody), resp.StatusCode, nil
}
