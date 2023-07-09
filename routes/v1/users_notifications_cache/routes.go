package users_notifications_cache

import (
	"config-service/handlers"
	"config-service/types"
	"config-service/utils/consts"

	"github.com/gin-gonic/gin"
)

func AddRoutes(g *gin.Engine) {
	handlers.AddRoutes(g, handlers.NewRouterOptionsBuilder[*types.Cache]().
		WithPath(consts.UsersNotificationsCachePath).
		WithDBCollection(consts.UsersNotificationsCacheCollection).
		WithGetNamesList(false).
		WithValidatePostUniqueName(false).
		Get()...)
}
