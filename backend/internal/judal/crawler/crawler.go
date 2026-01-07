package crawler

import (
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"dx-unified/internal/judal/database"
	"dx-unified/internal/judal/models"

	"github.com/PuerkitoBio/goquery"
)

const (
	baseURL      = "https://www.judal.co.kr"
	themeListURL = baseURL + "/?view=themeList"
	stockListURL = baseURL + "/?view=stockList&themeIdx="
)

// Crawler 크롤러 구조체
type Crawler struct {
	client *http.Client
	repo   *database.Repository
	delay  time.Duration
}

// NewCrawler 새 크롤러 생성
func NewCrawler(delay time.Duration) *Crawler {
	return &Crawler{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		repo:  database.NewRepository(),
		delay: delay,
	}
}

// CrawlResult 크롤링 결과
type CrawlResult struct {
	ThemesCrawled int       `json:"themes_crawled"`
	StocksCrawled int       `json:"stocks_crawled"`
	HistorySaved  int       `json:"history_saved,omitempty"`
	CrawlDate     string    `json:"crawl_date,omitempty"`
	Errors        []string  `json:"errors,omitempty"`
	StartTime     time.Time `json:"start_time"`
	EndTime       time.Time `json:"end_time"`
	Duration      string    `json:"duration"`
}

// CrawlAll 전체 크롤링 실행 (최신 데이터만 업데이트)
func (c *Crawler) CrawlAll() (*CrawlResult, error) {
	return c.doCrawl(false)
}

// CrawlAllWithHistory 전체 크롤링 + 히스토리 저장 (일배치용)
func (c *Crawler) CrawlAllWithHistory() (*CrawlResult, error) {
	return c.doCrawl(true)
}

// doCrawl 실제 크롤링 수행
func (c *Crawler) doCrawl(saveHistory bool) (*CrawlResult, error) {
	result := &CrawlResult{
		StartTime: time.Now(),
	}

	crawlDate := time.Now().Format("2006-01-02")
	if saveHistory {
		result.CrawlDate = crawlDate
	}

	// 1. 테마 목록 크롤링
	log.Println("Starting theme list crawling...")
	themes, err := c.CrawlThemeList()
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("Theme list error: %v", err))
		log.Printf("Error crawling theme list: %v", err)
	} else {
		result.ThemesCrawled = len(themes)
		log.Printf("Crawled %d themes", len(themes))
	}

	// 2. 각 테마별 종목 크롤링
	stockCodes := make(map[string]bool) // 중복 제거용
	historyCount := 0

	for i, theme := range themes {
		log.Printf("[%d/%d] Crawling stocks for theme: %s (idx: %d)", i+1, len(themes), theme.Name, theme.ThemeIdx)

		stocks, err := c.CrawlStockList(theme.ThemeIdx)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("Stock list error (theme %d): %v", theme.ThemeIdx, err))
			log.Printf("Error crawling stocks for theme %d: %v", theme.ThemeIdx, err)
			continue
		}

		// 테마-종목 매핑 저장 및 히스토리 저장
		for _, stock := range stocks {
			if !stockCodes[stock.Code] {
				stockCodes[stock.Code] = true

				// 히스토리 저장 (saveHistory가 true이고 처음 보는 종목일 때만)
				if saveHistory {
					if err := c.repo.SaveStockHistory(crawlDate, &stock); err != nil {
						log.Printf("Error saving history for stock %s: %v", stock.Code, err)
					} else {
						historyCount++
					}
				}
			}
			if err := c.repo.AddThemeStock(theme.ThemeIdx, stock.Code); err != nil {
				log.Printf("Error adding theme-stock mapping: %v", err)
			}
		}

		// 테마 종목 수 업데이트
		theme.StockCount = len(stocks)
		c.repo.UpsertTheme(&theme)

		log.Printf("  -> Found %d stocks", len(stocks))

		// Rate limiting
		time.Sleep(c.delay)
	}

	result.StocksCrawled = len(stockCodes)
	result.HistorySaved = historyCount
	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime).String()

	// 크롤링 로그 저장
	if saveHistory {
		durationSec := result.EndTime.Sub(result.StartTime).Seconds()
		if err := c.repo.SaveCrawlLog(crawlDate, "daily_batch", result.ThemesCrawled, result.StocksCrawled, durationSec, "completed"); err != nil {
			log.Printf("Error saving crawl log: %v", err)
		}
	}

	log.Printf("Crawling completed. Themes: %d, Stocks: %d, History: %d, Duration: %s",
		result.ThemesCrawled, result.StocksCrawled, historyCount, result.Duration)

	return result, nil
}

// CrawlThemeList 테마 목록 크롤링
func (c *Crawler) CrawlThemeList() ([]models.Theme, error) {
	resp, err := c.client.Get(themeListURL)
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

	var themes []models.Theme
	themeIdxRe := regexp.MustCompile(`themeIdx=(\d+)`)

	// 테마 링크에서 테마 정보 추출
	doc.Find("a[href*='view=stockList'][href*='themeIdx=']").Each(func(i int, s *goquery.Selection) {
		href, exists := s.Attr("href")
		if !exists {
			return
		}

		matches := themeIdxRe.FindStringSubmatch(href)
		if len(matches) < 2 {
			return
		}

		themeIdx, err := strconv.Atoi(matches[1])
		if err != nil {
			return
		}

		name := strings.TrimSpace(s.Text())
		if name == "" || name == "테마토크" || strings.Contains(name, "테마토크") {
			return
		}

		theme := models.Theme{
			ThemeIdx: themeIdx,
			Name:     name,
		}

		// 데이터베이스에 저장
		if err := c.repo.UpsertTheme(&theme); err != nil {
			log.Printf("Error upserting theme %s: %v", name, err)
		}

		themes = append(themes, theme)
	})

	// 중복 제거
	seen := make(map[int]bool)
	var uniqueThemes []models.Theme
	for _, t := range themes {
		if !seen[t.ThemeIdx] {
			seen[t.ThemeIdx] = true
			uniqueThemes = append(uniqueThemes, t)
		}
	}

	return uniqueThemes, nil
}

