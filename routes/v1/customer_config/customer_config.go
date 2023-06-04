package customer_config

import (
	"config-service/db"
	"config-service/handlers"
	"config-service/types"
	"config-service/utils/consts"
	"config-service/utils/log"
	"net/http"

	"github.com/imdario/mergo"

	"github.com/gin-gonic/gin"
)

func getCustomerConfigHandler(c *gin.Context) {
	if handlers.GetNamesListHandler[*types.CustomerConfig](c, true) {
		return
	}
	if getCustomerConfigByNameHandler(c) {
		return
	}
	handlers.HandleGetAllWithGlobals[*types.CustomerConfig](c)
}

func getCustomerConfigByNameHandler(c *gin.Context) bool {
	defer log.LogNTraceEnterExit("getCustomerConfigByNameHandler", c)()
	configName := getConfigName(c)
	if configName == "" {
		return false
	}

	//get the default config
	var defaultConfig *types.CustomerConfig
	var err error
	if defaultCustomerConfig != nil {
		// get the default config from config file if provided
		defaultConfig = defaultCustomerConfig
	} else {
		// get the default config from db if not provided by config file
		defaultConfig, err = db.GetCachedDocument[*types.CustomerConfig](consts.DefaultCustomerConfigKey)
		if err != nil {
			handlers.ResponseInternalServerError(c, "failed to get default config", err)
			return true
		}
	}

	//case default config is requested - return it
	if configName == consts.GlobalConfigName {
		c.JSON(http.StatusOK, defaultConfig)
		return true
	}
	//try and get config by name from db
	doc, err := db.GetDocByName[types.CustomerConfig](c, configName)
	if err != nil {
		handlers.ResponseInternalServerError(c, "failed to get document by name", err)
		return true
	} else if unmerged, _ := c.GetQuery("unmerged"); unmerged != "" {
		//case unmerged is requested - return the unmerged config if exists
		if doc == nil {
			handlers.ResponseDocumentNotFound(c)
			return true
		} else {
			c.JSON(http.StatusOK, doc)
			return true
		}
	}
	//case customer config is requested - return it merged with default config
	if configName == consts.CustomerConfigName {
		if doc != nil {
			if err := mergo.Merge(doc, *defaultConfig); err != nil {
				handlers.ResponseInternalServerError(c, "failed to merge configuration", err)
				return true
			} else {
				c.JSON(http.StatusOK, doc)
				return true
			}
		} else {
			//case customer config is requested but not exists - return default config
			c.JSON(http.StatusOK, defaultConfig)
			return true
		}
	}
	//case cluster config is requested - return it merged with customer and default config
	customerConfig, err := db.GetDocByName[types.CustomerConfig](c, consts.CustomerConfigName)
	if err != nil {
		handlers.ResponseInternalServerError(c, "failed to get document by name", err)
		return true
	}
	customerConfig, err = mergeConfigurations(customerConfig, defaultConfig)
	if err != nil {
		handlers.ResponseInternalServerError(c, "failed to merge configuration", err)
		return true
	}

	doc, err = mergeConfigurations(doc, customerConfig)
	if err != nil {
		handlers.ResponseInternalServerError(c, "failed to merge configuration", err)
		return true
	}
	c.JSON(http.StatusOK, doc)
	return true
}

func mergeConfigurations(dest, src *types.CustomerConfig) (*types.CustomerConfig, error) {
	if dest == nil {
		return src, nil
	}
	if src == nil {
		return dest, nil
	}
	destCopy := *dest
	if err := mergo.Merge(&destCopy, *src); err != nil {
		return nil, err
	}
	return &destCopy, nil
}

func validatePutCustomerConfig(c *gin.Context, docs []*types.CustomerConfig) ([]*types.CustomerConfig, bool) {
	defer log.LogNTraceEnterExit("validatePutCustomerConfig", c)()
	if len(docs) > 1 {
		handlers.ResponseBulkNotSupported(c)
		return nil, false
	}
	configName := getConfigName(c)
	if configName == "" {
		if docs[0].Name != "" {
			configName = docs[0].Name
		} else {
			handlers.ResponseMissingName(c)
			return nil, false
		}
	}
	if existingDoc, err := db.GetDocByName[types.CustomerConfig](c, configName); err != nil {
		handlers.ResponseInternalServerError(c, "failed to get doc by name", err)
		return nil, false
	} else if existingDoc == nil {
		handlers.ResponseDocumentNotFound(c)
		return nil, false
	} else {
		docs[0].SetGUID(existingDoc.GetGUID())
	}
	return docs, true
}

func deleteCustomerConfig(c *gin.Context) {
	defer log.LogNTraceEnterExit("deleteCustomerConfig", c)()
	if configName := getConfigName(c); configName != "" {
		if configName == consts.GlobalConfigName {
			handlers.ResponseBadRequest(c, "default config cannot be deleted")
			return
		}
		handlers.DeleteDocByNameHandler[*types.CustomerConfig](c, configName)
	} else {
		handlers.ResponseMissingName(c)
		return
	}
}

func getConfigName(c *gin.Context) string {
	if clusterName, ok := c.GetQuery(consts.ClusterNameParam); ok && clusterName != "" {
		return clusterName
	}
	if configName, ok := c.GetQuery(consts.ConfigNameParam); ok && configName != "" {
		return configName
	}
	if scope, ok := c.GetQuery(consts.ScopeParam); ok && scope != "" {
		switch scope {
		case consts.CustomerScope:
			return consts.CustomerConfigName
		case consts.DefaultScope:
			return consts.GlobalConfigName
		}
	}
	return ""
}
