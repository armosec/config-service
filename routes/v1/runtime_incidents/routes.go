package runtime_incidents

import (
	"config-service/handlers"
	"config-service/types"

	"config-service/utils/consts"

	"github.com/aws/smithy-go/ptr"
	"github.com/gin-gonic/gin"
)

func AddRoutes(g *gin.Engine) {
	schemaInfo := types.SchemaInfo{
		ArrayPaths: []string{"relatedAlerts", "relatedResources"},
		FieldsType: map[string]types.FieldType{
			"creationTimestamp":       "date",
			"seenAt":                  "date",
			"timestamp":               "date",
			"relatedAlerts.timestamp": "date",
		},
		TimestampFieldName: ptr.String("creationTimestamp"),
		MustExcludeFields:  []string{"relatedAlerts"},
	}
	guardRelatedAlerts := func(c *gin.Context, docs []*types.RuntimeIncident) (verifiedDocs []*types.RuntimeIncident, valid bool) {
		for i := range docs {
			if docs[i].RelatedAlerts != nil {
				docs[i].RelatedAlerts = nil
			}
		}
		return docs, true
	}

	handlers.AddRoutes(g, handlers.NewRouterOptionsBuilder[*types.RuntimeIncident]().
		WithPath(consts.RuntimeIncidentPath).
		WithDBCollection(consts.RuntimeIncidentCollection).
		WithGetNamesList(false).
		WithValidatePostUniqueName(false).
		WithValidatePostMandatoryName(false).
		WithSchemaInfo(schemaInfo).
		WithV2ListSearch(true).
		WithPutValidators(guardRelatedAlerts).
		Get()...)
}
