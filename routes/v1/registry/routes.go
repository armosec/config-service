package registry

import (
	"config-service/handlers"
	"config-service/types"
	"config-service/utils/consts"

	"github.com/aws/smithy-go/ptr"
	"github.com/gin-gonic/gin"
)

func AddRoutes(g *gin.Engine) {
	schemaInfo := types.SchemaInfo{
		FieldsType: map[string]types.FieldType{
			"creationTime": "date",
			"updatedTime":  "date",
		},
		TimestampFieldName: ptr.String("updatedTime"),
	}

	routerOptionsBuilder := handlers.NewRouterOptionsBuilder[*types.ContainerImageRegistry]().
		WithPath(consts.ContainerImageRegistriesPath).
		WithDBCollection(consts.ContainerImageRegistriesCollection).
		WithSchemaInfo(schemaInfo).
		WithValidatePostUniqueName(false).
		WithV2ListSearch(true)

	handlers.AddRoutes(g, routerOptionsBuilder.Get()...)
}
