package main

import (
	"github.com/Aman913k/ReportManager/route"
	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()
	route.RegisterRoute(r)

	r.Run()
}

