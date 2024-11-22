package v1

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/jedib0t/go-pretty/v6/table"
)

var client *http.Client
var once sync.Once

func getHTTPClient() *http.Client {
	once.Do(func() {
		client = &http.Client{
			Timeout: time.Second * 10,
			Transport: &http.Transport{
				MaxConnsPerHost: 5,
			},
		}
	})
	return client
}

func makeRequest(url string, param *QueryParam) (*QueryResp, error) {

	req, err := http.NewRequest("POST", url, strings.NewReader(param.ToReqParam().Encode()))
	if err != nil {
		return nil, err
	}

	// 添加自定义的HTTP请求头
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("Accept", "application/json")
	req.Header.Add("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/121.0.0.0 Safari/537.36")
	req.Header.Add("Origin", "https://sn.huatu.com")
	req.Header.Add("Referer", "https://sn.huatu.com/zt/2024skbmrscx/")

	resp, err := getHTTPClient().Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, err
	}

	// 读取响应体
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// 创建一个MyData对象
	var data QueryResp

	// 使用json.Unmarshal将JSON数据反序列化到对象中
	err = json.Unmarshal(body, &data)
	if err != nil {
		return nil, err
	}
	return &data, nil
}

func parseHtml(html string) (*JobDetail, error) {
	html = "<html><body><table>" + html + "</table></body></html>"
	reader, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return nil, err
	}
	var job *JobDetail
	reader.Find("td").Each(func(i int, selection *goquery.Selection) {
		attr, exists := selection.Attr("attr")
		if exists {
			if job == nil {
				job = &JobDetail{}
			}
			value := selection.Text()
			tb := strings.TrimSuffix(attr, "：")
			switch {
			case tb == "地市":
				job.City = value
			case tb == "用人单位":
				job.Department = value
			case tb == "招考职位":
				job.JobName = value
			case tb == "职位代码":
				job.Code = value
			case tb == "招考人数":
				num, _ := strconv.Atoi(strings.TrimSuffix(value, "人"))
				job.RecruitsNumber = num
			case tb == "学历":
				job.Educational = value
			case tb == "报名人数":
				num, _ := strconv.Atoi(strings.TrimSuffix(value, "人"))
				job.ApplicantsNumber = num
			}
		}
	})
	if job == nil {
		return nil, errors.New("无效html")
	}
	return job, nil
}

func Exec(codes []string) {
	url := "https://sn.huatu.com/zt/2024skbmrscx/app/executor.php"
	relustChan := make(chan *JobDetail, len(codes))
	errChan := make(chan error, len(codes))

	// 使用 WaitGroup 来等待所有 goroutine 完成
	var wg sync.WaitGroup

	// 限制并发数为5
	sem := make(chan struct{}, 5)

	for _, code := range codes {
		wg.Add(1)
		go func(jobCode string) {
			defer wg.Done()

			// 获取信号量
			sem <- struct{}{}
			defer func() { <-sem }()

			data, err := makeRequest(url, &QueryParam{
				Code: jobCode,
			})
			if err != nil {
				errChan <- fmt.Errorf("请求失败: %v", err)
				return
			}
			job, err := parseHtml(data.Str)
			if err != nil {
				errChan <- fmt.Errorf("解析失败: %v", err)
			} else {
				relustChan <- job
			}
		}(code)
	}

	// 等待所有请求完成后再关闭通道
	go func() {
		// 这里使用goroutine等待是因为:
		// 1. 主线程需要立即开始从channel中读取数据,如果在主线程中等待会导致死锁
		// 2. 所有worker goroutine完成后才能关闭channel,否则可能导致panic
		// 3. 使用goroutine可以让主线程继续执行后续的结果处理逻辑
		wg.Wait()
		close(relustChan)
		close(errChan)
	}()

	errCodes := make(map[string]error)
	var jobs []*JobDetail

	// 收集结果
	for job := range relustChan {
		jobs = append(jobs, job)
	}
	for err := range errChan {
		errCodes[codes[len(errCodes)]] = err
	}

	// 输出结果
	if len(jobs) > 0 {
		// 排序
		sort.Slice(jobs, func(i, j int) bool {
			if jobs[i].GetRatio() == jobs[j].GetRatio() {
				return strings.Compare(jobs[i].Code, jobs[j].Code) < 0
			}
			return jobs[i].GetRatio() > jobs[j].GetRatio()
		})

		t := table.Table{}
		t.AppendHeader(table.Row{"职位代码", "地市", "用人单位", "招考职位", "招考人数", "学历", "报名人数", "比值"})

		for _, job := range jobs {
			t.AppendRow(table.Row{
				job.Code,
				job.City,
				job.Department,
				job.JobName,
				job.RecruitsNumber,
				job.Educational,
				job.ApplicantsNumber,
				fmt.Sprintf("%.2f%%", job.GetRatio()*100),
			})
		}
		fmt.Println(t.Render())
	}

	if len(errCodes) > 0 {
		t := table.Table{}
		t.AppendHeader(table.Row{"职位代码", "错误信息"})
		for code, err := range errCodes {
			t.AppendRow(table.Row{code, err})
		}
		fmt.Println(t.Render())
	}
}
