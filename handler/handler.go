package handler

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/Aman913k/ReportManager/models"
	"github.com/Aman913k/ReportManager/storage"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/gin-gonic/gin"
	"github.com/johnfercher/maroto/v2"
	"github.com/johnfercher/maroto/v2/pkg/components/col"
	"github.com/johnfercher/maroto/v2/pkg/components/row"
	"github.com/johnfercher/maroto/v2/pkg/components/text"
	"github.com/johnfercher/maroto/v2/pkg/config"
	"github.com/johnfercher/maroto/v2/pkg/consts/align"
	"github.com/johnfercher/maroto/v2/pkg/consts/fontstyle"
	"github.com/johnfercher/maroto/v2/pkg/props"
)

// In-memory "database"
var transactionDB = []models.Transaction{
	{ID: "txn_001", UserID: 1, Amount: 5000.00, Currency: "INR", Status: "completed", TransactionType: "credit", CreatedAt: time.Now().Add(-48 * time.Hour), Reference: "Salary"},
	{ID: "txn_002", UserID: 1, Amount: -200.00, Currency: "INR", Status: "completed", TransactionType: "debit", CreatedAt: time.Now().Add(-24 * time.Hour), Reference: "Groceries"},
	{ID: "txn_003", UserID: 123, Amount: 1500.00, Currency: "INR", Status: "completed", TransactionType: "credit", CreatedAt: time.Now().Add(-10 * time.Hour), Reference: "Refund"},
	{ID: "txn_004", UserID: 123, Amount: -50.00, Currency: "INR", Status: "pending", TransactionType: "debit", CreatedAt: time.Now().Add(-1 * time.Hour), Reference: "Coffee"},
}

