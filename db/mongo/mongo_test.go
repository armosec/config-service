package mongo

import (
	"config-service/utils"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOverrideConfigFromEnvVars(t *testing.T) {
	expectedUser := "testuser"
	expectedPassword := "testpassword"
	os.Setenv(MongoDbUserEnvVar, expectedUser)
	os.Setenv(MongoDbPasswordEnvVar, expectedPassword)

	config := utils.MongoConfig{
		Host:       "localhost",
		Port:       "27017",
		User:       "user",
		Password:   "password",
		DB:         "db",
		ReplicaSet: "rs",
	}

	OverrideConfigFromEnvVars(&config)

	// assert
	assert.Equal(t, expectedUser, config.User)
	assert.Equal(t, expectedPassword, config.Password)

	// cleanup
	os.Unsetenv(MongoDbUserEnvVar)
	os.Unsetenv(MongoDbPasswordEnvVar)
}
