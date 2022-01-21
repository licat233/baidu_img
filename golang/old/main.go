package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gocolly/colly/v2"
)

type Spider_baidu_image struct {
	RWMux         *sync.RWMutex
	SearchKeyword string
	SearchNumber  int
	PageCount     int
	ReqUrl        string
	ReqMethod     string
	ReqHeaders    map[string]string
	ReqParams     map[string]string
	CurrentPath   string
	ImgDirName    string
	ImgDir        string
	ImgUrls       []string
}

type RespJson struct {
	Data []struct {
		Url string `json:"thumbURL"`
	} `json:"data"`
}

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
	s.getAllUrls()
	s.getAllImgData()
}

func (s *Spider_baidu_image) getAllImgData() {
	saveNum := s.SearchNumber
	allCount := len(s.ImgUrls)
	if allCount == 0 {
		fmt.Println("没有爬取到图片")
		return
	}
	if allCount < s.SearchNumber {
		fmt.Printf("只爬取到%d张图片，没有达到%d张，是否下载？(y/n)", allCount, saveNum)
		var answer string
		fmt.Scanln(&answer)
		if answer == "n" {
			return
		} else if answer == "y" {
			saveNum = allCount
		} else {
			s.getAllImgData()
			return
		}
	}
	var wg sync.WaitGroup
	for k := 0; k < saveNum; k++ {
		wg.Add(1)
		go func(index int, url string) {
			c := colly.NewCollector()
			c.OnResponse(func(resp *colly.Response) {
				file, err := os.Create(fmt.Sprintf("%s/%d.jpg", s.ImgDir, index))
				if err != nil {
					fmt.Println("文件创建失败", err.Error())
					return
				}
				file.Write(resp.Body)
				defer file.Close()
			})
			c.OnError(func(_ *colly.Response, e error) {
				fmt.Printf("下载图片%s失败:%s\n", url, e)
			})
			c.Visit(url)
			defer wg.Done()
		}(k+1, s.ImgUrls[k])
	}
	wg.Wait()
	fmt.Printf("%d张%s的图片已下载完成！保存位置%s\n", saveNum, s.SearchKeyword, s.ImgDir)
}

func (s *Spider_baidu_image) getAllUrls() {
	var wg sync.WaitGroup
	for pageNum := 1; pageNum <= s.PageCount; pageNum++ {
		wg.Add(1)
		go func(pageNum int) {
			s.getSinglePageImgUrls(pageNum)
			defer wg.Done()
		}(pageNum)
	}
	wg.Wait()
}

func (s *Spider_baidu_image) getSinglePageImgUrls(pageNum int) {
	url := s.generateUrl(pageNum)
	c := colly.NewCollector()
	c.OnRequest(func(r *colly.Request) {
		for k, v := range s.ReqHeaders {
			r.Headers.Add(k, v)
		}
	})
	c.OnResponse(func(resp *colly.Response) {
		var respinfo RespJson
		e := json.Unmarshal(resp.Body, &respinfo)
		if e != nil {
			fmt.Println("json.Unmarshal failed")
			return
		}
		for _, v := range respinfo.Data {
			s.ImgUrls = append(s.ImgUrls, v.Url)
		}
	})
	c.OnError(func(_ *colly.Response, e error) {
		fmt.Printf("请求链接%s失败:%s\n", url, e)
	})
	c.Visit(url)
}

func (s *Spider_baidu_image) generateUrl(pageNum int) string {
	var nowtime = strconv.FormatInt(time.Now().Local().UnixMilli(), 10)
	s.RWMux.Lock()
	s.ReqParams[nowtime] = ""
	s.ReqParams["pn"] = strconv.Itoa(pageNum * 60)
	url := fmt.Sprintf("%s?%s=", s.ReqUrl, nowtime)
	for k, v := range s.ReqParams {
		url = fmt.Sprintf("%s&%s=%s", url, k, v)
	}
	s.RWMux.Unlock()
	return url
}

