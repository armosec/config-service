package handlers

import (
	"config-service/types"
	"config-service/utils/consts"
	"fmt"

	"github.com/gin-gonic/gin"
)

// router options
type routerOptions[T types.DocContent] struct {
	dbCollection              string                //mandatory db collection name
	path                      string                //mandatory uri path
	serveGet                  bool                  //default true, serve GET /<path> to get all documents and GET /<path>/<GUID> to get document by GUID
	serveGetNamesList         bool                  //default true, GET will return all documents names if "list" query param exist
	serveGetWithGUIDOnly      bool                  //default false, GET will return the document by GUID only
	servePost                 bool                  //default true, serve POST
	servePut                  bool                  //default true, serve PUT /<path> to update document by GUID in body and PUT /<path>/<GUID> to update document by GUID in path
	serveDelete               bool                  //default true, serve DELETE  /<path>/<GUID> to delete document by GUID in path
	validatePostUniqueName    bool                  //default true, POST will validate that the name is unique
	validatePutGUID           bool                  //default true, PUT will validate GUID existence in body or path
	nameQueryParam            string                //default empty, the param name that indicates query by name (e.g. clusterName) when set GET will check for this param and will return the document by name
	QueryConfig               *QueryParamsConfig    //default nil, when set, GET will check for the specified query params and will return the documents by the query params
	serveGetIncludeGlobalDocs bool                  //default false, when true, in GET all the response will include global documents (with customers[""])
	serveDeleteByName         bool                  //default false, when true, DELETE will check for name param and will delete the document by name
	uniqueShortName           func(T) string        //default nil, when set, POST will create a unique short name (aka "alias") attribute from the value returned from the function & Put will validate that the short name is not deleted
	putValidators             []MutatorValidator[T] //default nil, when set, PUT will call the mutators/validators before updating the document
	postValidators            []MutatorValidator[T] //default nil, when set, POST will call the mutators/validators before creating the document
	bodyDecoder               BodyDecoder[T]        //default nil, when set, replace the default body decoder
	responseSender            ResponseSender[T]     //default nil, when set, replace the default response sender
	putFields                 []string              //default nil, when set, PUT will update only the specified fields
}

func newRouterOptions[T types.DocContent]() *routerOptions[T] {
	return &routerOptions[T]{
		serveGet:                  true,
		servePost:                 true,
		servePut:                  true,
		serveDelete:               true,
		validatePostUniqueName:    true,
		validatePutGUID:           true,
		serveGetNamesList:         true,
		serveGetIncludeGlobalDocs: false,
		serveDeleteByName:         false,
	}
}

func AddRoutes[T types.DocContent](g *gin.Engine, options ...RouterOption[T]) *gin.RouterGroup {
	opts := newRouterOptions[T]()
	opts.apply(options)
	if err := opts.validate(); err != nil {
		panic(err)
	}
	routerGroup := g.Group(opts.path)
	//add middleware
	routerGroup.Use(DBContextMiddleware(opts.dbCollection))
	if opts.responseSender != nil {
		routerGroup.Use(ResponseSenderContextMiddleware(&opts.responseSender))
	}
	if opts.bodyDecoder != nil {
		routerGroup.Use(BodyDecoderContextMiddleware(&opts.bodyDecoder))
	}
	if opts.putFields != nil {
		routerGroup.Use(PutFieldsContextMiddleware(opts.putFields))
	}

	//add routes
	if opts.serveGet {
		if !opts.serveGetWithGUIDOnly {
			routerGroup.GET("", HandleGet(opts))
		}
		routerGroup.GET("/:"+consts.GUIDField, HandleGetDocWithGUIDInPath[T])
	}
	if opts.servePost {
		postValidators := []MutatorValidator[T]{}
		if opts.validatePostUniqueName {
			postValidators = append(postValidators, ValidateUniqueValues(NameKeyGetter[T]))
		}
		if opts.uniqueShortName != nil {
			postValidators = append(postValidators, ValidatePostAttributeShortName(opts.uniqueShortName))
		}
		postValidators = append(postValidators, opts.postValidators...)
		routerGroup.POST("", HandlePostDocWithValidation(postValidators...)...)
	}
	if opts.servePut {
		putValidators := []MutatorValidator[T]{}
		if opts.validatePutGUID {
			putValidators = append(putValidators, ValidateGUIDExistence[T])
		}
		if opts.uniqueShortName != nil {
			putValidators = append(putValidators, ValidatePutAttributerShortName[T])
		}
		putValidators = append(putValidators, opts.putValidators...)
		routerGroup.PUT("", HandlePutDocWithValidation(putValidators...)...)
		routerGroup.PUT("/:"+consts.GUIDField, HandlePutDocWithValidation(putValidators...)...)
	}
	if opts.serveDelete {
		if opts.serveDeleteByName {
			routerGroup.DELETE("", HandleDeleteDocByName[T](opts.nameQueryParam))
		}
		routerGroup.DELETE("/:"+consts.GUIDField, HandleDeleteDoc[T])
	}
	return routerGroup
}

