package repository

import (
	"config-service/handlers"
	"config-service/types"
	"config-service/utils/consts"

	"github.com/gin-gonic/gin"
)

func AddRoutes(g *gin.Engine) {
	//getter for short name base value for repo
	repoValueGetter := func(doc *types.Repository) string {
		return doc.RepoName
	}

	handlers.AddRoutes(g, handlers.NewRouterOptionsBuilder[*types.Repository]().
		WithPath(consts.RepositoryPath).
		WithDBCollection(consts.RepositoryCollection).
		WithValidatePostUniqueName(true).
		WithValidatePutGUID(true).
		WithDeleteByName(false).
		WithUniqueShortName(repoValueGetter).
		WithV2ListSearch(true).
		Get()...)
}
