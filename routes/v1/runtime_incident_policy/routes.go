package runtime_incident_policy

import (
	"config-service/handlers"
	"config-service/types"
	"config-service/utils/consts"

	"github.com/aws/smithy-go/ptr"
	"github.com/gin-gonic/gin"
)

func AddRoutes(g *gin.Engine) {
	schemaInfo := types.SchemaInfo{
		ArrayPaths: []string{
			"notifications", "actions", "scope.riskFactors",
			"scope.designators", "incidentTypeIDs", "managedRuleSetIDs",
		},
		FieldsType: map[string]types.FieldType{
			"creationTime": "date",
		},
		TimestampFieldName: ptr.String("creationTime"),
	}

	routerOptionsBuilder := handlers.NewRouterOptionsBuilder[*types.IncidentPolicy]().
		WithPath(consts.RuntimeIncidentPolicyPath).
		WithDBCollection(consts.RuntimeIncidentPolicyCollection).
		WithSchemaInfo(schemaInfo).
		WithNameQuery(consts.PolicyNameParam).
		WithDeleteByName(true).
		WithValidatePostUniqueName(false).
		WithValidatePutGUID(true).
		WithValidatePostMandatoryName(true).
		WithV2ListSearch(true)

	handlers.AddRoutes(g, routerOptionsBuilder.Get()...)
}