// Common router config for policies
func AddPolicyRoutes[T types.DocContent](g *gin.Engine, path, dbCollection string, paramConf *QueryParamsConfig) *gin.RouterGroup {
	return AddRoutes(g, NewRouterOptionsBuilder[T]().
		WithPath(path).
		WithDBCollection(dbCollection).
		WithNameQuery(consts.PolicyNameParam).
		WithQueryConfig(paramConf).
		WithIncludeGlobalDocs(true).
		WithDeleteByName(true).
		WithValidatePostUniqueName(true).
		WithValidatePutGUID(true).
		Get()...)
}

func (opts *routerOptions[T]) apply(options []RouterOption[T]) {
	for _, option := range options {
		option(opts)
	}
}

func (opts *routerOptions[T]) validate() error {
	if opts.dbCollection == "" || opts.path == "" {
		return fmt.Errorf("dbCollection and path must be set")
	}
	if opts.serveGetIncludeGlobalDocs && !opts.serveGet {
		return fmt.Errorf("serveGetIncludeGlobalDocs can only be true when serveGet is true")
	}
	if opts.serveDeleteByName && !opts.serveDelete {
		return fmt.Errorf("serveDeleteByName can only be true when serveDelete is true")
	}
	if opts.uniqueShortName != nil && (!opts.servePost || !opts.servePut) {
		return fmt.Errorf("uniqueShortName can only be set when servePost and servePut are true")
	}
	return nil
}

type RouterOption[T types.DocContent] func(*routerOptions[T])

type RouterOptionsBuilder[T types.DocContent] struct {
	options []RouterOption[T]
}

func NewRouterOptionsBuilder[T types.DocContent]() *RouterOptionsBuilder[T] {
	return &RouterOptionsBuilder[T]{options: []RouterOption[T]{}}
}

func (b *RouterOptionsBuilder[T]) Get() []RouterOption[T] {
	return b.options
}

func (b *RouterOptionsBuilder[T]) WithPutFields(fields []string) *RouterOptionsBuilder[T] {
	b.options = append(b.options, func(opts *routerOptions[T]) {
		opts.putFields = fields
	})
	return b
}

func (b *RouterOptionsBuilder[T]) WithBodyDecoder(decoder BodyDecoder[T]) *RouterOptionsBuilder[T] {
	b.options = append(b.options, func(opts *routerOptions[T]) {
		opts.bodyDecoder = decoder
	})
	return b
}

func (b *RouterOptionsBuilder[T]) WithResponseSender(sender ResponseSender[T]) *RouterOptionsBuilder[T] {
	b.options = append(b.options, func(opts *routerOptions[T]) {
		opts.responseSender = sender
	})
	return b
}

func (b *RouterOptionsBuilder[T]) WithDBCollection(dbCollection string) *RouterOptionsBuilder[T] {
	b.options = append(b.options, func(opts *routerOptions[T]) {
		opts.dbCollection = dbCollection
	})
	return b
}

func (b *RouterOptionsBuilder[T]) WithPath(path string) *RouterOptionsBuilder[T] {
	b.options = append(b.options, func(opts *routerOptions[T]) {
		opts.path = path
	})
	return b
}

