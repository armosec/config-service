package utils

import (
	"os"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOverrideConfigFromEnvVars(t *testing.T) {
	os.Setenv(ConfigPathEnvVar, "../config.json")
	defer os.Unsetenv(ConfigPathEnvVar)

	// config read from cofig.json file using env var
	initOnce = sync.Once{}
	config := GetConfig()
	assert.Equal(t, "admin", config.Mongo.User)
	assert.Equal(t, "admin", config.Mongo.Password)

	// override config from env vars
	expectedUser := "override-user"
	expectedPassword := "override-password"
	os.Setenv(MongoDbUserEnvVar, expectedUser)
	os.Setenv(MongoDbPasswordEnvVar, expectedPassword)
	defer os.Unsetenv(MongoDbUserEnvVar)
	defer os.Unsetenv(MongoDbPasswordEnvVar)

	// test config read from env vars
	initOnce = sync.Once{}
	config = GetConfig()
	assert.Equal(t, expectedUser, config.Mongo.User)
	assert.Equal(t, expectedPassword, config.Mongo.Password)

	// reset singleton
	initOnce = sync.Once{}
}