// CrawlStockList 특정 테마의 종목 크롤링
func (c *Crawler) CrawlStockList(themeIdx int) ([]models.Stock, error) {
	url := stockListURL + strconv.Itoa(themeIdx)
	resp, err := c.client.Get(url)
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

	return c.parseStockList(doc)
}

// parseStockList HTML에서 종목 리스트 파싱
func (c *Crawler) parseStockList(doc *goquery.Document) ([]models.Stock, error) {
	var stocks []models.Stock
	codeRe := regexp.MustCompile(`code=(\d{6})`)

	// 테이블 행에서 종목 정보 추출
	doc.Find("table tbody tr").Each(func(i int, row *goquery.Selection) {
		stock := models.Stock{}

		// 종목명과 코드 추출
		firstCell := row.Find("th[scope='row'], td").First()
		stockLink := firstCell.Find("a[href*='finance.naver.com']")
		if stockLink.Length() == 0 {
			stockLink = firstCell.Find("a[href*='code=']")
		}

		if stockLink.Length() > 0 {
			href, _ := stockLink.Attr("href")
			matches := codeRe.FindStringSubmatch(href)
			if len(matches) >= 2 {
				stock.Code = matches[1]
			}

			// 종목명 추출
			nameText := strings.TrimSpace(stockLink.Text())
			lines := strings.Split(nameText, "\n")
			if len(lines) > 0 {
				stock.Name = strings.TrimSpace(lines[0])
			}

			// 시장 구분 추출
			if strings.Contains(nameText, "KOSPI") {
				stock.Market = "KOSPI"
			} else if strings.Contains(nameText, "KOSDAQ") {
				stock.Market = "KOSDAQ"
			}
		}

		if stock.Code == "" || stock.Name == "" {
			return
		}

		// 각 셀에서 데이터 추출
		cells := row.Find("td")
		cells.Each(func(j int, cell *goquery.Selection) {
			text := strings.TrimSpace(cell.Text())
			c.parseStockCell(&stock, j, text)
		})

		// 데이터베이스에 저장
		if err := c.repo.UpsertStock(&stock); err != nil {
			log.Printf("Error upserting stock %s: %v", stock.Code, err)
		}

		stocks = append(stocks, stock)
	})

	return stocks, nil
}

// parseStockCell 셀 데이터 파싱
func (c *Crawler) parseStockCell(stock *models.Stock, colIndex int, text string) {
	// 숫자 정리 (콤마, % 제거)
	cleanNum := func(s string) string {
		s = strings.ReplaceAll(s, ",", "")
		s = strings.ReplaceAll(s, "%", "")
		s = strings.TrimSpace(s)
		return s
	}

	parseInt := func(s string) int64 {
		s = cleanNum(s)
		v, _ := strconv.ParseInt(s, 10, 64)
		return v
	}

	parseFloat := func(s string) float64 {
		s = cleanNum(s)
		v, _ := strconv.ParseFloat(s, 64)
		return v
	}

	// 컬럼 인덱스에 따라 데이터 매핑
	switch colIndex {
	case 0: // 현재가
		if v := parseInt(text); v != 0 {
			stock.CurrentPrice.Int64 = v
			stock.CurrentPrice.Valid = true
		}
	case 1: // 전일비
		if v := parseInt(text); v != 0 {
			stock.PriceChange.Int64 = v
			stock.PriceChange.Valid = true
		}
	case 2: // 등락률
		if v := parseFloat(text); v != 0 {
			stock.ChangeRate.Float64 = v
			stock.ChangeRate.Valid = true
		}
	case 3: // 3일합산
		if v := parseFloat(text); v != 0 {
			stock.ThreeDaySum.Float64 = v
			stock.ThreeDaySum.Valid = true
		}
	case 4: // 52주 최고가
		if v := parseInt(text); v != 0 {
			stock.High52W.Int64 = v
			stock.High52W.Valid = true
		}
	case 5: // 52주 최저가
		if v := parseInt(text); v != 0 {
			stock.Low52W.Int64 = v
			stock.Low52W.Valid = true
		}
	case 8: // 52주 소외지수
		if v := parseFloat(text); v != 0 {
			stock.NeglectIndex52W.Float64 = v
			stock.NeglectIndex52W.Valid = true
		}
	case 12: // 3년 주가지수
		if v := parseFloat(text); v != 0 {
			stock.PriceIndex3Y.Float64 = v
			stock.PriceIndex3Y.Valid = true
		}
	case 13: // 기대수익률
		if v := parseFloat(text); v != 0 {
			stock.ExpectedReturn.Float64 = v
			stock.ExpectedReturn.Valid = true
		}
	case 14: // PBR
		if v := parseFloat(text); v != 0 {
			stock.PBR.Float64 = v
			stock.PBR.Valid = true
		}
	case 15: // PER
		if v := parseFloat(text); v != 0 {
			stock.PER.Float64 = v
			stock.PER.Valid = true
		}
	case 16: // EPS
		if v := parseInt(text); v != 0 {
			stock.EPS.Int64 = v
			stock.EPS.Valid = true
		}
	case 17: // 시가총액
		// 시가총액은 억 단위로 표시됨
		if v := parseInt(text); v != 0 {
			stock.MarketCap.Int64 = v
			stock.MarketCap.Valid = true
		}
	}
}
