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

	fmt.Println("path:", path)
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
	path := r.URL.String()
	method := r.Method
	//cookie := r.Cookies()
	data, err := readFile(f.dir + path + ".json")
	if err != nil {
		w.Write([]byte(err.Error()))
	}

	end := new(EndPoint)
	err = json.Unmarshal(data, end)
	if err != nil {
		w.Write([]byte(err.Error()))
	}

	context, ok := end.Method[method]
	if !ok {
		w.Write([]byte("方法不存在"))
	}

	err = checkHeader(r, context.Header)
	fmt.Println(err)
	//checkQuery()
	//checkReq()
	writeResp(w, context.RespHeader, context.Resp)
}

func checkHeader(r *http.Request, rules map[string][]string) error {
	headers := r.Header
	for k, rule := range rules {
		vLen := len(rule)
		hkLen := len(headers[k])
		for i := 0; i < vLen; i++ {
			if rule[i] != "" && i >= hkLen {
				return errors.New("缺少 headers:" + k + "[" + fmt.Sprint(i) + "]")
			}
			reg := regexp.MustCompile(rule[i])
			ok := reg.Match([]byte(headers[k][i]))
			if !ok {
				return errors.New("headers:" + k + "[" + fmt.Sprint(i) + "]无法通过校验,规则 -> regexp[" + rule[i] + "]")
			}
		}
	}
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
	Method map[string]EndContext
}

// EndContext 接口请求内容
type EndContext struct {
	Header     map[string][]string
	RespHeader map[string][]string `json:"resp_header"`
	Query      map[string][]string
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
	case string:
		return "string"
	case int8, int16, int, int32, int64:
		return "int"
	case float32, float64:
		return "float"
	case map[string]interface{}:
		return "map[string]interface{}"
	case []interface{}:
		return "[]interface{}"
	default:
		return "string"
	}
}

// 检查参数的规则
func checkRuleRegex() {

}
