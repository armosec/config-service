package cluster

import (
	"config-service/handlers"
	"config-service/types"
	"config-service/utils/consts"

	"github.com/gin-gonic/gin"
)

func AddRoutes(g *gin.Engine) {
	handlers.AddRoutes(g, handlers.NewRouterOptionsBuilder[*types.Cluster]().
		WithPath(consts.ClusterPath).
		WithDBCollection(consts.ClustersCollection).
		WithValidatePostUniqueName(true).
		WithValidatePutGUID(true).
		WithDeleteByName(false).
		WithUniqueShortName(handlers.NameValueGetter[*types.Cluster]).
		WithNameQuery(consts.NameField).
		Get()...)
}
