package types

import (
	"time"

	"github.com/armosec/armoapi-go/armotypes"
)

type AwsCredentials struct {
	EncryptedExternalKey string       `json:"encryptedExternalKey" bson:"encryptedExternalKey"`
	EncryptedRoleARN     string       `json:"encryptedRoleARN" bson:"encryptedRoleARN"`
	Regions              []string     `json:"regions" bson:"regions"`
	Services             []AwsService `json:"services" bson:"services"`
}

type AwsService struct {
	Name string `json:"name" bson:"name"`
}

type AzureCredentials struct {
	EncryptedClientID     string `json:"encryptedClientID" bson:"encryptedClientID"`
	EncryptedClientSecret string `json:"encryptedClientSecret" bson:"encryptedClientSecret"`
	EncryptedTenantID     string `json:"encryptedTenantID" bson:"encryptedTenantID"`
	SubscriptionID        string `json:"subscriptionID" bson:"subscriptionID"`
}

type Credentials struct {
	AwsCredentials   `json:",inline" bson:"awsCredentials,omitempty"`
	AzureCredentials `json:",inline" bson:"azureCredentials,omitempty"`
}

type CloudCredentials struct {
	armotypes.PortalBase `json:",inline" bson:"inline"`
	Enabled              bool        `json:"enabled" bson:"enabled"`
	CreationTime         string      `json:"creationTime" bson:"creationTime"`
	Provider             string      `json:"provider" bson:"provider"`
	Credentials          Credentials `json:"credentials" bson:"credentials"`
}

func (cc *CloudCredentials) GetReadOnlyFields() []string {
	return clusterReadOnlyFields
}
func (cc *CloudCredentials) InitNew() {
	cc.CreationTime = time.Now().UTC().Format(time.RFC3339)
	if cc.Attributes == nil {
		cc.Attributes = make(map[string]interface{})
	}
}

func (cc *CloudCredentials) GetCreationTime() *time.Time {
	if cc.CreationTime == "" {
		return nil
	}
	creationTime, err := time.Parse(time.RFC3339, cc.CreationTime)
	if err != nil {
		return nil
	}
	return &creationTime
}
