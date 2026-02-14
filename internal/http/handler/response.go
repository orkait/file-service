package handler

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

func respondError(c echo.Context, status int, message string) error {
	return c.JSON(status, map[string]string{jsonKeyError: message})
}

func respondMessage(c echo.Context, status int, message string) error {
	return c.JSON(status, map[string]string{jsonKeyMessage: message})
}

func handleHTTPError(c echo.Context, err error) error {
	if he, ok := err.(*echo.HTTPError); ok {
		msg, _ := he.Message.(string)
		if msg == "" {
			msg = http.StatusText(he.Code)
		}
		return respondError(c, he.Code, msg)
	}

	return respondError(c, http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError))
}
