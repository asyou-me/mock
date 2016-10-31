package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"
)

var (
	// 静态文件服务器
	fileHandler = http.FileServer(http.Dir("./"))
)

// Usage 启动时参数规则
func Usage() {
	fmt.Fprint(os.Stderr, "Usage of ", os.Args[0], ":\n")
	flag.PrintDefaults()
	fmt.Fprint(os.Stderr, "\n")
}

func main() {
	// 获取监听的端口
	httpAddr := flag.String("http", ":9090", "监听端口")
	// 获取存放接口的目录
	dir := flag.String("dir", "", "接口存放目录")
	flag.Parse()

	path := *dir
	// 当 path 为空时当前目录作为 path
	pathLen := len(path)
	if pathLen == 0 {
		path = "."
	}

	// 保证 path 为一个目录
	if path[:pathLen-1] == "/" {
		path = path[:pathLen-1]
	}

	// 业务处理句柄
	handler := &httpHandler{
		dir:   path,
		cache: map[string][]byte{},
	}

	// http 服务对象
	server := http.Server{
		Addr:        *httpAddr,
		Handler:     handler,
		ReadTimeout: 10 * time.Second,
	}
	server.ListenAndServe()
}

type httpHandler struct {
	// 接口数据保存目录,数据将以文件形式存在目录
	dir string
	// 接口数据缓存
	cache map[string][]byte
}

// ServeHTTP
func (f *httpHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	url := r.URL.String()
	suffix := getSuffix(url)
	switch suffix {
	case "":
		// 业务服务
		f.MockAPI(w, r)
	case "js", "css", "jpg", "png":
		// 文件服务
		fileHandler.ServeHTTP(w, r)
	}
}

// MockAPI 接口模拟函数
func (f *httpHandler) MockAPI(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	method := r.Method
	//cookie := r.Cookies()
	data, err := readFile(f.dir + path + ".json")
	if err != nil {
		w.Write([]byte(err.Error()))
		return
	}

	end := &EndPoint{}
	err = json.Unmarshal(data, end)
	if err != nil {
		fmt.Println("err:", err)
		w.Write([]byte(err.Error()))
		return
	}

	context, ok := end.Method[method]
	if !ok {
		w.WriteHeader(404)
		data, _ := readFile(f.dir + "404.json")
		w.Write(data)
		return
	}

	err = checkHeader(r, context.Header)
	if err != nil {
		writeErr(w, err)
		return
	}

	err = checkQuery(r, context.Query)
	if err != nil {
		writeErr(w, err)
		return
	}

	err = checkReq(r, context.Req)
	if err != nil {
		writeErr(w, err)
		return
	}

	writeResp(w, context.RespHeader, context.Resp)
}

func checkHeader(r *http.Request, rules map[string]string) error {
	headers := r.Header
	for k, rule := range rules {
		hkLen := len(headers[k])
		if rule != "" && hkLen < 1 {
			return errors.New("缺少 headers: " + k)
		}
		for i := 0; i < hkLen; i++ {
			reg := regexp.MustCompile("^" + rule + "$")
			ok := reg.Match([]byte(headers[k][i]))
			if !ok {
				return errors.New("headers: " + k + "[" + fmt.Sprint(i) + "]无法通过校验" /*,规则 -> regexp[" + "^" + rule[i] + "$" + "]"*/)
			}
		}
	}
	return nil
}

func checkQuery(r *http.Request, rules map[string]string) error {
	querys := r.URL.Query()
	for k, rule := range rules {
		vLen := len(rule)
		hkLen := len(querys[k])
		if rule != "" && hkLen < 1 {
			return errors.New("缺少 headers: " + k)
		}
		for i := 0; i < vLen; i++ {
			reg := regexp.MustCompile("^" + rule + "$")
			ok := reg.Match([]byte(querys[k][i]))
			if !ok {
				return errors.New("url.querys: " + k + "[" + fmt.Sprint(i) + "]无法通过校验" /*,规则 -> regexp[" + "^" + rule[i] + "$" + "]"*/)
			}
		}
	}
	return nil
}

