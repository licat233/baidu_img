/**
version:4.0
*/
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
	verifyInput(fmt.Scanln(&input))
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
		s.NewGoroutine(s.QueryColly, url)
	}
	s.Group.Wait()
	s.QueryColly.Wait()
	s.DownloadColly.Wait()
	saveNum := s.fileNum()
	fmt.Printf("%d张%s的图片已下载完成！保存位置%s\n", saveNum, s.SearchKeyword, s.ImgDir)
}

func newSpider() *Spider_baidu_image {
	var keyword string
	fmt.Print("请输入搜索图片关键字:")
	verifyInput(fmt.Scanln(&keyword))

	var number int
	fmt.Print("请输入搜索数量:")
	verifyInput(fmt.Scanln(&number))
	headerParams := map[string]string{
		"Accept":                    "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.9",
		"Accept-Encoding":           "gzip, deflate, br",
		"Accept-Language":           "zh-CN,zh;q=0.9",
		"Cache-Control":             "max-age=0",
		"Connection":                "keep-alive",
		"DNT":                       "1",
		"Host":                      "image.baidu.com",
		"sec-ch-ua":                 "'Not A;Brand';v='99', 'Chromium';v='96', 'Google Chrome';v='96'",
		"sec-ch-ua-mobile":          "?0",
		"sec-ch-ua-platform":        "macOS",
		"Sec-Fetch-Dest":            "document",
		"Sec-Fetch-Mode":            "navigate",
		"Sec-Fetch-Site":            "none",
		"Sec-Fetch-User":            "?1",
		"Upgrade-Insecure-Requests": "1",
		"User-Agent":                "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/96.0.4664.110 Safari/537.36",
	}
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
	var saveImage func(resp *colly.Response, path string)
	saveImage = func(resp *colly.Response, path string) {
		if err := resp.Save(path); err != nil {
			saveImage(resp, path)
		}
	}
	var genDownSpider = func() *colly.Collector {
		downloadColly := colly.NewCollector(colly.UserAgent(headerParams["User-Agent"]), colly.Async(true))
		downloadColly.OnResponse(func(resp *colly.Response) {
			saveImage(resp, fmt.Sprintf("%s/%s.jpg", imgSaveDir, resp.FileName()))
		})
		downloadColly.OnError(func(resp *colly.Response, err error) {
			downloadColly.Visit(resp.Request.URL.String())
			// fmt.Printf("下载图片%s失败:%s\n", resp.Request.URL, err)
		})
		return downloadColly
	}
	DownSpider := genDownSpider()
	pageCount := (number + 60 - 1) / 60
	var genUrl = func(pageNum int) string {
		pn, rn := pageNum*60, 60
		if pageNum == pageCount {
			pn = number
			if num := number % 60; num != 0 {
				rn = num
			}
		}
		reqUrl := fmt.Sprintf("https://image.baidu.com/search/acjson?is=&adpicid=&height=&istype=2&tn=resultjson_com&gsm=1e&cl=2&expermode=&fp=result&cg=star&ie=utf-8&s=&face=0&ipn=rj&ct=201326592&z=&width=&pn=%d&rn=%d&word=%s&1642768249311=&logid=10967718936230564742&lm=-1&st=-1&ic=0&latest=&se=&queryWord=%s&fr=&oe=utf-8&copyright=&tab=&qc=&nc=1&nojc=&isAsync=&hd=&%s=", pn, rn, keyword, keyword, strconv.FormatInt(time.Now().Local().UnixMilli(), 10))
		return reqUrl
	}
	var queryUrls = []string{}
	for pageNum := 1; pageNum <= pageCount; pageNum++ {
		queryUrls = append(queryUrls, genUrl(pageNum))
	}
	collyHeaders := &http.Header{}
	for k, v := range headerParams {
		collyHeaders.Add(k, v)
	}
	var globalWG = &sync.WaitGroup{}
	var newGoroutine = func(c *colly.Collector, url string) {
		globalWG.Add(1)
		go func() {
			c.Visit(url)
			defer globalWG.Done()
		}()
	}
	var genQuerySpider = func() *colly.Collector {
		queryColly := colly.NewCollector(colly.Async(true))
		queryColly.OnRequest(func(r *colly.Request) {
			*r.Headers = *collyHeaders
		})
		queryColly.OnResponse(func(resp *colly.Response) {
			var respinfo RespJson
			e := json.Unmarshal(resp.Body, &respinfo)
			if e != nil {
				fmt.Println("json.Unmarshal failed")
				return
			}
			for _, v := range respinfo.Data {
				newGoroutine(DownSpider, v.Url)
			}
		})
		queryColly.OnError(func(resp *colly.Response, err error) {
			queryColly.Visit(resp.Request.URL.String())
			// fmt.Printf("请求链接%s失败:%s\n", resp.Request.URL, err)
		})
		return queryColly
	}
	QuerySpider := genQuerySpider()
	return &Spider_baidu_image{
		QueryUrls:      queryUrls,
		SearchKeyword:  keyword,
		SearchNumber:   number,
		Group:          globalWG,
		ImgDir:         imgSaveDir,
		DownloadColly:  DownSpider,
		QueryColly:     QuerySpider,
		GenQuerySpider: genQuerySpider,
		NewGoroutine:   newGoroutine,
	}
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
	SearchKeyword  string
	SearchNumber   int
	PageCount      int
	GenQuerySpider func() *colly.Collector
	DownloadColly  *colly.Collector
	QueryColly     *colly.Collector
	QueryUrls      []string
	NewGoroutine   func(c *colly.Collector, url string)
	Group          *sync.WaitGroup
	ImgDir         string
}

type RespJson struct {
	Data []struct {
		Url string `json:"thumbURL"`
	} `json:"data"`
}
