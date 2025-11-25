package router

import (
	"bytes"
	"io"
	"net/http"

	"github.com/fathima-sithara/api-gateway/internal/middleware"
	"github.com/fathima-sithara/api-gateway/internal/proxy"
	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

// convert Fiber → *http.Request
func convertToHTTPRequest(c *fiber.Ctx) (*http.Request, error) {
	bodyBytes := c.Body()
	reqBody := io.NopCloser(bytes.NewReader(bodyBytes))

	req, err := http.NewRequest(
		string(c.Request().Header.Method()),
		c.OriginalURL(),
		reqBody,
	)
	if err != nil {
		return nil, err
	}

	// Copy headers
	c.Request().Header.VisitAll(func(k, v []byte) {
		req.Header.Set(string(k), string(v))
	})

	req.URL.RawQuery = string(c.Context().URI().QueryString())
	req.RemoteAddr = c.IP()

	return req, nil
}

// convert http.Response → Fiber response
func writeHTTPResponse(c *fiber.Ctx, resp *http.Response) error {
	c.Status(resp.StatusCode)

	// copy headers
	for k, vv := range resp.Header {
		for _, v := range vv {
			c.Set(k, v)
		}
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return c.Send(body)
}

func RegisterRoutes(app *fiber.App, p *proxy.Proxy, jwt *middleware.JWTMiddleware, rl *middleware.IPRateLimiter, logger *zap.Logger) {

	// Health
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	})

	// ---------------- PUBLIC ROUTES ----------------

	app.All("/auth/*", func(c *fiber.Ctx) error {
		req, err := convertToHTTPRequest(c)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "req conversion failed"})
		}

		rr := NewFakeWriter()
		p.ServeReverseProxy(c.Context(), "auth", "/auth", rr, req)
		return writeHTTPResponse(c, rr.Response())
	})

	app.All("/media/*", func(c *fiber.Ctx) error {
		req, err := convertToHTTPRequest(c)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "req conversion failed"})
		}

		rr := NewFakeWriter()
		p.ServeReverseProxy(c.Context(), "media", "/media", rr, req)
		return writeHTTPResponse(c, rr.Response())
	})

	// ---------------- PROTECTED ROUTES ----------------
	protected := app.Group("/", jwt.Handler(), rl.Handler())

	protected.All("/user/*", func(c *fiber.Ctx) error {
		req, err := convertToHTTPRequest(c)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "req conversion failed"})
		}

		rr := NewFakeWriter()
		p.ServeReverseProxy(c.Context(), "user", "/user", rr, req)
		return writeHTTPResponse(c, rr.Response())
	})

	protected.All("/chat/*", func(c *fiber.Ctx) error {
		req, err := convertToHTTPRequest(c)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "req conversion failed"})
		}

		rr := NewFakeWriter()
		p.ServeReverseProxy(c.Context(), "chat", "/chat", rr, req)
		return writeHTTPResponse(c, rr.Response())
	})

	logger.Info("routes registered")
}
