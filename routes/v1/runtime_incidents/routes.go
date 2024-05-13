package runtime_incidents

import (
	"config-service/handlers"
	"config-service/types"
	"time"

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
			"creationDayDate":         "date",
			"resolveDayDate":          "date",
		},
		TimestampFieldName: ptr.String("creationTimestamp"),
		MustExcludeFields:  []string{"relatedAlerts", "creationDayDate", "resolveDayDate"},
	}

	handlers.AddRoutes(g, handlers.NewRouterOptionsBuilder[*types.RuntimeIncident]().
		WithPath(consts.RuntimeIncidentPath).
		WithDBCollection(consts.RuntimeIncidentCollection).
		WithGetNamesList(false).
		WithValidatePostUniqueName(false).
		WithValidatePostMandatoryName(false).
		WithPutValidators(incidentUpdateResolveDayDate).
		WithSchemaInfo(schemaInfo).
		WithV2ListSearch(true).
		Get()...)
}

func incidentUpdateResolveDayDate(c *gin.Context, docs []*types.RuntimeIncident) (verifiedDocs []*types.RuntimeIncident, valid bool) {
	for _, doc := range docs {
		if doc.IsDismissed && doc.ResolveDayDate == nil {
			nowUtc := time.Now().UTC()
			nowUtc = time.Date(nowUtc.Year(), nowUtc.Month(), nowUtc.Day(), 0, 0, 0, 0, time.UTC)
			doc.ResolveDayDate = &nowUtc
		}
	}
	return docs, true
}
