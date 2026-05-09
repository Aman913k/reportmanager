package models

import "time"

type LedgerReport struct {
	ReportID       string        `json:"report_id"`
	GeneratedAt    time.Time     `json:"generated_at"`
	AccountID      string        `json:"account_id"`
	OpeningBalance float64       `json:"opening_balance"`
	ClosingBalance float64       `json:"closing_balance"`
	TotalCredits   float64       `json:"total_credits"`
	TotalDebits    float64       `json:"total_debits"`
	Transactions   []Transaction `json:"transactions"`              // Detailed list for the Excel rows [cite: 16]
	S3DownloadURL  string        `json:"s3_download_url,omitempty"` // For the pre-signed URLs you implemented [cite: 9]
}
