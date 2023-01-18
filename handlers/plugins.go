package handlers

import (
	"config-service/db"
	"config-service/types"
	"config-service/utils/consts"
	"config-service/utils/log"
	"fmt"

	"github.com/gin-gonic/gin"
)

// MutatorValidator is a function that validates and/or modifies the request body and returns true if the request is valid
// a MutatorValidator may initialize the doc with required values therefor it returns the docs as well
type MutatorValidator[T types.DocContent] func(c *gin.Context, docs []T) (verifiedDocs []T, valid bool)

// BodyDecoder is used for custom decoding of the request body it returns the decoded docs or an error
type BodyDecoder[T types.DocContent] func(c *gin.Context) ([]T, error)

// ResponseSender is used for custom response sending it is called after the request has been processed
type ResponseSender[T types.DocContent] func(c *gin.Context, doc T, docs []T)

// type ArrayRequestHandler is used to handle request for updating internal arrays in a document
// it returns the path to the array (e.g. "internalFields.tags"), the value to add or pull and a query filter to find the document
type ArrayRequestHandler func(c *gin.Context) (pathToArray string, valueToAdd interface{}, queryFilter *db.FilterBuilder, valid bool)

func GetCustomBodyDecoder[T types.DocContent](c *gin.Context) (BodyDecoder[T], error) {
	if iDecoder, ok := c.Get(consts.BodyDecoder); ok {
		if decoder, ok := iDecoder.(*BodyDecoder[T]); ok && decoder != nil {
			return *decoder, nil
		}
		err := fmt.Errorf("invalid body decoder type")
		log.LogNTraceError("invalid body decoder type", err, c)
		return nil, err
	}
	return nil, nil
}

func GetCustomResponseSender[T types.DocContent](c *gin.Context) (ResponseSender[T], error) {
	if iSender, ok := c.Get(consts.ResponseSender); ok {
		if sender, ok := iSender.(*ResponseSender[T]); ok && sender != nil {
			return *sender, nil
		}
		err := fmt.Errorf("invalid response sender type")
		log.LogNTraceError("invalid response sender type", err, c)
		return nil, err
	}
	return nil, nil
}

func GetCustomPutFields(c *gin.Context) []string {
	if iFields, ok := c.Get(consts.PutDocFields); ok {
		if fieldsNames, ok := iFields.([]string); ok {
			return fieldsNames
		}
		err := fmt.Errorf("invalid fieldsNames type")
		log.LogNTraceError("invalid fieldsNames type", err, c)
		return nil
	}
	return nil
}