func newSpider() *Spider_baidu_image {
	params := map[string]string{
		"tn":        "resultjson_com",
		"logid":     "8697370931949901543",
		"ipn":       "rj",
		"ct":        "201326592",
		"is":        "",
		"fp":        "result",
		"fr":        "",
		"cg":        "star",
		"cl":        "2",
		"lm":        "-1",
		"ie":        "utf-8",
		"oe":        "utf-8",
		"adpicid":   "",
		"st":        "-1",
		"z":         "",
		"ic":        "0",
		"hd":        "",
		"latest":    "",
		"copyright": "",
		"s":         "",
		"se":        "",
		"tab":       "",
		"width":     "",
		"height":    "",
		"face":      "0",
		"istype":    "2",
		"qc":        "",
		"nc":        "1",
		"expermode": "",
		"nojc":      "",
		"isAsync":   "",
		"pn":        "60",
		"rn":        "60",
		"gsm":       "1e",
	}
	var keyword string
	fmt.Print("请输入搜索图片关键字:")
	fmt.Scanln(&keyword)
	params["word"] = keyword
	params["queryWord"] = keyword

	var number int
	fmt.Print("请输入搜索数量:")
	fmt.Scanln(&number)
	pageCount := (number + 60 - 1) / 60
	headers := map[string]string{
		"Accept":                    "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.9",
		"Accept-Encoding":           "gzip, deflate, br",
		"Accept-Language":           "zh-CN,zh;q=0.9",
		"Cache-Control":             "max-age=0",
		"Connection":                "keep-alive",
		"Cookie":                    "BDqhfp=%E5%88%98%E4%BA%A6%E8%8F%B2%26%260-10-1undefined%26%260%26%261; BAIDUID=B2AC247DBEADF4A0C6533A03FB41881B:FG=1; BIDUPSID=B2AC247DBEADF4A06C5FD0B41E7DCB3A; PSTM=1642571761; BDRCVFR[Fc9oatPmwxn]=aeXf-1x8UdYcs; H_PS_PSSID=35740_35104_35733_35489_34584_35491_35699_35688_35542_35325_26350_35746; BDRCVFR[X_XKQks0S63]=mk3SLVN4HKm; __yjs_duid=1_c15adeed5a9fa62c48e91fd3e40c53041642571763639; firstShowTip=1; indexPageSugList=%5B%22%E5%88%98%E4%BA%A6%E8%8F%B2%22%5D; cleanHistoryStatus=0; BDRCVFR[dG2JNJb_ajR]=mk3SLVN4HKm; delPer=0; PSINO=6; userFrom=null; ab_sr=1.0.1_ODZjZmQ4ZjUzOTViYjc4ZmM3OTFlYjJjZWRjNTM1NDcyZWQxNWFkZTgzMDFlNjM4Mjg2MWViODZmM2QyMmYyYThjZTYzYzI5ZmU2ZmU3NDQwOGJiMzBjYWE1YTI5Zjc4",
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
		"Referer":                   "https://image.baidu.com/search/index?tn=baiduimage&ipn=r&ct=201326592&cl=2&lm=-1&st=-1&fm=result&fr=&sf=1&fmq=1642695379008_R&pv=&ic=0&nc=1&z=&hd=&latest=&copyright=&se=1&showtab=0&fb=0&width=&height=&face=0&istype=2&dyTabStr=&ie=utf-8&sid=&word=%E5%88%98%E4%BA%A6%E8%8F%B2",
	}
	dir, _ := os.Getwd()
	s := &Spider_baidu_image{
		RWMux:         &sync.RWMutex{},
		SearchKeyword: keyword,
		SearchNumber:  number,
		PageCount:     pageCount,
		ReqUrl:        "https://image.baidu.com/search/acjson",
		ReqMethod:     "GET",
		ReqHeaders:    headers,
		ReqParams:     params,
		CurrentPath:   dir,
		ImgDirName:    keyword,
		ImgDir:        fmt.Sprintf("%s/%s", dir, keyword),
	}
	var dirExist = func() bool {
		dir, err := os.Stat(s.ImgDir)
		if err != nil {
			return false
		}
		return dir.IsDir()
	}
	var verifyDir func()
	verifyDir = func() {
		if dirExist() {
			fmt.Printf("文件夹'%s'已经存在，是否使用该文件夹保存图片？(y/n):", s.ImgDir)
			var answer string
			fmt.Scan(&answer)
			answer = strings.Trim(answer, " ")
			if answer == "y" || answer == "Y" || answer == "yes" {
				return
			} else if answer == "n" || answer == "N" || answer == "no" {
				fmt.Print("请设置新的文件夹名字:")
				fmt.Scan(&s.ImgDirName)
				s.ImgDir = fmt.Sprintf("%s/%s", s.CurrentPath, s.ImgDirName)
			}
			verifyDir()
			return
		}
		err := os.Mkdir(s.ImgDir, os.ModePerm)
		if err != nil {
			log.Fatalf("创建文件出错![%v]\n", err)
		}
	}
	verifyDir()
	return s
}
