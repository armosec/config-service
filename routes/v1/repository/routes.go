package repository

import (
	"config-service/handlers"
	"config-service/types"
	"config-service/utils/consts"

	"github.com/aws/smithy-go/ptr"
	"github.com/gin-gonic/gin"
)

func AddRoutes(g *gin.Engine) {
	//getter for short name base value for repo
	repoValueGetter := func(doc *types.Repository) string {
		return doc.RepoName
	}
	schemaInfo := types.SchemaInfo{
		TimestampFieldName: ptr.String("creationDate"),
	}

	handlers.AddRoutes(g, handlers.NewRouterOptionsBuilder[*types.Repository]().
		WithPath(consts.RepositoryPath).
		WithDBCollection(consts.RepositoryCollection).
		WithValidatePostUniqueName(true).
		WithValidatePutGUID(true).
		WithDeleteByName(false).
		WithUniqueShortName(repoValueGetter).
		WithNameQuery(consts.NameField).
		WithSchemaInfo(schemaInfo).
		WithV2ListSearch(true).
		Get()...)
}
