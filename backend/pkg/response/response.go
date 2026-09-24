package response

import (
	"github.com/gin-gonic/gin"
)

func OK(c *gin.Context, data any) {
	c.JSON(200, gin.H{"data": data})
}

func Err(c *gin.Context, code int, msg string) {
	c.JSON(code, gin.H{"error": msg})
}
