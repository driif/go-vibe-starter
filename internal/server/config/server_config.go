package config

import "os"

type App struct {
	Environment string
	Service
	Keycloak Keycloak
}

type Service struct {
	Port string
}

type Keycloak struct {
	URL                  string
	ISS                  string
	Realm                string
	ClientID             string
	ClientSecret         string
	ExternalClientID     string
	ExternalClientSecret string
	AuthServerURL        string
	CertURL              string
}

func DefaultServiceConfigFromEnv() App {
	environment := getEnv("APP_ENVIRONMENT", "development")
	port := getEnv("SERVICE_PORT", ":8443")
	return App{
		Environment: environment,
		Service: Service{
			Port: port,
		},
		Keycloak: Keycloak{
			URL:                  getEnv("KEYCLOAK_URL", "http://localhost:8080"),
			ISS:                  getEnv("KEYCLOAK_ISS", "http://localhost:8080/auth/realms/myrealm"),
			Realm:                getEnv("KEYCLOAK_REALM", "myrealm"),
			ClientID:             getEnv("KEYCLOAK_CLIENT_ID", "myclient"),
			ClientSecret:         getEnv("KEYCLOAK_CLIENT_SECRET", ""),
			ExternalClientID:     getEnv("KEYCLOAK_EXTERNAL_CLIENT_ID", "myexternalclient"),
			ExternalClientSecret: getEnv("KEYCLOAK_EXTERNAL_CLIENT_SECRET", ""),
			AuthServerURL:        getEnv("KEYCLOAK_AUTH_SERVER_URL", "http://localhost:8080/auth"),
			CertURL:              getEnv("KEYCLOAK_CERT_URL", "http://localhost:8080/auth/realms/myrealm/protocol/openid-connect/certs"),
		},
	}
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