func checkReq(r *http.Request, rules interface{}) error {
	if rules == nil {
		return nil
	}
	buf, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return err
	}

	switch getType(rules) {
	case "{}":
		obj := map[string]interface{}{}
		err = json.Unmarshal(buf, &obj)
		if err != nil {
			return errors.New("body 必须为json类型 ")
		}
		return compareStruct(obj, rules.(map[string]interface{}), "")
	case "[]":
		obj := []interface{}{}
		err = json.Unmarshal(buf, &obj)
		if err != nil {
			return errors.New("body 必须为json类型 ")
		}
		return compareSlice(obj, rules.([]interface{}), "")
	case "string":
		rule := rules.(string)
		index := strings.Index(rule, "|")
		if index == -1 {
			return errors.New("规则" + rule + "定义出错")
		}
		reg := regexp.MustCompile("^" + string(rule[:index]) + "$")
		ok := reg.Match(buf)
		if !ok {
			return errors.New("body 无法通过校验" /*,规则 -> regexp[" + "^" + rule[i] + "$" + "]"*/)
		}
	}
	return nil
}

func compareStruct(body map[string]interface{}, rules map[string]interface{}, path string) error {
	for k, v := range rules {
		switch getType(v) {
		case "{}":
			b, err := body[k].(map[string]interface{})
			if !err {
				return errors.New(path + k + " 必须为json类型 ")
			}
			return compareStruct(b, v.(map[string]interface{}), path+k)
		case "[]":
			b, err := body[k].([]interface{})
			if !err {
				return errors.New(path + k + " 必须为数组类型 ")
			}
			return compareSlice(b, v.([]interface{}), "")
		case "string":
			rule := v.(string)
			index := strings.Index(rule, "|")
			if index == -1 {
				return errors.New("规则" + rule + "定义出错")
			}
			reg := regexp.MustCompile("^" + rule[index+1:] + "$")
			ok := reg.Match([]byte(fmt.Sprint(body[k])))
			if !ok {
				return errors.New(path + k + " 无法通过校验" /*,规则 -> regexp[" + "^" + rule[i] + "$" + "]"*/)
			}
		}
	}
	return nil
}

func compareSlice(body []interface{}, rules []interface{}, path string) error {
	return nil
}

func compareEnd(body interface{}, rules interface{}, path string) error {

	return nil
}

// 写入结果
func writeResp(w http.ResponseWriter, respHeader map[string][]string, resp interface{}) {
	header := w.Header()
	for k, v := range respHeader {
		header.Add(k, v[0])
	}
	// 设置跨域请求
	header.Set("Access-Control-Allow-Origin", "*")
	data, err := json.Marshal(resp)
	if err == nil {
		header.Set("Content-Type", "application/json;charset=utf-8")
		w.Write(data)
	} else {
		header.Set("Content-Type", "text/html")
		w.Write([]byte(fmt.Sprint(data)))
	}
}

// 写入错误
func writeErr(w http.ResponseWriter, err error) {
	header := w.Header()
	// 设置跨域请求
	header.Set("Access-Control-Allow-Origin", "*")
	header.Set("Content-Type", "application/json;charset=utf-8")
	w.Write([]byte(`{"msg":"` + err.Error() + `"}`))
}

// 获取字符串后缀,后缀长度不能超过8
func getSuffix(url string) string {
	urlBytes := []byte(url)
	urlLen := len(urlBytes)
	var max int
	if urlLen < 10 {
		max = urlLen
	} else {
		max = 10
	}
	for i := 1; i < max; i++ {
		index := urlLen - i
		if urlBytes[index] == 46 {
			return url[index+1:]
		}
	}
	return ""
}

func readFile(path string) ([]byte, error) {
	fi, err := os.Open(path)
	if err != nil {
		return nil, errors.New("not found")
	}
	defer fi.Close()
	fd, err := ioutil.ReadAll(fi)
	return fd, nil
}

// EndPoint 接口后端
type EndPoint struct {
	Method map[string]EndContext `json:"Method"`
}

// EndContext 接口请求内容
type EndContext struct {
	Header     map[string]string
	Query      map[string]string
	RespHeader map[string][]string `json:"resp_header"`
	Req        interface{}
	Resp       interface{}
}

// Rule 参数规则
type Rule struct {
	Type  string
	Regex string
}

// 获取的类型
func getType(data interface{}) string {
	switch data.(type) {
	case map[string]interface{}:
		return "{}"
	case []interface{}:
		return "[]"
	case string:
		return "string"
	case int8, int16, int, int32, int64:
		return "int"
	case float32, float64:
		return "float"
	default:
		return "string"
	}
}
