package cloud_credentials

import (
	"config-service/handlers"
	"config-service/types"
	"config-service/utils/consts"

	"github.com/gin-gonic/gin"
)

func AddRoutes(g *gin.Engine) {
	schemaInfo := types.SchemaInfo{
		ArrayPaths: []string{"credentials.regions", "credentials.services"},
	}
	handlers.AddRoutes(g, handlers.NewRouterOptionsBuilder[*types.CloudAccount]().
		WithPath(consts.CloudAccountPath).
		WithDBCollection(consts.CloudAccountsCollection).
		WithValidatePostUniqueName(true).
		WithDeleteByName(true).
		WithValidatePutGUID(true).
		WithSchemaInfo(schemaInfo).
		WithPostValidators(validatePostMustParams()).
		WithV2ListSearch(true).
		WithNameQuery(consts.NameField).
		Get()...)
}

func validatePostMustParams() func(c *gin.Context, docs []*types.CloudAccount) ([]*types.CloudAccount, bool) {
	return func(c *gin.Context, docs []*types.CloudAccount) ([]*types.CloudAccount, bool) {
		for i := range docs {
			if docs[i].Provider == "" {
				return docs, false
			}
			if docs[i].AccountID == "" {
				return docs, false
			}
			if docs[i].Name == "" {
				return docs, false
			}
			if docs[i].Enabled == nil {
				return docs, false
			}
		}
		return docs, true
	}
}
