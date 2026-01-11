package main

import (
	"fmt"
	"os"
	"strconv"
)

type config struct {
	postgresHost     string
	postgresPort     int
	postgresDB       string
	postgresUser     string
	postgresPassword string

	httpPort int
}

func loadConfigFromEnv() config {
	return config{
		postgresHost:     getEnv("POSTGRES_HOST", "localhost"),
		postgresPort:     parseIntOr(getEnv("POSTGRES_PORT", "5432"), 5432),
		postgresDB:       getEnv("POSTGRES_DB", "project-sem-1"),
		postgresUser:     getEnv("POSTGRES_USER", "validator"),
		postgresPassword: getEnv("POSTGRES_PASSWORD", "val1dat0r"),
		httpPort:         parseIntOr(getEnv("PORT", "8080"), 8080),
	}
}

func (c config) httpAddr() string {
	return fmt.Sprintf(":%d", c.httpPort)
}

func (c config) postgresConnString() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		c.postgresHost, c.postgresPort, c.postgresUser, c.postgresPassword, c.postgresDB,
	)
}

func getEnv(key, fallback string) string {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	return v
}

func parseIntOr(s string, fallback int) int {
	v, err := strconv.Atoi(s)
	if err != nil {
		return fallback
	}
	return v
}
