package cluster

import (
	"config-service/handlers"
	"config-service/types"
	"config-service/utils/consts"

	"github.com/aws/smithy-go/ptr"
	"github.com/gin-gonic/gin"
)

func AddRoutes(g *gin.Engine) {
	schemaInfo := types.SchemaInfo{
		TimestampFieldName: ptr.String("subscription_date"),
	}
	handlers.AddRoutes(g, handlers.NewRouterOptionsBuilder[*types.Cluster]().
		WithPath(consts.ClusterPath).
		WithDBCollection(consts.ClustersCollection).
		WithValidatePostUniqueName(true).
		WithValidatePutGUID(true).
		WithDeleteByName(false).
		WithUniqueShortName(handlers.NameValueGetter[*types.Cluster]).
		WithV2ListSearch(true).
		WithNameQuery(consts.NameField).
		WithSchemaInfo(schemaInfo).
		Get()...)
}
