package user

import (
	"config-service/handlers"
	"config-service/utils/consts"
	"net/http"

	"github.com/armosec/armoapi-go/armotypes"
	"github.com/gin-gonic/gin"
)

func bannerMiddleware(c *gin.Context) (unsubscribePath string, valuesToAdd []interface{}, valid bool) {
	bannerID := c.Param("bannerID")
	if bannerID == "" {
		handlers.ResponseMissingKey(c, "bannerID")
		return "", nil, false
	}

	banner := &armotypes.Banner{}
	if c.Request.Method == http.MethodPut {
		if err := c.ShouldBindJSON(&banner); err != nil {
			handlers.ResponseFailedToBindJson(c, err)
			return "", nil, false
		}
	}

	customerGuid := c.GetString(consts.CustomerGUID)
	if customerGuid == "" {
		handlers.ResponseMissingKey(c, consts.CustomerGUID)
		return "", nil, false
	}

	c.Params = append(c.Params, gin.Param{Key: consts.GUIDField, Value: customerGuid})

	bannerPath := "dismissedBanners." + bannerID

	return bannerPath, []interface{}{banner}, true
}
