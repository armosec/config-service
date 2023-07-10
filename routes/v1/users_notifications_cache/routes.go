package users_notifications_cache

import (
	"config-service/handlers"
	"config-service/types"
	"config-service/utils/consts"
	"time"

	"github.com/gin-gonic/gin"
)

const defaultTTL = time.Hour * 24 * 90 // 90 days

func AddRoutes(g *gin.Engine) {
	ttlValidator := handlers.ValidateCacheTTL(defaultTTL, 0)
	handlers.AddRoutes(g, handlers.NewRouterOptionsBuilder[*types.Cache]().
		WithPath(consts.UsersNotificationsCachePath).
		WithDBCollection(consts.UsersNotificationsCacheCollection).
		WithV2ListSearch(true).
		WithGetNamesList(false).
		WithValidatePostUniqueName(false).
		WithPostValidators(ttlValidator).
		WithPutValidators(ttlValidator).
		Get()...)
}
