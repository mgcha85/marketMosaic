package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"dx-unified/internal/shared/config"
	"dx-unified/internal/shared/scheduler"

	// DART
	dartAPI "dx-unified/internal/dart/api"
	dartDB "dx-unified/internal/dart/database"
	dartScheduler "dx-unified/internal/dart/scheduler"

	// Judal
	judalAPI "dx-unified/internal/judal/api"
	"dx-unified/internal/judal/crawler"
	judalDB "dx-unified/internal/judal/database"

	// Candle
	candleAPI "dx-unified/internal/candle/api"
	candleDB "dx-unified/internal/candle/database"
	"dx-unified/internal/candle/providers/alpaca"
	"dx-unified/internal/candle/providers/kiwoom"
	"dx-unified/internal/candle/providers/kiwoomrest"
	"dx-unified/internal/candle/service/candles"

	// News
	newsAPI "dx-unified/internal/news/api"
	"dx-unified/internal/news/fetcher"
	"dx-unified/internal/news/fetcher/naver"
	"dx-unified/internal/news/fetcher/newsapi"
	"dx-unified/internal/news/pipeline"
	newsMeili "dx-unified/internal/news/store/meili"

	"github.com/gin-gonic/gin"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Println("Starting DX Unified Server...")

	// Load Configuration
	cfg := config.Load()

	// ========== Initialize Databases ==========

	// DART DB
	if cfg.DartAPIKey != "" {
		if err := dartDB.InitDB(cfg.DartDBPath); err != nil {
			log.Printf("[DART] Failed to initialize database: %v", err)
		} else {
			log.Println("[DART] Database initialized")
		}
	}

	// Judal DB
	if err := judalDB.InitDB(cfg.JudalDBPath); err != nil {
		log.Printf("[JUDAL] Failed to initialize database: %v", err)
	} else {
		log.Println("[JUDAL] Database initialized")
	}

	// Candle (DuckDB + Parquet/Hive)
	if err := candleDB.InitDB(cfg.CandleDataDir); err != nil {
		log.Printf("[CANDLE] Failed to initialize DuckDB: %v", err)
	} else {
		log.Println("[CANDLE] DuckDB initialized with Hive partition support")
	}

	// News Store (Meilisearch)
	var newsStore *newsMeili.Store
	if cfg.MeiliHost != "" {
		var err error
		newsStore, err = newsMeili.New(cfg.MeiliHost, cfg.MeiliAPIKey)
		if err != nil {
			log.Printf("[NEWS] Failed to initialize Meilisearch: %v", err)
		} else {
			log.Println("[NEWS] Meilisearch connected")
		}
	}

	// ========== Setup HTTP Server ==========

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Logger())
	r.Use(gin.Recovery())

	// CORS Middleware
	r.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "service": "dx-unified"})
	})

	// ========== Setup Services ==========

	// Candle Service
	var candleSvc *candles.Service
	if candleDB.DB != nil {
		kiwoomClient := kiwoom.NewClient(cfg.KiwoomAppKey, cfg.KiwoomAppSecret, cfg.KiwoomBaseURL)
		alpacaClient := alpaca.NewClient(cfg.AlpacaAPIKey, cfg.AlpacaAPISecret)
		candleSvc = candles.NewService(kiwoomClient, alpacaClient)
	}

	// News Service
	var newsProcessor *pipeline.Processor
	if newsStore != nil {
		naverFetcher := naver.New(cfg)
		newsapiFetcher := newsapi.New(cfg)
		fetchers := []fetcher.Fetcher{naverFetcher, newsapiFetcher}
		newsProcessor = pipeline.NewProcessor(cfg, newsStore, fetchers)
	}

	// ========== Register API Routes ==========

	// DART API (/dart/*)
	if dartDB.DB != nil {
		dartHandler := dartAPI.NewHandler(dartDB.GetDB())
		dartHandler.RegisterRoutes(r.Group(""))
		log.Println("[DART] API routes registered")
	}

	// Judal API (/judal/*)
	if judalDB.DB != nil {
		judalHandler := judalAPI.NewHandler()
		judalHandler.RegisterRoutes(r.Group(""))
		log.Println("[JUDAL] API routes registered")
	}

	// Candle API (/candle/*)
	if candleDB.DB != nil && candleSvc != nil {
		// Create Kiwoom REST client for fundamentals and daily candles
		kiwoomRestClient := kiwoomrest.NewClient(cfg.KiwoomRestAPIURL)
		if kiwoomRestClient.IsConfigured() {
			log.Println("[KIWOOM-REST] API client configured")
		}
		candleHandler := candleAPI.NewHandlerWithKiwoom(candleSvc, kiwoomRestClient)
		candleHandler.RegisterRoutes(r.Group(""))
		log.Println("[CANDLE] API routes registered")
	}

	// News API (/news/*)
	if newsStore != nil {
		newsHandler := newsAPI.NewHandler(newsStore)
		newsHandler.RegisterRoutes(r.Group(""))
		log.Println("[NEWS] API routes registered")
	}

	// ========== Unified Scheduler ==========

	sched := scheduler.New()

	// DART Jobs
	if cfg.DartAPIKey != "" && dartDB.DB != nil {
		dartJobs := dartScheduler.NewDartJobs(cfg.DartAPIKey, cfg.StorageDir)
		go dartJobs.InitialSetup()
		sched.AddJob("DART-FetchFilings", "@hourly", dartJobs.FetchFilings)
		sched.AddJob("DART-DownloadDocs", "@every 5m", dartJobs.DownloadDocuments)
		sched.AddJob("DART-UpdateCorpCodes", "@weekly", dartJobs.UpdateCorpCodes)
	}

	// Candle Jobs (US Market)
	// US Market closes at 06:00 KST (approx). Schedule ingestion after close.
	if candleSvc != nil {
		// Daily ingestion at 06:00 KST
		sched.AddJob("Candle-Ingest-US-1d", "0 6 * * 2-6", func() {
			candleSvc.Run(candles.IngestParams{Market: "US", Timeframe: "1d"})
		})
		// Minute ingestion at 06:10 KST
		sched.AddJob("Candle-Ingest-US-1m", "10 6 * * 2-6", func() {
			candleSvc.Run(candles.IngestParams{Market: "US", Timeframe: "1m"})
		})
	}

	// Judal Jobs (Themes)
	// Daily crawl at 00:00 KST
	if judalDB.DB != nil {
		judalJob := crawler.NewCrawler(1500 * time.Millisecond)
		sched.AddJob("Judal-Daily-Crawl", "0 0 * * *", func() {
			_, err := judalJob.CrawlAllWithHistory()
			if err != nil {
				log.Printf("[JUDAL] Daily crawl failed: %v", err)
			} else {
				log.Println("[JUDAL] Daily crawl completed")
			}
		})
	}

	// News Jobs (Every 15 mins)
	if newsProcessor != nil {
		sched.AddJob("News-Fetch", "*/15 * * * *", func() {
			newsProcessor.Run()
		})
	}

	sched.Start()

	// ========== Start HTTP Server ==========

	srv := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: r,
	}

	go func() {
		log.Printf("Server listening on port %s", cfg.Port)
		log.Println("")
		log.Println("API Endpoints:")
		log.Println("  GET  /health                     - Health check")
		log.Println("")
		log.Println("  DART (/dart/*):")
		log.Println("    GET  /dart/corps               - List corporations")
		log.Println("    GET  /dart/filings             - List filings")
		log.Println("    GET  /dart/filings/:rcept_no   - Get filing detail")
		log.Println("")
		log.Println("  JUDAL (/judal/*):")
		log.Println("    GET  /judal/themes             - List themes")
		log.Println("    GET  /judal/themes/:idx/stocks - Get theme stocks")
		log.Println("    GET  /judal/stocks             - List stocks")
		log.Println("    GET  /judal/stocks/:code       - Get stock detail")
		log.Println("    GET  /judal/realtime/themes/:tab - Realtime theme crawl")
		log.Println("    GET  /judal/realtime/stocks/:tab - Realtime stock crawl")
		log.Println("")
		log.Println("  CANDLE (/candle/*):")
		log.Println("    GET  /candle/universe          - List universe")
		log.Println("    GET  /candle/stocks            - Get candle data")
		log.Println("    GET  /candle/stocks/:symbol    - Get symbol candles")
		log.Println("    GET  /candle/runs              - Get ingest runs")
		log.Println("")
		log.Println("  NEWS (/news/*):")
		log.Println("    GET  /news/articles            - List articles")
		log.Println("    GET  /news/articles/:id        - Get article")
		log.Println("    GET  /news/search?q=...        - Search articles")
		log.Println("    GET  /news/runs                - Get batch runs")
		log.Println("")

		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()

	// ========== Graceful Shutdown ==========

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	sched.Stop()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("Server shutdown error: %v", err)
	}

	// Close databases
	if judalDB.DB != nil {
		judalDB.CloseDB()
	}
	if candleDB.DB != nil {
		candleDB.Close()
	}

	log.Println("Server stopped. Goodbye.")
}
