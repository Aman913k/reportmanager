package route

import (
	"github.com/Aman913k/ReportManager/handler"
	"github.com/gin-gonic/gin"
)

func RegisterRoute(r *gin.Engine) {
	r.GET("transaction/download/:id", handler.DownloadTransaction)
	r.GET("ledger/download/:id", handler.DownloadLedger)
}
