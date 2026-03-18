package keycloak

import (
	"strings"

	"github.com/labstack/echo/v4"
)

// Enrich Echo Context with Keycloak Realm
func EnrichEchoContextWithRealm(c echo.Context) echo.Context {
	origin := c.Request().Header.Get("Origin")
	// remove http:// or https://
	if origin != "" {
		origin = strings.TrimPrefix(origin, "http://")
		origin = strings.TrimPrefix(origin, "https://")
		origin = strings.Split(origin, ".")[0]
		//if origin contains a double point, that split origin
		if strings.Contains(origin, ":") {
			origin = strings.Split(origin, ":")[0]
		}
	} else {
		origin = "localhost"
	}

	c.Set("realm", origin)

	return c
}
