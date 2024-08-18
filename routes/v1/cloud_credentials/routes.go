package cloud_credentials

import (
	"config-service/handlers"
	"config-service/types"
	"config-service/utils/consts"
	"fmt"

	"github.com/gin-gonic/gin"
)

func AddRoutes(g *gin.Engine) {
	schemaInfo := types.SchemaInfo{
		ArrayPaths: []string{"credentials.regions", "credentials.services"},
	}
	//IDO: understnd how do not update the name if it is empty
	handlers.AddRoutes(g, handlers.NewRouterOptionsBuilder[*types.CloudAccount]().
		WithPath(consts.CloudCredentialsPath).
		WithDBCollection(consts.CloudCredentialsCollection).
		WithValidatePostUniqueName(true).
		WithValidatePutGUID(true).
		WithSchemaInfo(schemaInfo).
		WithPostValidators(validatePostMustParams()).
		Get()...)
}

func validatePostMustParams() func(c *gin.Context, docs []*types.CloudAccount) ([]*types.CloudAccount, bool) {
	return func(c *gin.Context, docs []*types.CloudAccount) ([]*types.CloudAccount, bool) {
		for i := range docs {
			fmt.Println(docs[i], docs[i].Enabled)
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
