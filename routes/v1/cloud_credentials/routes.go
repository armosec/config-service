package cloud_credentials

import (
	"config-service/handlers"
	"config-service/types"
	"config-service/utils/consts"

	"github.com/gin-gonic/gin"
)

func AddRoutes(g *gin.Engine) {
	handlers.AddRoutes(g, handlers.NewRouterOptionsBuilder[*types.CloudCredentials]().
		WithPath(consts.CloudCredentialsPath).
		WithDBCollection(consts.CloudCredentialsCollection).
		WithValidatePostUniqueName(true).
		WithValidatePutGUID(true).
		WithDeleteByName(false).
		WithUniqueShortName(handlers.NameValueGetter[*types.CloudCredentials]).
		WithV2ListSearch(true).
		WithNameQuery(consts.NameField).
		Get()...)
}
