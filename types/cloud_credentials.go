package types

import (
	"time"

	"github.com/armosec/armoapi-go/armotypes"
)

type AwsService struct {
	Name string `json:"name" bson:"name"`
}

type AwsCredentials struct {
	EncryptedExternalId string       `json:"encryptedExternalKey,omitempty" bson:"encryptedExternalKey,omitempty"`
	EncryptedRoleARN    string       `json:"encryptedRoleARN,omitempty" bson:"encryptedRoleARN,omitempty"`
	Regions             []string     `json:"regions,omitempty" bson:"regions,omitempty"`
	Services            []AwsService `json:"services,omitempty" bson:"services,omitempty"`
}

type AzureCredentials struct {
	EncryptedTenantID       string `json:"encryptedTenantID,omitempty" bson:"encryptedTenantID,omitempty"`
	EncryptedSubscriptionID string `json:"encryptedSubscriptionID,omitempty" bson:"encryptedSubscriptionID,omitempty"`
}

type GcpCredentials struct {
	EncryptedPrincipalID string `json:"encryptedPrincipalID,omitempty" bson:"encryptedPrincipalID,omitempty"`
}

type Credentials struct {
	AwsCredentials   `json:",inline,omitempty" bson:"awsCredentials,omitempty"`
	AzureCredentials `json:",inline,omitempty" bson:"azureCredentials,omitempty"`
	GcpCredentials   `json:",inline,omitempty" bson:"gcpCredentials,omitempty"`
}

type CloudCredentials struct {
	armotypes.PortalBase `json:",inline" bson:"inline"`
	UpdatedBy            string      `json:"updatedBy" bson:"updatedBy"`
	Enabled              bool        `json:"enabled" bson:"enabled"`
	CreationTime         string      `json:"creationTime" bson:"creationTime"`
	Provider             string      `json:"provider" bson:"provider"`
	Credentials          Credentials `json:"credentials" bson:"credentials"`
	AccountID            string      `json:"accountID" bson:"accountID"`
}

func (cc *CloudCredentials) GetReadOnlyFields() []string {
	return CloudCredentialsReadOnlyFields
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

var CloudCredentialsReadOnlyFields = append([]string{"Provider"}, commonReadOnlyFieldsAllowRename...)
