package workflows

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
			"scope", "conditions", "notifications",
			"notifications.teamsWebhookURLs", "notifications.slackChannels", "notifications.jiraTicketIdentifiers",
		},
		FieldsType: map[string]types.FieldType{
			"creationTime": "date",
		},
		TimestampFieldName: ptr.String("creationTime"),
	}

	routerOptionsBuilder := handlers.NewRouterOptionsBuilder[*types.Workflow]().
		WithPath(consts.WorkflowPath).
		WithDBCollection(consts.WorkflowCollection).
		WithSchemaInfo(schemaInfo).
		WithNameQuery(consts.PolicyNameParam).
		WithDeleteByName(true).
		WithValidatePostUniqueName(false).
		WithValidatePutGUID(true).
		WithValidatePostMandatoryName(true).
		WithV2ListSearch(true)

	handlers.AddRoutes(g, routerOptionsBuilder.Get()...)
}