func DownloadTransaction(c *gin.Context) {
	// 1. Get authenticated user
	idStr := c.Param("id")
	userID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id in path"})
		return
	}

	// 2. Query parameters
	startDate := c.Query("start_date")
	endDate := c.Query("end_date")
	limitStr := c.DefaultQuery("limit", "500")

	limit, _ := strconv.Atoi(limitStr)
	if limit < 1 || limit > 5000 {
		limit = 5000
	}

	var start, end time.Time
	layout := "2006-01-02"
	if startDate != "" {
		start, err = time.Parse(layout, startDate)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid start_date format"})
			return
		}
	} else {
		start = time.Now().AddDate(0, -1, 0) // Default last 30 days
	}

	if endDate != "" {
		end, err = time.Parse(layout, endDate)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid end_date format"})
			return
		}
		end = end.Add(24*time.Hour - time.Second)
	} else {
		end = time.Now()
	}

	// 3. S3 Check: See if this report already exists
	bucket := "my-pdf-storage-go"
	// Create a stable key based on parameters (no timestamp for the check)
	reportKey := fmt.Sprintf("statements/user_%d/statement_%s_to_%s.pdf", userID, start.Format("20060102"), end.Format("20060102"))

	awsCfg, err := awsconfig.LoadDefaultConfig(context.TODO(), awsconfig.WithRegion("eu-north-1"))
	if err != nil {
		c.JSON(500, gin.H{"error": "aws config failed"})
		return
	}
	s3Client := s3.NewFromConfig(awsCfg)

	exists, _ := storage.CheckFileExistsInS3(s3Client, bucket, reportKey)
	if exists {
		downloadURL, err := storage.GeneratePresignedURL(s3Client, bucket, reportKey, 15*time.Minute)
		if err == nil {
			c.JSON(http.StatusOK, gin.H{
				"message":       "existing report found",
				"download_url":  downloadURL,
				"expires_in":    "15 minutes",
				"s3_object_key": reportKey,
			})
			return
		}
	}

	// 4. Fetch transactions from in-memory DB
	transactions := getUserTransactions(userID, start, end, limit)
	if len(transactions) == 0 {
		c.JSON(http.StatusOK, gin.H{"message": "No transactions found for this period"})
		return
	}

	// 5. Create PDF with Maroto v2
	cfg := config.NewBuilder().WithPageNumber().Build()
	mrt := maroto.New(cfg)

	mrt.RegisterHeader(
		row.New(14).Add(
			text.NewCol(12, "Transaction Statement", props.Text{Style: fontstyle.Bold, Size: 18, Align: align.Center}),
		),
		row.New(10).Add(
			text.NewCol(12, fmt.Sprintf("User ID: %d  •  Generated: %s", userID, time.Now().Format("02 Jan 2006 15:04 MST")),
				props.Text{Size: 10, Align: align.Center}),
		),
		row.New(6),
	)

	mrt.RegisterFooter(
		row.New(10).Add(
			text.NewCol(12, "Page {current} of {total} • Confidential", props.Text{Size: 9, Align: align.Center}),
		),
	)

	mrt.AddRows(
		row.New(12).Add(
			text.NewCol(12, fmt.Sprintf("Period: %s to %s • %d transactions",
				start.Format("02 Jan 2006"), end.Format("02 Jan 2006"), len(transactions)),
				props.Text{Size: 13, Style: fontstyle.Bold, Align: align.Center}),
		),
		row.New(8),
	)

	headerProps := props.Text{Style: fontstyle.Bold, Size: 11, Align: align.Center}
	mrt.AddRows(row.New(11).Add(
		text.NewCol(3, "ID / Ref", headerProps),
		text.NewCol(3, "Date & Time", headerProps),
		text.NewCol(2, "Type", headerProps),
		text.NewCol(2, "Amount", headerProps),
		text.NewCol(2, "Status", headerProps),
	))

	var totalCredit, totalDebit float64
	for _, tx := range transactions {
		amountDisplay := fmt.Sprintf("%.2f %s", tx.Amount, tx.Currency)
		if tx.Amount > 0 {
			amountDisplay = "+" + amountDisplay
			totalCredit += tx.Amount
		} else {
			totalDebit += tx.Amount
		}

		mrt.AddRows(row.New(9).Add(
			text.NewCol(3, tx.ID, props.Text{Size: 10, Align: align.Left}),
			text.NewCol(3, tx.CreatedAt.Format("02 Jan 2006 15:04"), props.Text{Size: 10, Align: align.Center}),
			text.NewCol(2, tx.TransactionType, props.Text{Size: 10, Align: align.Center}),
			text.NewCol(2, amountDisplay, props.Text{Size: 10, Align: align.Right}),
			text.NewCol(2, tx.Status, props.Text{Size: 10, Align: align.Center}),
		))
	}

	mrt.AddRows(
		row.New(12),
		row.New(10).Add(
			col.New(8).Add(text.New("Totals:", props.Text{Style: fontstyle.Bold, Size: 11})),
			col.New(4).Add(text.New(fmt.Sprintf("Credit: +%.2f | Debit: %.2f", totalCredit, totalDebit),
				props.Text{Style: fontstyle.Bold, Size: 11, Align: align.Right})),
		),
	)

	doc, err := mrt.Generate()
	if err != nil {
		c.JSON(500, gin.H{"error": "failed to generate PDF"})
		return
	}
	pdfBytes := doc.GetBytes()

	// 6. Upload to S3 and Respond
	err = storage.UploadPDFToS3(s3Client, bucket, reportKey, pdfBytes)
	if err != nil {
		c.JSON(500, gin.H{"error": "S3 upload failed: " + err.Error()})
		return
	}

	downloadURL, err := storage.GeneratePresignedURL(s3Client, bucket, reportKey, 15*time.Minute)
	if err != nil {
		c.JSON(500, gin.H{"error": "presign failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":       "report generated and uploaded",
		"download_url":  downloadURL,
		"expires_in":    "15 minutes",
		"s3_object_key": reportKey,
	})
}

func getUserTransactions(userID int64, start, end time.Time, limit int) []models.Transaction {
	var filtered []models.Transaction
	count := 0
	for _, tx := range transactionDB {
		if tx.UserID == userID && (tx.CreatedAt.After(start) || tx.CreatedAt.Equal(start)) && (tx.CreatedAt.Before(end) || tx.CreatedAt.Equal(end)) {
			filtered = append(filtered, tx)
			count++
			if count >= limit {
				break
			}
		}
	}
	return filtered
}

func DownloadLedger(c *gin.Context) {
	// Implementation for Ledger would follow a similar pattern
}
