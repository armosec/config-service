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
		NestedDocPath: "notifications_config.workflows",
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
		WithServePut(false).
		WithServeDelete(false).
		WithServeGet(false).
		WithServePost(false).
		WithPath(consts.WorkflowPath).
		WithDBCollection(consts.CustomersCollection).
		WithSchemaInfo(schemaInfo).
		WithNameQuery(consts.NameField).
		WithV2ListSearch(true)
	// WithResponseSender(workflowsResponseSender)
	handlers.AddRoutes(g, routerOptionsBuilder.Get()...)
}
