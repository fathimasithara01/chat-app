package routes

import (
	"api-gateway/internal/proxy"
	"io"
	"net/http"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

// ProxyHandler returns a fiber.Handler that forwards requests to the given ServiceProxy.
func ProxyHandler(p *proxy.ServiceProxy, logger *zap.SugaredLogger) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Convert fiber request to http.Request
		orig := c.Request()
		// body reader
		var bodyReader io.Reader
		if orig.Body() != nil && len(orig.Body()) > 0 {
			bodyReader = io.NopCloser(io.Reader(fiber.NewReader(orig.Body())))
		} else {
			bodyReader = nil
		}

		req, err := http.NewRequest(string(orig.Header.Method()), c.OriginalURL(), bodyReader)
		if err != nil {
			logger.Errorf("failed to create proxied request: %v", err)
			return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{"error": "bad gateway"})
		}

		// copy headers
		orig.Header.VisitAll(func(k, v []byte) {
			req.Header.Set(string(k), string(v))
		})

		// forward to target
		resp, err := p.Forward(req)
		if err != nil {
			logger.Errorf("proxy forward error: %v", err)
			return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{"error": "service unavailable"})
		}
		defer resp.Body.Close()

		// copy response headers
		for k, vv := range resp.Header {
			for _, v := range vv {
				c.Set(k, v)
			}
		}
		c.Status(resp.StatusCode)

		// stream body back
		_, err = io.Copy(c, resp.Body)
		if err != nil {
			logger.Warnf("copy response body error: %v", err)
		}
		return nil
	}
}
