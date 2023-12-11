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
	ttlValidator := validateCacheTTL(defaultTTL, 0)
	handlers.AddRoutes(g, handlers.NewRouterOptionsBuilder[*types.Cache]().
		WithPath(consts.UsersNotificationsCachePath).
		WithDBCollection(consts.UsersNotificationsCacheCollection).
		WithV2ListSearch(true).
		WithGetNamesList(false).
		WithValidatePostUniqueName(false).
		WithPostValidators(ttlValidator).
		WithPutValidators(ttlValidator).
		WithSchemaInfo(types.SchemaInfo{
			FieldsType: map[string]types.FieldType{
				"expiryTime": types.Date,
			},
		}).
		Get()...)
}

func validateCacheTTL(defaultTTL, maxTTL time.Duration) func(c *gin.Context, docs []*types.Cache) ([]*types.Cache, bool) {
	return func(c *gin.Context, docs []*types.Cache) ([]*types.Cache, bool) {
		for i := range docs {
			if docs[i].ExpiryTime.IsZero() {
				docs[i].SetTTL(defaultTTL)
			} else if maxTTL > 0 && docs[i].ExpiryTime.Sub(time.Now()) > maxTTL {
				docs[i].SetTTL(maxTTL)
			}
		}
		return docs, true
	}
}
