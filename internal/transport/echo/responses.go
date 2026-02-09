package echo

import "net/http"

type SuccessResponse struct {
	Status       string      `json:"status"`
	ResponseCode int         `json:"response_code"`
	Data         interface{} `json:"data"`
}

type FailureResponse struct {
	Status       string `json:"status"`
	ResponseCode int    `json:"response_code"`
	ErrorMessage string `json:"error_message"`
}

func getFailureResponse(err error) FailureResponse {
	return FailureResponse{
		Status:       "Failure",
		ResponseCode: http.StatusInternalServerError,
		ErrorMessage: err.Error(),
	}
}

func getListFolderSuccessResponse(payload interface{}) SuccessResponse {
	return SuccessResponse{
		Status:       "Success",
		ResponseCode: http.StatusOK,
		Data:         payload,
	}
}

func getSuccessResponse(message string) SuccessResponse {
	return SuccessResponse{
		Status:       "Success",
		ResponseCode: http.StatusOK,
		Data:         message,
	}
}

func getSuccessResponseWithData(data interface{}) SuccessResponse {
	return SuccessResponse{
		Status:       "Success",
		ResponseCode: http.StatusOK,
		Data:         data,
	}
}
