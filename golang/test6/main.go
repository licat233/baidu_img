package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gocolly/colly/v2"
)

var (
	globalRWMux = &sync.RWMutex{}   //全局读写锁，用于管理tcp数量
	globalWG    = &sync.WaitGroup{} //全局等待组，用于管理并发
)

func main() {
	server()
}
func server() {
	newSpider().run()
	overAlert()
}

func overAlert() {
	var input string
	fmt.Print("本次爬取结束！是否新的爬取？(y/n)?")
	fmt.Scanln(&input)
	if input == "n" {
		return
	} else if input == "y" {
		server()
	} else {
		overAlert()
	}
}

func (s *Spider_baidu_image) run() {
	defer func(start time.Time) {
		terminal := time.Since(start)
		fmt.Println("耗时:", terminal)
	}(time.Now())
	for _, url := range s.QueryUrls {
		s.newGoroutine(s.QueryColly, url)
	}
	globalWG.Wait()
	s.DownloadColly.Wait()
	if saveNum := s.fileNum(); saveNum != 0 {
		fmt.Printf("【%d】张《%s》的图片已下载完成！\n保存位置%s\n", saveNum, s.SearchKeyword, s.ImgDir)
	} else {
		fmt.Println("失败！！一张图片都没有下载")
	}
}

func newSpider() *Spider_baidu_image {
	//用户设置
	var keyword string
	fmt.Print("请输入搜索图片关键字:")
	fmt.Scanln(&keyword)
	keyword = strings.Trim(keyword, " ")
	var number int
	fmt.Print("请输入搜索数量:")
	fmt.Scanln(&number)
	//检测默认目录是否存在
	CurrentPath, _ := os.Getwd()
	imgSaveDir := fmt.Sprintf("%s/%s", CurrentPath, keyword)
	var dirExist = func() bool {
		dir, err := os.Stat(imgSaveDir)
		if err != nil {
			return false
		}
		return dir.IsDir()
	}
	var verifyDir func()
	verifyDir = func() {
		if dirExist() {
			fmt.Printf("文件夹'%s'已经存在，是否使用该文件夹保存图片？(y/n):", imgSaveDir)
			var answer string
			verifyInput(fmt.Scan(&answer))
			answer = strings.Trim(answer, " ")
			if answer == "y" || answer == "Y" || answer == "yes" {
				return
			} else if answer == "n" || answer == "N" || answer == "no" {
				fmt.Print("请设置新的文件夹名字:")
				var newDirName string
				fmt.Scan(&newDirName)
				imgSaveDir = fmt.Sprintf("%s/%s", CurrentPath, newDirName)
			}
			verifyDir()
			return
		}
		err := os.Mkdir(imgSaveDir, os.ModePerm)
		if err != nil {
			log.Fatalf("创建文件出错![%v]\n", err)
		}
	}
	verifyDir()
	//计算页数，生成请求链接
	pageCount := (number + 30 - 1) / 30
	var genUrl = func(pageNum int) string {
		pn, rn := pageNum*30, 30
		if pageNum == pageCount {
			pn = number
			if num := number % 30; num != 0 {
				rn = num
			}
		}
		reqUrl := fmt.Sprintf("https://image.baidu.com/search/acjson?is=&adpicid=&height=&istype=2&tn=resultjson_com&gsm=1e&cl=2&expermode=&fp=result&cg=star&ie=utf-8&s=&face=0&ipn=rj&ct=201326592&z=&width=&pn=%d&rn=%d&word=%s&logid=11011187488361796081&lm=-1&st=-1&ic=0&latest=&se=&queryWord=%s&fr=&oe=utf-8&copyright=&tab=&qc=&nc=1&nojc=&isAsync=&hd=&%s=", pn, rn, keyword, keyword, strconv.FormatInt(time.Now().Local().UnixMilli(), 10))
		return reqUrl
	}
	var queryUrls = []string{}
	for pageNum := 1; pageNum <= pageCount; pageNum++ {
		queryUrls = append(queryUrls, genUrl(pageNum))
	}

	UA := "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/96.0.4664.110 Safari/537.36"
	//设置请求头部
	headerParams := map[string]string{
		"Accept":                    "text/plain, */*; q=0.01",
		"Accept-Encoding":           "gzip, deflate, br",
		"Accept-Language":           "zh-CN,zh;q=0.9",
		"Cache-Control":             "max-age=0",
		"Connection":                "keep-alive",
		"DNT":                       "1",
		"Host":                      "image.baidu.com",
		"sec-ch-ua":                 "'Not A;Brand';v='99', 'Chromium';v='96', 'Google Chrome';v='96'",
		"sec-ch-ua-mobile":          "?0",
		"sec-ch-ua-platform":        "macOS",
		"Sec-Fetch-Dest":            "empty",
		"Sec-Fetch-Mode":            "cors",
		"Sec-Fetch-Site":            "same-origin",
		"Sec-Fetch-User":            "?1",
		"Upgrade-Insecure-Requests": "1",
		"User-Agent":                UA,
		"X-Requested-With":          "XMLHttpRequest",
	}
	collyHeaders := &http.Header{}
	for k, v := range headerParams {
		collyHeaders.Add(k, v)
	}

	downloadColly := colly.NewCollector(colly.UserAgent(UA), colly.Async(true))
	downloadColly.OnResponse(func(resp *colly.Response) {
		globalWG.Add(1)
		//异步保存图片
		go func(resp *colly.Response) {
			if err := resp.Save(fmt.Sprintf("%s/%d.jpg", imgSaveDir, time.Now().Local().UnixNano())); err != nil {
				fmt.Println("图片保存失败:", err.Error())
			}
			defer globalWG.Done()
		}(resp)
	})
	downloadColly.OnError(func(resp *colly.Response, err error) {
		fmt.Printf("下载图片%s失败:%s\n", resp.Request.URL, err)
	})

	s := &Spider_baidu_image{
		TcpNumRWMux:   &sync.RWMutex{},
		SearchKeyword: keyword,
		SearchNumber:  number,
		ImgDir:        imgSaveDir,
		TcpNum:        0,
		CollyHeaders:  collyHeaders,
		QueryUrls:     queryUrls,
		DownloadColly: downloadColly,
		QueryColly:    nil,
	}

	queryColly := colly.NewCollector(colly.Async(true))
	queryColly.OnRequest(func(r *colly.Request) {
		*r.Headers = *s.CollyHeaders
	})
	queryColly.OnResponse(func(resp *colly.Response) {
		var respinfo RespJson
		if err := json.Unmarshal(resp.Body, &respinfo); err != nil {
			fmt.Println("json.Unmarshal failed:", err)
			fmt.Println("request url:", resp.Request.URL.String())
			resp.Save(CurrentPath + "/body.txt")
			// fmt.Println("content:", string(resp.Body))
			return
		}
		for _, v := range respinfo.Data {
			s.newGoroutine(s.DownloadColly, v.Url)
		}
	})

	queryColly.OnError(func(resp *colly.Response, err error) {
		queryColly.Visit(resp.Request.URL.String())
		// fmt.Printf("请求链接%s失败:%s\n", resp.Request.URL, err)
	})
	s.QueryColly = queryColly
	return s
}

