package api

import (
	"context"
	"errors"
	"github.com/labstack/echo/v4"
	"gomodlag/internal/storage"
	"net/http"
	"time"
)

type apiError struct {
	Code int    `json:"code,omitempty"`
	Text string `json:"text,omitempty"`
}
type ApiResp struct {
	Error    *apiError   `json:"error,omitempty"`
	Response interface{} `json:"response,omitempty"`
	Data     interface{} `json:"data,omitempty"`
}

const Invalid = "invalid data"
const invalidToken = "invalid token"

func Ok(c echo.Context, resp interface{}, data interface{}) error {
	return c.JSON(http.StatusOK, ApiResp{Response: resp, Data: data})
}
func BadReq(c echo.Context, msg string) error {
	return c.JSON(http.StatusBadRequest, ApiResp{Error: &apiError{Code: 400, Text: msg}})
}
func unauth(c echo.Context) error {
	return c.JSON(http.StatusUnauthorized, ApiResp{Error: &apiError{Code: 401, Text: "unauthorized"}})
}
func norute(c echo.Context, msg string) error {
	return c.JSON(http.StatusForbidden, ApiResp{Error: &apiError{Code: 403, Text: msg}})
}
func notImpl(c echo.Context) error {
	return c.JSON(http.StatusNotImplemented, ApiResp{Error: &apiError{Code: 501, Text: "not implemented"}})
}
func somewrong(c echo.Context) error {
	return c.JSON(http.StatusNotImplemented, ApiResp{Error: &apiError{Code: 500, Text: "something went wrong"}})
}

func AuthTokenRequired(db storage.TokenValidator) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			var body map[string]interface{}
			if err := c.Bind(&body); err != nil {
				return BadReq(c, Invalid)
			}
			tokenRaw, ok := body["token"]
			if !ok {
				return unauth(c)
			}

			tokenStr, ok := tokenRaw.(string)
			if !ok || tokenStr == "" {
				return unauth(c)
			}
			ctx, cancel := context.WithTimeout(c.Request().Context(), time.Second*2)
			defer cancel()
			userid, err := db.ValidateToken(ctx, tokenStr)
			if err != nil {
				cancel()
				if errors.Is(err, storage.Invalidtoken) {
					return BadReq(c, invalidToken)
				} else {
					return somewrong(c)
				}
			}
			c.Set("userid", userid)
			return next(c)

		}
	}
}
func AddContext(timectx time.Duration) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			ctx, cancel := context.WithTimeout(c.Request().Context(), time.Second*timectx)
			defer cancel()
			newctx := c.Request().WithContext(ctx)
			c.SetRequest(newctx)

			err := next(c)
			if ctx.Err() != nil {
				switch {
				case errors.Is(ctx.Err(), context.DeadlineExceeded):
					return c.JSON(http.StatusGatewayTimeout, ApiResp{
						Error: &apiError{
							Code: 504,
							Text: "request timed out",
						},
					})
				case errors.Is(ctx.Err(), context.Canceled):
					return c.JSON(http.StatusRequestTimeout, ApiResp{
						Error: &apiError{
							Code: 408,
							Text: "request canceled",
						},
					})
				}
			}
			return err
		}
	}
}
