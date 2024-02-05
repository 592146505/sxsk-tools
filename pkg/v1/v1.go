package v1

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/jedib0t/go-pretty/v6/table"
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
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

func makeRequest(url string, param *QueryParam, wg *sync.WaitGroup) (*QueryResp, error) {
	defer wg.Done()

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
				break
			case tb == "用人单位":
				job.Department = value
				break
			case tb == "招考职位":
				job.JobName = value
				break
			case tb == "职位代码":
				job.Code = value
				break
			case tb == "招考人数":
				num, _ := strconv.Atoi(strings.TrimSuffix(value, "人"))
				job.RecruitsNumber = num
				break
			case tb == "学历":
				job.Educational = value
				break
			case tb == "报名人数":
				num, _ := strconv.Atoi(strings.TrimSuffix(value, "人"))
				job.ApplicantsNumber = num
				break
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
	var wg sync.WaitGroup
	errCodes := make(map[string]error)
	var jobs []*JobDetail
	for _, code := range codes {
		wg.Add(1)
		code := code
		go func() {
			data, err := makeRequest(url, &QueryParam{
				Code: code,
			}, &wg)
			// 失败
			if err != nil {
				errCodes[code] = err
			} else {
				job, err := parseHtml(data.Str)
				if err != nil {
					errCodes[code] = err
				} else {
					jobs = append(jobs, job)
				}
			}
		}()
	}
	wg.Wait()

	if len(jobs) > 0 {
		// 排序
		sort.Slice(jobs, func(i, j int) bool {
			return jobs[i].GetRatio() > jobs[j].GetRatio()
		})
		t := table.Table{}
		header := table.Row{"职位代码", "地市", "用人单位", "招考职位", "招考人数", "学历", "报名人数", "比值"}
		t.AppendHeader(header)
		for _, job := range jobs {
			row := table.Row{job.Code, job.City, job.Department, job.JobName, job.RecruitsNumber, job.Educational, job.ApplicantsNumber, fmt.Sprintf("%.2f%%", job.GetRatio()*100)}
			t.AppendRow(row)
		}
		fmt.Println(t.Render())
	}

	if len(errCodes) > 0 {
		t := table.Table{}
		header := table.Row{"职位代码", "错误信息"}
		t.AppendHeader(header)
		for code, err := range errCodes {
			row := table.Row{code, err}
			t.AppendRow(row)
		}
		fmt.Println(t.Render())
	}

}
