package users_notifications_vulnerabilities

import (
	"config-service/handlers"
	"config-service/types"
	"config-service/utils/consts"

	"github.com/gin-gonic/gin"
)

var schema = types.SchemaInfo{
	ArrayPaths: []string{"workloads", "images", "wlids"},
}

func AddRoutes(g *gin.Engine) {
	handlers.AddRoutes(g, handlers.NewRouterOptionsBuilder[*types.AggregatedVulnerability]().
		WithPath(consts.UsersNotificationsVulnerabilitiesPath).
		WithDBCollection(consts.UsersNotificationsVulnerabilitiesCollection).
		WithV2ListSearch(true).
		WithGetNamesList(false).
		WithValidatePostUniqueName(false).
		WithSchemaInfo(schema).
		Get()...)
}
