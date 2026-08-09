package httpx

import "github.com/gin-gonic/gin"

const (
	contextErrorStatusKey  = "_httpx_error_status"
	contextErrorCodeKey    = "_httpx_error_code"
	contextErrorMessageKey = "_httpx_error_message"
	contextErrorDetailsKey = "_httpx_error_details"
	contextErrorPrivateKey = "_httpx_error_private_details"
)

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

// PaginationMeta holds pagination metadata
type PaginationMeta struct {
	Page  int   `json:"page"`
	Limit int   `json:"limit"`
	Total int64 `json:"total"`
}

func Success(c *gin.Context, status int, data any) {
	c.JSON(status, SuccessEnvelope{
		Data: data,
		Meta: ResponseMeta{RequestID: RequestID(c)},
	})
}

func Error(c *gin.Context, status int, code, message string, details any) {
	c.Set(contextErrorStatusKey, status)
	c.Set(contextErrorCodeKey, code)
	c.Set(contextErrorMessageKey, message)
	c.Set(contextErrorDetailsKey, details)

	c.AbortWithStatusJSON(status, ErrorEnvelope{
		Error: APIError{
			Code:      code,
			Message:   message,
			Details:   details,
			RequestID: RequestID(c),
		},
	})
}

func SetPrivateErrorDetails(c *gin.Context, details any) {
	c.Set(contextErrorPrivateKey, details)
}

func InternalServerError(c *gin.Context, message string, err error) {
	if err != nil {
		SetPrivateErrorDetails(c, err.Error())
	}
	Error(c, 500, "INTERNAL_ERROR", message, nil)
}

type ErrorInfo struct {
	Status  int
	Code    string
	Message string
	Details any
	Private any
}

func CurrentError(c *gin.Context) (ErrorInfo, bool) {
	statusValue, ok := c.Get(contextErrorStatusKey)
	if !ok {
		return ErrorInfo{}, false
	}

	status, _ := statusValue.(int)
	code, _ := c.Get(contextErrorCodeKey)
	message, _ := c.Get(contextErrorMessageKey)
	details, _ := c.Get(contextErrorDetailsKey)
	privateDetails, _ := c.Get(contextErrorPrivateKey)

	codeString, _ := code.(string)
	messageString, _ := message.(string)

	return ErrorInfo{
		Status:  status,
		Code:    codeString,
		Message: messageString,
		Details: details,
		Private: privateDetails,
	}, true
}

func RequestID(c *gin.Context) string {
	requestID, _ := c.Get("request_id")
	value, _ := requestID.(string)
	return value
}
