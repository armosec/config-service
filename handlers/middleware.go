package handlers

import (
	"config-service/types"
	"config-service/utils/consts"
	"config-service/utils/log"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
)

// ////////////////////////////////db handler middleware//////////////////////////////////
// DBContextMiddleware is a middleware that adds db parameters to the context
func DBContextMiddleware(collectionName string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set(consts.Collection, collectionName)
		c.Next()
	}
}

func SchemaContextMiddleware(schema types.SchemaInfo) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set(consts.SchemaInfo, schema)
		c.Next()
	}
}

func NestedDocContextMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if baseDocID := c.Param(consts.GUIDField); baseDocID != "" {
			c.Set(consts.BaseDocID, baseDocID)
		} else {
			ResponseBadRequest(c, "missing base document id")
			return
		}
		c.Next()
	}
}

func PutFieldsContextMiddleware(fields []string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set(consts.PutDocFields, fields)
		c.Next()
	}
}

func BodyDecoderContextMiddleware[T types.DocContent](decoder *BodyDecoder[T]) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set(consts.BodyDecoder, decoder)
		c.Next()
	}
}

func ResponseSenderContextMiddleware[T types.DocContent](sender *ResponseSender[T]) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set(consts.ResponseSender, sender)
		c.Next()
	}
}

// PostValidationMiddleware validate post request and if valid sets one or many DocContents in context for next handler, otherwise abort request
func PostValidationMiddleware[T types.DocContent](validators ...MutatorValidator[T]) func(c *gin.Context) {
	return func(c *gin.Context) {
		defer log.LogNTraceEnterExit("PostValidationMiddleware", c)()
		var doc T
		var docs []T
		if customDecoder, _ := GetCustomBodyDecoder[T](c); customDecoder != nil {
			var err error
			if docs, err = customDecoder(c); err != nil {
				ResponseFailedToBindJson(c, err)
				return
			}
		} else if err := c.ShouldBindBodyWith(&doc, binding.JSON); err != nil || doc == nil {
			//check if bulk request
			if err := c.ShouldBindBodyWith(&docs, binding.JSON); err != nil || docs == nil {
				ResponseFailedToBindJson(c, err)
				return
			}
		} else {
			//single request, append to slice
			docs = append(docs, doc)
		}

		//validate
		if len(docs) == 0 {
			ResponseBadRequest(c, "no documents in request")
			return
		}

		for _, validator := range validators {
			var ok bool
			if docs, ok = validator(c, docs); !ok {
				return
			}
		}
		c.Set(consts.DocContentKey, docs)
		c.Next()
	}
}

// PutValidationMiddleware validate put request and if valid set DocContent in context for next handler, otherwise abort request
func PutValidationMiddleware[T types.DocContent](validators ...MutatorValidator[T]) func(c *gin.Context) {
	return func(c *gin.Context) {
		defer log.LogNTraceEnterExit("PutValidationMiddleware", c)()
		var doc T
		if customDecoder, _ := GetCustomBodyDecoder[T](c); customDecoder != nil {
			if docs, err := customDecoder(c); err != nil {
				ResponseFailedToBindJson(c, err)
				return
			} else {
				doc = docs[0]
			}
		} else if err := c.ShouldBindJSON(&doc); err != nil {
			ResponseFailedToBindJson(c, err)
			return
		}
		//validate
		for _, validator := range validators {
			if docs, ok := validator(c, []T{doc}); !ok {
				return
			} else {
				doc = docs[0]
			}
		}
		c.Set(consts.DocContentKey, doc)
		c.Next()
	}
}
