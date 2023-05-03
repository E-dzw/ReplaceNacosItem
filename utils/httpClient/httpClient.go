package utils

import (
	"bytes"
	"io/ioutil"
	"log"
	"mime/multipart"
	"net/http"
	"strings"
)

type methodtype int

const (
	GET methodtype = iota
	POST
	PUT
	DELETE
	HEAD
)

var METHODTYPES = [...]string{GET: "GET", POST: "POST", PUT: "PUT", DELETE: "DELETE", HEAD: "HEAD"}

func SendHttpRequest(method string, urlString string, reqBody string, urlParams map[string]string) string {
	methodupper := strings.ToUpper(method)
	flag := false
	for _, value := range METHODTYPES {
		if value == methodupper {
			flag = true
			break
		}
	}

	if flag {
		log.Printf("request methods not support")
	}

	bodyReader := strings.NewReader(reqBody)

	req, err := http.NewRequest(methodupper, urlString, bodyReader)
	if err != nil {
		log.Fatal("http request failed: ", err)
	}

	req.Header.Add("Content-Type", "application/json")

	//添加url参数
	q := req.URL.Query()
	if len(urlParams) != 0 {
		for key, value := range urlParams {
			q.Set(key, value)
		}
	}

	req.URL.RawQuery = q.Encode()

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Fatal("http request failed: ", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Fatal("http request failed: ", err)
	}

	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal("http request failed: ", err)
	}

	return string(respBody)
}

// POST 表单请求
func HttpPostWithFormData(urlString string, postData map[string]string) string {
	body := new(bytes.Buffer)
	w := multipart.NewWriter(body)
	for k, v := range postData {
		w.WriteField(k, v)
	}
	w.Close()
	req, err := http.NewRequest("POST", urlString, body)
	if err != nil {
		log.Fatal("http request failed: ", err)
	}
	req.Header.Add("Content-Type", w.FormDataContentType())
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Fatal("http request failed: ", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Fatal("http request failed: ", err)
	}

	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal("http request failed: ", err)
	}

	return string(respBody)
}
