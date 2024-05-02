package runtime_alerts

import (
	"config-service/handlers"
	"config-service/types"

	"config-service/utils/consts"

	"github.com/aws/smithy-go/ptr"
	"github.com/gin-gonic/gin"
)

func AddRoutes(g *gin.Engine) {
	schemaInfo := types.SchemaInfo{
		NestedDocPath: "relatedAlerts",
		ArrayPaths:    []string{"relatedAlerts", "relatedResources"},
		FieldsType: map[string]types.FieldType{
			"creationTimestamp":       "date",
			"seenAt":                  "date",
			"timestamp":               "date",
			"relatedAlerts.timestamp": "date",
		},
		TimestampFieldName: ptr.String("creationTimestamp"),
		MustExcludeFields:  []string{"relatedAlerts"},
	}

	handlers.AddRoutes(g, handlers.NewRouterOptionsBuilder[*types.RuntimeAlert]().
		WithPath(consts.RuntimeAlertPath).
		WithDBCollection(consts.RuntimeIncidentCollection).
		WithSchemaInfo(schemaInfo).
		WithV2ListSearch(true).
		Get()...)
}
