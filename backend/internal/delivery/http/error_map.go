package http

import (
	"errors"
	"net/http"

	"github.com/RakhaYandra/pulse/internal/domain"
	"github.com/gin-gonic/gin"
)

// Central domain → HTTP mapping. Only place that knows status codes.
func writeErr(c *gin.Context, err error) {
	var fe *domain.FieldError
	switch {
	case errors.As(err, &fe):
		c.JSON(http.StatusBadRequest, gin.H{"error": fe.Message})
	case errors.Is(err, domain.ErrValidation):
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
	case errors.Is(err, domain.ErrUnauthorized):
		c.JSON(http.StatusUnauthorized, gin.H{"error": "email/password salah"})
	case errors.Is(err, domain.ErrConflict):
		c.JSON(http.StatusConflict, gin.H{"error": "email already registered"})
	case errors.Is(err, domain.ErrNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
	}
}

func writeNotFound(c *gin.Context, resource string) {
	c.JSON(http.StatusNotFound, gin.H{"error": resource + " not found"})
}

func writeOK(c *gin.Context, data any) {
	c.JSON(http.StatusOK, gin.H{"data": data})
}
