package handler

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
)

const (
	contentTypeJSON          = "application/json"
	maxStrictBodyBytes int64 = 1 << 20 // Keep parser bound aligned with global body limit.
)

func bindStrictJSON(c echo.Context, dst interface{}) error {
	if !strings.HasPrefix(strings.ToLower(c.Request().Header.Get(echo.HeaderContentType)), contentTypeJSON) {
		return echo.NewHTTPError(http.StatusUnsupportedMediaType, msgContentTypeJSONRequired)
	}

	body := io.LimitReader(c.Request().Body, maxStrictBodyBytes)
	decoder := json.NewDecoder(body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(dst); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, msgInvalidRequestBody)
	}

	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return echo.NewHTTPError(http.StatusBadRequest, msgInvalidRequestBody)
	}

	return nil
}
