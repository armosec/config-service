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
		NestedDocPath:      "relatedAlerts",
		TimestampFieldName: ptr.String("timestamp"),
		// not sure we need all the rest since we are using replace root
		ArrayPaths: []string{"relatedAlerts", "relatedResources"},
		FieldsType: map[string]types.FieldType{
			"creationTimestamp":       "date",
			"seenAt":                  "date",
			"timestamp":               "date",
			"relatedAlerts.timestamp": "date",
		},
	}

	handlers.AddRoutes(g, handlers.NewRouterOptionsBuilder[*types.RuntimeAlert]().
		WithPath(consts.RuntimeAlertPath).
		WithDBCollection(consts.RuntimeIncidentCollection).
		WithSchemaInfo(schemaInfo).
		WithV2ListSearch(true).
		Get()...)
}
