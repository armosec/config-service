package handlers

import (
	"config-service/db"
	"config-service/types"
	"config-service/utils/consts"
	"config-service/utils/log"

	"github.com/gin-gonic/gin"
	"golang.org/x/exp/slices"
)

func ValidateGUIDExistence[T types.DocContent](c *gin.Context, docs []T) ([]T, bool) {
	guid := c.Param(consts.GUIDField)
	if guid != "" && len(docs) != 1 {
		ResponseBadRequest(c, "GUID in path is not allowed in bulk request")
		return nil, false
	}
	for i := range docs {
		if guid != "" {
			docs[i].SetGUID(guid)
		}
		if docs[i].GetGUID() == "" {
			ResponseMissingGUID(c)
			return nil, false
		}
	}
	return docs, true
}

type UniqueKeyValueInfo[T types.DocContent] func() (key string, mandatory bool, valueGetter func(T) string)

func ValidateUniqueValues[T types.DocContent](uniqueKeyValues ...UniqueKeyValueInfo[T]) func(c *gin.Context, docs []T) ([]T, bool) {
	return func(c *gin.Context, docs []T) ([]T, bool) {
		filter := db.NewFilterBuilder()
		projection := db.NewProjectionBuilder()
		keys2Values := map[string][]string{}
		for _, uniqueKeyValue := range uniqueKeyValues {
			key, mandatory, valueGetter := uniqueKeyValue()
			values := []string{}
			for _, doc := range docs {
				value := valueGetter(doc)
				if mandatory && value == "" {
					ResponseMissingKey(c, key)
					return nil, false
				}
				if slices.Contains(values, value) {
					ResponseDuplicateKey(c, key, value)
					return nil, false
				}
				values = append(values, value)
			}
			if len(filter.Get()) > 0 {
				filter.WarpOr()
			}
			if len(docs) > 1 {
				filter.WithIn(key, values)
			} else {
				filter.WithValue(key, values[0])
			}
			projection.Include(key)
			keys2Values[key] = values
		}

		if existingDocs, err := db.FindForCustomer[T](c, filter, projection.Get()); err != nil {
			ResponseInternalServerError(c, "failed to read documents", err)
			return nil, false
		} else if len(existingDocs) > 0 {
			key2ExistingValues := map[string][]string{}
			for _, uniqueKeyValue := range uniqueKeyValues {
				key, _, valueGetter := uniqueKeyValue()
				values := keys2Values[key]
				for _, doc := range existingDocs {
					value := valueGetter(doc)
					if slices.Contains(values, value) && !slices.Contains(key2ExistingValues[key], value) {
						key2ExistingValues[key] = append(key2ExistingValues[key], value)
					}
				}
			}
			ResponseDuplicateKeysNValues(c, key2ExistingValues)
			return nil, false
		}
		return docs, true
	}
}

func NameKeyGetter[T types.DocContent]() (key string, mandatory bool, valueGetter func(T) string) {
	return "name", true, func(doc T) string { return doc.GetName() }
}

func NameValueGetter[T types.DocContent](doc T) string {
	return doc.GetName()
}

func ValidatePostAttributeShortName[T types.DocContent](valueGetter func(T) string) func(c *gin.Context, docs []T) ([]T, bool) {
	return func(c *gin.Context, docs []T) ([]T, bool) {
		defer log.LogNTraceEnterExit("validatePostAttributeShortName", c)()
		for i := range docs {
			attributes := docs[i].GetAttributes()
			if attributes == nil {
				attributes = map[string]interface{}{}
			}
			if shortName, ok := attributes[consts.ShortNameAttribute]; !ok || shortName == "" {
				attributes[consts.ShortNameAttribute] = getUniqueShortName[T](valueGetter(docs[i]), c)
			}
		}
		return docs, true
	}
}

func ValidatePutAttributerShortName[T types.DocContent](c *gin.Context, docs []T) ([]T, bool) {
	defer log.LogNTraceEnterExit("validatePutAttributerShortName", c)()
	for i := range docs {
		attributes := docs[i].GetAttributes()
		if len(attributes) == 0 {
			continue //the doc does not update attributes
		}
		// if request attributes do not include alias add it from the old cluster
		if _, ok := attributes[consts.ShortNameAttribute]; !ok {
			if oldCluster, err := db.GetDocByGUID[types.Cluster](c, docs[i].GetGUID()); err != nil {
				ResponseInternalServerError(c, "failed to read cluster", err)
				return nil, false
			} else if oldCluster == nil {
				ResponseDocumentNotFound(c)
				return nil, false
			} else {
				attributes[consts.ShortNameAttribute] = oldCluster.Attributes[consts.ShortNameAttribute]
				docs[i].SetAttributes(attributes)
			}
		}
	}
	return docs, true
}
