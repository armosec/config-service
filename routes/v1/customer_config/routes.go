package customer_config

import (
	"config-service/db"
	"config-service/handlers"
	"config-service/types"
	"config-service/utils"
	"config-service/utils/consts"
	"time"

	"github.com/gin-gonic/gin"
)

// Holds the default customer config if it was loaded from config file
var defaultCustomerConfig *types.CustomerConfig

func AddRoutes(g *gin.Engine) {
	customerConfigRouter := handlers.AddRoutes(g, handlers.NewRouterOptionsBuilder[*types.CustomerConfig]().
		WithPath(consts.CustomerConfigPath).
		WithDBCollection(consts.CustomerConfigCollection).
		WithServeGet(false).                          // customer config needs custom get handler
		WithServeDelete(false).                       // customer config needs custom delete handler
		WithValidatePutGUID(false).                   // customer config needs custom put validator
		WithPutValidators(validatePutCustomerConfig). //customer config custom put validator
		Get()...)

	customerConfigRouter.GET("", getCustomerConfigHandler)
	customerConfigRouter.DELETE("", deleteCustomerConfig)

	// load default customer config from config file
	if defaultConfigs := utils.GetConfig().DefaultConfigs; defaultConfigs != nil {
		defaultCustomerConfig = defaultConfigs.CustomerConfig
	}

	// add lazy cache to default customer config
	if defaultCustomerConfig == nil {
		db.AddCachedDocument[*types.CustomerConfig](consts.DefaultCustomerConfigKey,
			consts.CustomerConfigCollection,
			db.NewFilterBuilder().WithGlobal().WithName(consts.GlobalConfigName),
			time.Minute*5)
	}
}

func SetDefaultConfigForTest(c *types.CustomerConfig) {
	defaultCustomerConfig = c
}