//并发函数,用于集中控制TCP数量，以及管理并发
func (s *Spider_baidu_image) newGoroutine(c *colly.Collector, url string) {
	//防止tcp连接数溢出,限制999个
	globalRWMux.RLock()
	for {
		if s.TcpNum <= 999 {
			break
		}
	}
	globalRWMux.RUnlock()

	globalWG.Add(1)
	go func(c *colly.Collector, url string) {
		s.TcpNumRWMux.Lock()
		s.TcpNum++
		s.TcpNumRWMux.Unlock()
		c.Visit(url)
		s.TcpNumRWMux.Lock()
		s.TcpNum--
		s.TcpNumRWMux.Unlock()
		defer globalWG.Done()
	}(c, url)
}

func (s *Spider_baidu_image) fileNum() (k int) {
	filepath.Walk(s.ImgDir, func(filename string, fi os.FileInfo, err error) error {
		k++
		return nil
	})
	return
}

func verifyInput(_ int, err error) {
	if err != nil {
		log.Fatalln("错误的输入")
	}
}

type Spider_baidu_image struct {
	TcpNumRWMux   *sync.RWMutex
	SearchKeyword string
	SearchNumber  int
	ImgDir        string
	TcpNum        int
	CollyHeaders  *http.Header
	QueryUrls     []string
	DownloadColly *colly.Collector
	QueryColly    *colly.Collector
}

type RespJson struct {
	Data []struct {
		Url string `json:"thumbURL"`
	} `json:"data"`
}
