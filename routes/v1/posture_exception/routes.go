package posture_exception

import (
	"config-service/handlers"
	"config-service/types"
	"config-service/utils/consts"

	"github.com/gin-gonic/gin"
)

func AddRoutes(g *gin.Engine) {
	queryParamsConfig := handlers.DefaultQueryConfig()
	queryParamsConfig.Params2Query["scope"] = handlers.QueryConfig{
		FieldName:   "resources",
		PathInArray: "attributes",
		IsArray:     true,
	}
	queryParamsConfig.Params2Query["resources"] = handlers.QueryConfig{
		FieldName:   "resources",
		PathInArray: "",
		IsArray:     true,
	}
	queryParamsConfig.Params2Query["posturePolicies"] = handlers.QueryConfig{
		FieldName:   "posturePolicies",
		PathInArray: "",
		IsArray:     true,
	}
	handlers.AddPolicyRoutes[*types.PostureExceptionPolicy](g,
		consts.PostureExceptionPolicyPath,
		consts.PostureExceptionPolicyCollection, queryParamsConfig, false)
}
