package integration_reference

import (
	"config-service/handlers"
	"config-service/types"

	"config-service/utils/consts"

	"github.com/gin-gonic/gin"
)

func AddRoutes(g *gin.Engine) {
	schemaInfo := types.SchemaInfo{
		ArrayPaths: []string{"relatedObjects"},
		FieldsType: map[string]types.FieldType{
			"creationTime": "date",
		},
	}

	handlers.AddRoutes(g, handlers.NewRouterOptionsBuilder[*types.IntegrationReference]().
		WithPath(consts.IntegrationReferencePath).
		WithDBCollection(consts.IntegrationReferenceCollection).
		WithGetNamesList(false).
		WithValidatePostUniqueName(false).
		WithValidatePostMandatoryName(false).
		WithSchemaInfo(schemaInfo).
		WithV2ListSearch(true).
		Get()...)
}
