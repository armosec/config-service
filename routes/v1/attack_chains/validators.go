package attack_chains

import (
	"config-service/handlers"
	"config-service/types"

	"github.com/gin-gonic/gin"
)

func validateAttackChainId(c *gin.Context, docs []*types.AttackChain) ([]*types.AttackChain, bool) {
	for i := range docs {
		if docs[i].AttackChainID == "" {
			handlers.ResponseBadRequest(c, "Attack Chain must contain AttackChainID")
			return nil, false
		}
	}
	return docs, true
}
