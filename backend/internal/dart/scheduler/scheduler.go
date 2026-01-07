package scheduler

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"

	"dx-unified/internal/dart/database"
	"dx-unified/internal/dart/models"
	"dx-unified/pkg/dart"

	"gorm.io/gorm/clause"
)

// DartJobs contains all scheduled DART jobs
type DartJobs struct {
	client     *dart.Client
	storageDir string
}

// NewDartJobs creates a new DartJobs instance
func NewDartJobs(apiKey, storageDir string) *DartJobs {
	return &DartJobs{
		client:     dart.NewClient(apiKey),
		storageDir: storageDir,
	}
}

// UpdateCorpCodes fetches and updates corp codes
func (j *DartJobs) UpdateCorpCodes() {
	log.Println("[DART] Fetching corp codes...")
	corps, err := j.client.GetCorpCode()
	if err != nil {
		log.Printf("[DART] Failed to update corp codes: %v\n", err)
		return
	}

	batchSize := 100
	for i := 0; i < len(corps); i += batchSize {
		end := i + batchSize
		if end > len(corps) {
			end = len(corps)
		}

		batch := corps[i:end]
		result := database.DB.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "corp_code"}},
			DoUpdates: clause.AssignmentColumns([]string{"corp_name", "stock_code", "modified_at"}),
		}).Create(&batch)

		if result.Error != nil {
			log.Printf("[DART] Batch error at index %d: %v\n", i, result.Error)
		}
	}
	log.Printf("[DART] Successfully updated %d corp codes\n", len(corps))
}

// FetchFilings fetches recent filings (3-day lookback)
func (j *DartJobs) FetchFilings() {
	log.Println("[DART] Fetching recent filings (3-day lookback)...")
	for i := 0; i < 3; i++ {
		targetDate := time.Now().AddDate(0, 0, -i).Format("20060102")
		log.Printf("[DART] Fetching filings for %s...", targetDate)

		filings, err := j.client.GetDailyFilings(targetDate)
		if err != nil {
			log.Printf("[DART] Error fetching filings for %s: %v\n", targetDate, err)
			continue
		}

		if len(filings) > 0 {
			result := database.DB.Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "rcept_no"}},
				DoUpdates: clause.AssignmentColumns([]string{"corp_code", "corp_name", "report_nm", "rcept_dt", "flr_nm", "rm", "dcm_no"}),
			}).Create(&filings)

			if result.Error != nil {
				log.Printf("[DART] Error saving filings: %v\n", result.Error)
			} else {
				log.Printf("[DART] Processed %d filings for %s\n", len(filings), targetDate)
			}
		}
	}
}

// DownloadDocuments downloads pending documents
func (j *DartJobs) DownloadDocuments() {
	log.Println("[DART] Starting Document Downloader...")

	var pendingFilings []models.Filing

	err := database.DB.Raw(`
		SELECT * FROM filings 
		WHERE rcept_no NOT IN (SELECT rcept_no FROM filing_documents)
		ORDER BY rcept_dt DESC
		LIMIT 10
	`).Scan(&pendingFilings).Error

	if err != nil {
		log.Printf("[DART] Error finding pending downloads: %v\n", err)
		return
	}

	if len(pendingFilings) == 0 {
		return
	}

	if err := os.MkdirAll(j.storageDir, 0755); err != nil {
		log.Printf("[DART] Error creating storage dir: %v\n", err)
		return
	}

	for _, f := range pendingFilings {
		log.Printf("[DART] Downloading document for %s (%s)\n", f.CorpName, f.RceptNo)

		filename := fmt.Sprintf("%s.zip", f.RceptNo)
		filePath := filepath.Join(j.storageDir, filename)

		if err := j.client.DownloadDocument(f.RceptNo, filePath); err != nil {
			log.Printf("[DART] Failed to download %s: %v\n", f.RceptNo, err)
			continue
		}

		hash, err := calculateSHA256(filePath)
		if err != nil {
			log.Printf("[DART] Failed to calculate hash for %s: %v\n", filePath, err)
		}

		doc := models.FilingDocument{
			RceptNo:    f.RceptNo,
			DocType:    "MAIN_XML_ZIP",
			StorageURI: filePath,
			SHA256:     hash,
		}

		if err := database.DB.Create(&doc).Error; err != nil {
			log.Printf("[DART] Failed to save DB record for %s: %v\n", f.RceptNo, err)
		}

		time.Sleep(500 * time.Millisecond)
	}
}

// InitialSetup runs initial corp code fetch if empty
func (j *DartJobs) InitialSetup() {
	var count int64
	database.DB.Model(&models.Corp{}).Count(&count)
	if count == 0 {
		log.Println("[DART] Corp table empty. Fetching initial Corp Codes...")
		j.UpdateCorpCodes()
	} else {
		log.Printf("[DART] Corp table has %d records. Skipping initial fetch.\n", count)
	}
}

func calculateSHA256(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}
