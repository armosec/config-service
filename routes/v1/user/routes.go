package user

import (
	"config-service/handlers"
	"config-service/types"
	"config-service/utils/consts"

	"github.com/gin-gonic/gin"
)

func AddRoutes(g *gin.Engine) {
	handlers.AddRoutes(g, handlers.NewRouterOptionsBuilder[*types.User]().
		WithPath(consts.UserPath).
		WithDBCollection(consts.UserCollection).
		WithGetNamesList(false).
		WithServeGetWithGUIDOnly(true).
		WithValidatePostUniqueName(false).
		Get()...)
}
