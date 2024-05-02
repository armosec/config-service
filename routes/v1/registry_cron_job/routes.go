package registry_cron_job

import (
	"config-service/handlers"
	"config-service/types"
	"config-service/utils/consts"

	"github.com/aws/smithy-go/ptr"
	"github.com/gin-gonic/gin"
)

func AddRoutes(g *gin.Engine) {
	schemaInfo := types.SchemaInfo{
		TimestampFieldName: ptr.String("updatedTime"),
	}
	handlers.AddRoutes(g, handlers.NewRouterOptionsBuilder[*types.RegistryCronJob]().
		WithPath(consts.RegistryCronJobPath).
		WithDBCollection(consts.RegistryCronJobCollection).
		WithValidatePostUniqueName(true).
		WithValidatePutGUID(true).
		WithDeleteByName(true).
		WithNameQuery(consts.NameField).
		WithSchemaInfo(schemaInfo).
		WithQueryConfig(handlers.FlatQueryConfig()).
		Get()...)
}