func (b *RouterOptionsBuilder[T]) WithServeGet(serveGet bool) *RouterOptionsBuilder[T] {
	b.options = append(b.options, func(opts *routerOptions[T]) {
		opts.serveGet = serveGet
	})
	return b
}

func (b *RouterOptionsBuilder[T]) WithServeGetWithGUIDOnly(serveGetIncludeGlobalDocs bool) *RouterOptionsBuilder[T] {
	b.options = append(b.options, func(opts *routerOptions[T]) {
		opts.serveGetWithGUIDOnly = serveGetIncludeGlobalDocs
	})
	return b
}

func (b *RouterOptionsBuilder[T]) WithServePost(servePost bool) *RouterOptionsBuilder[T] {
	b.options = append(b.options, func(opts *routerOptions[T]) {
		opts.servePost = servePost
	})
	return b
}

func (b *RouterOptionsBuilder[T]) WithServePut(servePut bool) *RouterOptionsBuilder[T] {
	b.options = append(b.options, func(opts *routerOptions[T]) {
		opts.servePut = servePut
	})
	return b
}

func (b *RouterOptionsBuilder[T]) WithServeDelete(serveDelete bool) *RouterOptionsBuilder[T] {
	b.options = append(b.options, func(opts *routerOptions[T]) {
		opts.serveDelete = serveDelete
	})
	return b
}

func (b *RouterOptionsBuilder[T]) WithIncludeGlobalDocs(serveGetIncludeGlobalDocs bool) *RouterOptionsBuilder[T] {
	b.options = append(b.options, func(opts *routerOptions[T]) {
		opts.serveGetIncludeGlobalDocs = serveGetIncludeGlobalDocs
	})
	return b
}

func (b *RouterOptionsBuilder[T]) WithDeleteByName(serveDeleteByName bool) *RouterOptionsBuilder[T] {
	b.options = append(b.options, func(opts *routerOptions[T]) {
		opts.serveDeleteByName = serveDeleteByName
	})
	return b
}

func (b *RouterOptionsBuilder[T]) WithNameQuery(nameQueryParam string) *RouterOptionsBuilder[T] {
	b.options = append(b.options, func(opts *routerOptions[T]) {
		opts.nameQueryParam = nameQueryParam
	})
	return b
}

func (b *RouterOptionsBuilder[T]) WithQueryConfig(QueryConfig *QueryParamsConfig) *RouterOptionsBuilder[T] {
	b.options = append(b.options, func(opts *routerOptions[T]) {
		opts.QueryConfig = QueryConfig
	})
	return b
}

func (b *RouterOptionsBuilder[T]) WithPutValidators(validators ...MutatorValidator[T]) *RouterOptionsBuilder[T] {
	b.options = append(b.options, func(opts *routerOptions[T]) {
		opts.putValidators = validators
	})
	return b
}

func (b *RouterOptionsBuilder[T]) WithPostValidators(validators ...MutatorValidator[T]) *RouterOptionsBuilder[T] {
	b.options = append(b.options, func(opts *routerOptions[T]) {
		opts.postValidators = validators
	})
	return b
}

func (b *RouterOptionsBuilder[T]) WithValidatePostUniqueName(validatePostUniqueName bool) *RouterOptionsBuilder[T] {
	b.options = append(b.options, func(opts *routerOptions[T]) {
		opts.validatePostUniqueName = validatePostUniqueName
	})
	return b
}

func (b *RouterOptionsBuilder[T]) WithValidatePutGUID(validatePutGUID bool) *RouterOptionsBuilder[T] {
	b.options = append(b.options, func(opts *routerOptions[T]) {
		opts.validatePutGUID = validatePutGUID
	})
	return b
}

func (b *RouterOptionsBuilder[T]) WithUniqueShortName(baseShortNameValue func(T) string) *RouterOptionsBuilder[T] {
	b.options = append(b.options, func(opts *routerOptions[T]) {
		opts.uniqueShortName = baseShortNameValue
	})
	return b
}

func (b *RouterOptionsBuilder[T]) WithGetNamesList(serveNameList bool) *RouterOptionsBuilder[T] {
	b.options = append(b.options, func(opts *routerOptions[T]) {
		opts.serveGetNamesList = serveNameList
	})
	return b
}
