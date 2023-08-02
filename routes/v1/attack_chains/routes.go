package attack_chains

import (
	"config-service/handlers"
	"config-service/types"
	"config-service/utils/consts"

	"github.com/gin-gonic/gin"
)

func AddRoutes(g *gin.Engine) {
	handlers.AddRoutes(g, handlers.NewRouterOptionsBuilder[*types.AttackChain]().
		WithPath(consts.AttackChainsPath).
		WithDBCollection(consts.AttackChainsCollection).
		WithV2ListSearch(true).
		WithGetNamesList(false).
		WithValidatePostUniqueName(false).
		WithValidatePostMandatoryName(true).
		WithPostValidators(validateAttackChainId).
		Get()...)
}
