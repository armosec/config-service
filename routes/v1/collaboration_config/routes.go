package collaboration_config

import (
	"config-service/handlers"
	"config-service/types"
	"config-service/utils/consts"

	"github.com/gin-gonic/gin"
)

func AddRoutes(g *gin.Engine) {
	handlers.AddPolicyRoutes[*types.CollaborationConfig](g,
		consts.CollaborationConfigPath,
		consts.CollaborationConfigCollection, handlers.FlatQueryConfig())
}
