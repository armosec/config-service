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
		WithPath(consts.CloudCredentialsPath).
		WithDBCollection(consts.CloudCredentialsCollection).
		WithSchemaInfo(schemaInfo).
		WithValidatePostUniqueName(true).
		WithValidatePutGUID(true).
		WithDeleteByName(false).
		WithUniqueShortName(handlers.NameValueGetter[*types.CloudAccount]).
		WithV2ListSearch(true).
		WithNameQuery(consts.NameField).
		Get()...)
}
