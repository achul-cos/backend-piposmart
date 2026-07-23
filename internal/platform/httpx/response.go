package httpx

import "github.com/gin-gonic/gin"

type SuccessEnvelope struct {
	Data any          `json:"data"`
	Meta ResponseMeta `json:"meta"`
}

type ResponseMeta struct {
	RequestID string `json:"request_id"`
}

type ErrorEnvelope struct {
	Error APIError `json:"error"`
}

type APIError struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	Details   any    `json:"details,omitempty"`
	RequestID string `json:"request_id"`
}

func Success(c *gin.Context, status int, data any) {
	c.JSON(status, SuccessEnvelope{
		Data: data,
		Meta: ResponseMeta{RequestID: RequestID(c)},
	})
}

func Error(c *gin.Context, status int, code, message string, details any) {
	c.AbortWithStatusJSON(status, ErrorEnvelope{
		Error: APIError{
			Code:      code,
			Message:   message,
			Details:   details,
			RequestID: RequestID(c),
		},
	})
}

func RequestID(c *gin.Context) string {
	requestID, _ := c.Get("request_id")
	value, _ := requestID.(string)
	return value
}
