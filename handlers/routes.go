package handlers

import (
	"config-service/db"
	"config-service/types"
	"config-service/utils/consts"
	"fmt"

	"github.com/gin-gonic/gin"
)

// router options
type routerOptions[T types.DocContent] struct {
	dbCollection              string                    //mandatory db collection name
	path                      string                    //mandatory uri path
	serveGet                  bool                      //default true, serve GET /<path> to get all documents and GET /<path>/<GUID> to get document by GUID
	serveGetNamesList         bool                      //default true, GET will return all documents names if "list" query param exist
	serveGetWithGUIDOnly      bool                      //default false, GET will return the document by GUID only
	serveGetIncludeGlobalDocs bool                      //default false, when true, in GET all the response will include global documents (with customers[""])
	servePost                 bool                      //default true, serve POST
	servePostV2ListRequests   bool                      //default false, when true  POST /<path>/query with V2ListRequest is served
	servePut                  bool                      //default true, serve PUT /<path> to update document by GUID in body and PUT /<path>/<GUID> to update document by GUID in path
	serveDelete               bool                      //default true, serve DELETE  /<path>/<GUID> to delete document by GUID in path
	serveBulkDelete           bool                      //default true, serve DELETE /<path>/bulk with list of GUIDs in body or query to delete documents by GUIDs
	serveDeleteByQuery        bool                      //default true, serve DELETE /<path>/query with V2ListRequest in body - all documents matching the query will be deleted
	serveDeleteByName         bool                      //default false, when true, DELETE will check for name param and will delete the document by name
	validatePostUniqueName    bool                      //default true, POST will validate that the name is unique
	validatePostMandatoryName bool                      //default false, POST will validate that the name exists
	validatePutUniqueName     bool                      //default false, if element allows rename PUT will validate that the name is not empty and unique
	validatePutGUID           bool                      //default true, PUT will validate GUID existence in body or path
	nameQueryParam            string                    //default empty, the param name that indicates query by name (e.g. clusterName) when set GET will check for this param and will return the document by name
	QueryConfig               *QueryParamsConfig        //default nil, when set, GET will check for the specified query params and will return the documents by the query params
	uniqueShortName           func(T) string            //default nil, when set, POST will create a unique short name (aka "alias") attribute from the value returned from the function & Put will validate that the short name is not deleted
	putValidators             []MutatorValidator[T]     //default nil, when set, PUT will call the mutators/validators before updating the document
	postValidators            []MutatorValidator[T]     //default nil, when set, POST will call the mutators/validators before creating the document
	bodyDecoder               BodyDecoder[T]            //default nil, when set, replace the default body decoder
	responseSender            ResponseSender[T]         //default nil, when set, replace the default response sender
	putFields                 []string                  //default nil, when set, PUT will update only the specified fields
	containersHandlers        []containerHandlerOptions //default nil, list of container handlers to put and remove items from document's containers
	schemaInfo                types.SchemaInfo          //default nil, when set, the schema info will be used for queries (e.g. identify arrays)
}

type ContainerType string

const (
	ContainerTypeArray ContainerType = "array"
	ContainerTypeMap   ContainerType = "map"

	bulkSuffix         = "/bulk"
	querySuffix        = "/query"
	uniqueValuesSuffix = "/uniqueValues"
)

type containerHandlerOptions struct {
	path             string           //mandatory, the api path to handle the internal field (map or array)
	ContainerHandler ContainerHandler //mandatory, middleware function to validate the request and return the internal field and value
	servePut         bool             //Serve PUT <path> to add items
	serveDelete      bool             //Serve DELETE <path> to delete items
	containerType    ContainerType    //Type of container = currently map or array are supported
}

func newRouterOptions[T types.DocContent]() *routerOptions[T] {
	return &routerOptions[T]{
		serveGet:                  true,
		servePost:                 true,
		servePut:                  true,
		serveDelete:               true,
		serveBulkDelete:           true,
		serveDeleteByQuery:        true,
		validatePostUniqueName:    true,
		validatePutGUID:           true,
		serveGetNamesList:         true,
		serveGetIncludeGlobalDocs: false,
		serveDeleteByName:         false,
		validatePostMandatoryName: false,
	}
}

func AddRoutes[T types.DocContent](g *gin.Engine, options ...RouterOption[T]) *gin.RouterGroup {
	opts := newRouterOptions[T]()
	opts.apply(options)
	if err := opts.validate(); err != nil {
		panic(err)
	}
	//validate and initialize collection
	if err := db.ValidateCollection(opts.dbCollection); err != nil {
		panic(err)
	}
	addRouteInfo(opts)
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
		if opts.validatePostMandatoryName {
			postValidators = append(postValidators, ValidateNameExistence[T])
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
		if opts.validatePutUniqueName {
			putValidators = append(putValidators, ValidateUniqueValues(NameKeyGetter[T]))
		}
		putValidators = append(putValidators, opts.putValidators...)
		routerGroup.PUT("", HandlePutDocWithValidation(putValidators...)...)
		routerGroup.PUT("/:"+consts.GUIDField, HandlePutDocWithValidation(putValidators...)...)
	}
	if opts.serveDelete {
		if opts.serveDeleteByName {
			routerGroup.DELETE("", HandleDeleteDocByName[T](opts.nameQueryParam))
		}
		if opts.serveBulkDelete {
			routerGroup.DELETE(bulkSuffix, HandleBulkDeleteWithGUIDs[T])
		}
		if opts.serveDeleteByQuery {
			routerGroup.DELETE(querySuffix, HandleDeleteByQuery[T])
		}
		routerGroup.DELETE("/:"+consts.GUIDField, HandleDeleteDoc[T])
	}
	if opts.servePostV2ListRequests {
		putSchemaInContext := SchemaContextMiddleware(opts.schemaInfo)
		routerGroup.POST(querySuffix, putSchemaInContext, HandlePostV2ListRequest[T])
		routerGroup.POST(uniqueValuesSuffix, putSchemaInContext, HandlePostUniqueValuesRequestV2[T])

	}
	//add array handlers
	for _, containerHandler := range opts.containersHandlers {
		switch containerHandler.containerType {
		case ContainerTypeArray:
			if containerHandler.servePut {
				routerGroup.PUT(containerHandler.path, HandlerAddToArray(containerHandler.ContainerHandler))
			}
			if containerHandler.serveDelete {
				routerGroup.DELETE(containerHandler.path, HandlerRemoveFromArray(containerHandler.ContainerHandler))
			}
		case ContainerTypeMap:
			if containerHandler.servePut {
				routerGroup.PUT(containerHandler.path, HandlerSetField(containerHandler.ContainerHandler, true))
			}
			if containerHandler.serveDelete {
				routerGroup.DELETE(containerHandler.path, HandlerSetField(containerHandler.ContainerHandler, false))
			}
		}
	}
	return routerGroup
}

