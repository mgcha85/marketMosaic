package crawler

import (
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

// RealtimeData 실시간 크롤링 결과
type RealtimeData struct {
	Type      string                   `json:"type"`
	Title     string                   `json:"title"`
	URL       string                   `json:"url"`
	Count     int                      `json:"count"`
	Items     []map[string]interface{} `json:"items"`
	CrawledAt time.Time                `json:"crawled_at"`
}

// RealtimeCrawler 실시간 크롤러
type RealtimeCrawler struct {
	client *http.Client
}

// NewRealtimeCrawler 새 실시간 크롤러 생성
func NewRealtimeCrawler() *RealtimeCrawler {
	return &RealtimeCrawler{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// 테마 보기 탭 URL 정의
var ThemeListURLs = map[string]struct {
	URL   string
	Title string
}{
	"all":       {baseURL + "/?view=themeList", "전체 테마"},
	"rising":    {baseURL + "/?view=themeList&type=changeRateDesc", "상승중인 테마"},
	"falling":   {baseURL + "/?view=themeList&type=changeRateAsc", "하락중인 테마"},
	"expected":  {baseURL + "/?view=themeList&type=expectRateDesc", "기대수익률 높은 테마"},
	"hot":       {baseURL + "/?view=themeList&type=neglectRateHot", "현재 핫한 테마"},
	"neglected": {baseURL + "/?view=themeList&type=neglectRateDesc", "많이 소외된 테마"},
}

// 테마별 종목 탭 URL 정의
var StockListURLs = map[string]struct {
	URL   string
	Title string
}{
	"rising":        {baseURL + "/?view=stockList&type=changeRateDesc", "상승중인 종목"},
	"falling":       {baseURL + "/?view=stockList&type=changeRateAsc", "하락중인 종목"},
	"neglected":     {baseURL + "/?view=stockList&type=neglectRateDesc", "많이 소외된 종목"},
	"low_index":     {baseURL + "/?view=stockList&type=annualIndexAsc", "주가지수 낮은종목"},
	"low_pbr":       {baseURL + "/?view=stockList&type=PBRAsc", "PBR 낮은 종목"},
	"low_per":       {baseURL + "/?view=stockList&type=PERAsc", "PER 낮은 종목"},
	"high_expected": {baseURL + "/?view=stockList&type=expectRateDesc", "기대수익률 높은 종목"},
	"high_cap":      {baseURL + "/?view=stockList&type=marketCapDesc", "시가총액 높은 종목"},
	"low_cap":       {baseURL + "/?view=stockList&type=marketCapAsc", "시가총액 낮은 종목"},
	"high_52w":      {baseURL + "/?view=stockList&type=returnHigh", "전고점 돌파 종목(52주)"},
	"high_3y":       {baseURL + "/?view=stockList&type=returnHigh3year", "전고점 돌파 종목(3년)"},
	"fund_buy":      {baseURL + "/?view=stockList&type=fundBuy", "연기금 순매수 종목"},
	"foreign_buy":   {baseURL + "/?view=stockList&type=foreignerBuy", "외국인 순매수 종목"},
	"fund_sell":     {baseURL + "/?view=stockList&type=fundSell", "연기금 순매도 종목"},
	"foreign_sell":  {baseURL + "/?view=stockList&type=foreignerSell", "외국인 순매도 종목"},
	"today_hot":     {baseURL + "/?view=stockList&type=todayHotStock", "금일 핫종목"},
}

// CrawlThemeListTab 테마 보기 탭 실시간 크롤링
func (rc *RealtimeCrawler) CrawlThemeListTab(tabKey string) (*RealtimeData, error) {
	tabInfo, exists := ThemeListURLs[tabKey]
	if !exists {
		return nil, fmt.Errorf("unknown theme tab: %s", tabKey)
	}

	resp, err := rc.client.Get(tabInfo.URL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("status code: %d", resp.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, err
	}

	result := &RealtimeData{
		Type:      "theme_list",
		Title:     tabInfo.Title,
		URL:       tabInfo.URL,
		CrawledAt: time.Now(),
	}

	// 테마 테이블에서 데이터 추출
	themeIdxRe := regexp.MustCompile(`themeIdx=(\d+)`)

	doc.Find("table tbody tr").Each(func(i int, row *goquery.Selection) {
		theme := make(map[string]interface{})

		// 테마명과 인덱스 추출
		nameCell := row.Find("th[scope='row'] a, td a[href*='themeIdx']").First()
		if nameCell.Length() > 0 {
			href, _ := nameCell.Attr("href")
			matches := themeIdxRe.FindStringSubmatch(href)
			if len(matches) >= 2 {
				if idx, err := strconv.Atoi(matches[1]); err == nil {
					theme["theme_idx"] = idx
				}
			}
			theme["name"] = strings.TrimSpace(nameCell.Text())
		}

		// 각 셀에서 데이터 추출
		cells := row.Find("td")
		if cells.Length() > 0 {
			cellTexts := make([]string, 0)
			cells.Each(func(j int, cell *goquery.Selection) {
				cellTexts = append(cellTexts, strings.TrimSpace(cell.Text()))
			})

			// 테마 리스트에서 일반적인 컬럼: 등락률, 기대수익률 등
			if len(cellTexts) > 0 {
				theme["values"] = cellTexts
			}
		}

		if len(theme) > 0 && theme["name"] != nil {
			result.Items = append(result.Items, theme)
		}
	})

	result.Count = len(result.Items)
	return result, nil
}

// CrawlStockListTab 종목 리스트 탭 실시간 크롤링
func (rc *RealtimeCrawler) CrawlStockListTab(tabKey string) (*RealtimeData, error) {
	tabInfo, exists := StockListURLs[tabKey]
	if !exists {
		return nil, fmt.Errorf("unknown stock tab: %s", tabKey)
	}

	resp, err := rc.client.Get(tabInfo.URL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("status code: %d", resp.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, err
	}

	result := &RealtimeData{
		Type:      "stock_list",
		Title:     tabInfo.Title,
		URL:       tabInfo.URL,
		CrawledAt: time.Now(),
	}

	// 종목 테이블에서 데이터 추출
	codeRe := regexp.MustCompile(`code=(\d{6})`)

	doc.Find("table tbody tr").Each(func(i int, row *goquery.Selection) {
		stock := make(map[string]interface{})

		// 종목명과 코드 추출
		firstCell := row.Find("th[scope='row'], td").First()
		stockLink := firstCell.Find("a[href*='finance.naver.com'], a[href*='code=']")

		if stockLink.Length() > 0 {
			href, _ := stockLink.Attr("href")
			matches := codeRe.FindStringSubmatch(href)
			if len(matches) >= 2 {
				stock["code"] = matches[1]
			}

			nameText := strings.TrimSpace(stockLink.Text())
			lines := strings.Split(nameText, "\n")
			if len(lines) > 0 {
				stock["name"] = strings.TrimSpace(lines[0])
			}

			if strings.Contains(nameText, "KOSPI") {
				stock["market"] = "KOSPI"
			} else if strings.Contains(nameText, "KOSDAQ") {
				stock["market"] = "KOSDAQ"
			}
		}

		if stock["code"] == nil || stock["name"] == nil {
			return
		}

		// 각 셀에서 데이터 추출
		cells := row.Find("td")
		cellIdx := 0
		cells.Each(func(j int, cell *goquery.Selection) {
			text := strings.TrimSpace(cell.Text())
			text = strings.ReplaceAll(text, ",", "")
			text = strings.ReplaceAll(text, "%", "")

			switch cellIdx {
			case 0: // 현재가
				if v, err := strconv.ParseInt(text, 10, 64); err == nil && v != 0 {
					stock["current_price"] = v
				}
			case 1: // 전일비
				if v, err := strconv.ParseInt(text, 10, 64); err == nil {
					stock["price_change"] = v
				}
			case 2: // 등락률
				if v, err := strconv.ParseFloat(text, 64); err == nil {
					stock["change_rate"] = v
				}
			}
			cellIdx++
		})

		result.Items = append(result.Items, stock)
	})

	result.Count = len(result.Items)
	return result, nil
}

// GetAvailableTabs 사용 가능한 탭 목록 반환
func GetAvailableTabs() map[string]interface{} {
	themeTabs := make(map[string]string)
	for k, v := range ThemeListURLs {
		themeTabs[k] = v.Title
	}

	stockTabs := make(map[string]string)
	for k, v := range StockListURLs {
		stockTabs[k] = v.Title
	}

	return map[string]interface{}{
		"theme_tabs": themeTabs,
		"stock_tabs": stockTabs,
	}
}