// Common router config for policies
func AddPolicyRoutes[T types.DocContent](g *gin.Engine, path, dbCollection string, paramConf *QueryParamsConfig, allowRename bool, schema *types.SchemaInfo) *gin.RouterGroup {
	routerOptionsBuilder := NewRouterOptionsBuilder[T]().
		WithPath(path).
		WithDBCollection(dbCollection).
		WithNameQuery(consts.PolicyNameParam).
		WithQueryConfig(paramConf).
		WithIncludeGlobalDocs(true).
		WithDeleteByName(true).
		WithValidatePostUniqueName(true).
		WithValidatePutGUID(true).
		WithV2ListSearch(true).
		WithValidatePutUniqueName(allowRename)

	if schema != nil {
		routerOptionsBuilder.
			WithSchemaInfo(*schema)
	}

	return AddRoutes(g, routerOptionsBuilder.Get()...)
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
	if opts.serveGetWithGUIDOnly && !opts.serveGet {
		return fmt.Errorf("serveGetWithGUIDOnly can only be true when serveGet is true")
	}
	return nil
}

// map of collection name to admin query handler
var coll2AdminQueryHandler = map[string]gin.HandlerFunc{}

func GetAdminQueryHandler(collection string) gin.HandlerFunc {
	return coll2AdminQueryHandler[collection]
}

// keep the api info for each route
func addRouteInfo[T types.DocContent](options *routerOptions[T]) {
	apiInfo := types.APIInfo{
		BasePath:     options.path,
		DBCollection: options.dbCollection,
		Schema:       options.schemaInfo,
	}
	types.SetAPIInfo(options.path, apiInfo)
	//keep the admin query handler for this route
	coll2AdminQueryHandler[options.dbCollection] = HandleAdminPostV2ListRequest[T]
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

func (b *RouterOptionsBuilder[T]) WithSchemaInfo(schemaInfo types.SchemaInfo) *RouterOptionsBuilder[T] {
	b.options = append(b.options, func(opts *routerOptions[T]) {
		opts.schemaInfo = schemaInfo
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

func (b *RouterOptionsBuilder[T]) WithV2ListSearch(servePostV2ListRequests bool) *RouterOptionsBuilder[T] {
	b.options = append(b.options, func(opts *routerOptions[T]) {
		opts.servePostV2ListRequests = servePostV2ListRequests
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

func (b *RouterOptionsBuilder[T]) WithServeBulkDelete(serveBulkDelete bool) *RouterOptionsBuilder[T] {
	b.options = append(b.options, func(opts *routerOptions[T]) {
		opts.serveBulkDelete = serveBulkDelete
	})
	return b
}

func (b *RouterOptionsBuilder[T]) WithServeDeleteByQuery(serveDeleteByQuery bool) *RouterOptionsBuilder[T] {
	b.options = append(b.options, func(opts *routerOptions[T]) {
		opts.serveDeleteByQuery = serveDeleteByQuery
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

func (b *RouterOptionsBuilder[T]) WithValidatePostMandatoryName(validatePostMandatoryName bool) *RouterOptionsBuilder[T] {
	b.options = append(b.options, func(opts *routerOptions[T]) {
		opts.validatePostMandatoryName = validatePostMandatoryName
	})
	return b
}

func (b *RouterOptionsBuilder[T]) WithValidatePutUniqueName(validatePutUniqueName bool) *RouterOptionsBuilder[T] {
	b.options = append(b.options, func(opts *routerOptions[T]) {
		opts.validatePutUniqueName = validatePutUniqueName
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

func (b *RouterOptionsBuilder[T]) WithContainerHandler(path string, containerHandler ContainerHandler, containerType ContainerType, servePut, serveDelete bool) *RouterOptionsBuilder[T] {
	if path == "" || containerHandler == nil {
		panic("path and ContainerHandler are mandatory")
	}
	if !servePut && !serveDelete {
		panic("at least one of servePut and serveDelete must be true")
	}
	b.options = append(b.options, func(opts *routerOptions[T]) {
		opts.containersHandlers = append(opts.containersHandlers, containerHandlerOptions{
			path:             path,
			ContainerHandler: containerHandler,
			servePut:         servePut,
			serveDelete:      serveDelete,
			containerType:    containerType,
		})

	})
	return b
}

func AddRouteInfo[T types.DocContent](apiInfo types.APIInfo) {
	types.SetAPIInfo(apiInfo.BasePath, apiInfo)
	//keep the admin query handler for this route
	coll2AdminQueryHandler[apiInfo.DBCollection] = HandleAdminPostV2ListRequest[T]
}
