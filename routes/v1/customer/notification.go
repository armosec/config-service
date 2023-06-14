package customer

import (
	"config-service/handlers"
	"config-service/types"
	"config-service/utils/consts"
	"config-service/utils/log"
	"fmt"
	"net/http"

	"github.com/armosec/armoapi-go/armotypes"
	"github.com/gin-gonic/gin"
)

const (
	notificationConfigField = "notifications_config"
)

func addNotificationConfigRoutes(g *gin.Engine) {
	handlers.AddRoutes(g, handlers.NewRouterOptionsBuilder[*types.Customer]().
		WithDBCollection(consts.CustomersCollection). //same db as customers
		WithPath(consts.NotificationConfigPath).
		WithServeGetWithGUIDOnly(true).                                                                                            //only get single doc by GUID
		WithPutFields([]string{notificationConfigField, consts.UpdatedTimeField}).                                                 //only update notification-config and UpdatedTime fields in customer document
		WithServePost(false).                                                                                                      //no post
		WithServeDelete(false).                                                                                                    //no delete
		WithBodyDecoder(decodeNotificationConfig).                                                                                 //custom decoder
		WithResponseSender(notificationConfigResponseSender).                                                                      //custom response sender
		WithContainerHandler("/unsubscribe/:userId", unsubscribeMiddleware, handlers.ContainerTypeArray, true, true).              //Add put and delete form unsubscribe array
		WithContainerHandler("/latestPushReport/:clusterName", latestPushReportMiddleware, handlers.ContainerTypeMap, true, true). //Add put and delete in latest report maps
		Get()...)
}

func latestPushReportMiddleware(c *gin.Context) (latestPushPath string, valuesToAdd []interface{}, valid bool) {
	clusterName := c.Param("clusterName")
	if clusterName == "" {
		handlers.ResponseMissingKey(c, "clusterName")
		return "", nil, false
	}
	report := &armotypes.PushReport{}
	if c.Request.Method == http.MethodPut {
		if err := c.ShouldBindJSON(&report); err != nil {
			handlers.ResponseFailedToBindJson(c, err)
			return "", nil, false
		}
	}
	customerGuid := c.GetString(consts.CustomerGUID)
	if customerGuid == "" {
		panic("customerGuid is empty")
	}
	c.Params = append(c.Params, gin.Param{Key: consts.GUIDField, Value: customerGuid})

	latestPushPath = "notifications_config.latestPushReports." + clusterName
	return latestPushPath, []interface{}{report}, true
}

const unsubscribePathPrefix = "notifications_config.unsubscribedUsers."

func unsubscribeMiddleware(c *gin.Context) (unsubscribePath string, valuesToAdd []interface{}, valid bool) {
	userId := c.Param("userId")
	if userId == "" {
		handlers.ResponseMissingKey(c, "userId")
		return "", nil, false
	}
	notificationsIds, err := handlers.GetBulkOrSingleBody[*armotypes.NotificationConfigIdentifier](c)
	if err != nil {
		log.LogNTraceError("failed to get notificationsIds from body", err, c)
		return "", nil, false
	}
	if len(notificationsIds) == 0 {
		return "", nil, false
	}
	for _, notificationId := range notificationsIds {
		if notificationId == nil || notificationId.NotificationType == "" {
			handlers.ResponseMissingKey(c, "notificationId")
			return "", nil, false
		}
	}
	customerGuid := c.GetString(consts.CustomerGUID)
	if customerGuid == "" {
		panic("customerGuid is empty")
	}
	c.Params = append(c.Params, gin.Param{Key: consts.GUIDField, Value: customerGuid})

	unsubscribePath = unsubscribePathPrefix + userId
	resp := make([]interface{}, 0, len(notificationsIds))
	for _, notificationId := range notificationsIds {
		resp = append(resp, notificationId)
	}
	return unsubscribePath, resp, true
}

func notificationConfigResponseSender(c *gin.Context, customer *types.Customer, customers []*types.Customer) {
	//in Put we expect array of customers the old one and the updated one
	if c.Request.Method == http.MethodPut {
		if len(customers) != 2 {
			handlers.ResponseInternalServerError(c, "unexpected nill doc array response in PUT", nil)
			return
		}
		notifications := []*armotypes.NotificationsConfig{}
		for _, customer := range customers {
			notifications = append(notifications, customer2NotificationConfig(customer))
		}
		c.JSON(http.StatusOK, notifications)
		return
	}
	if customer == nil {
		handlers.ResponseInternalServerError(c, "unexpected nil doc response", nil)
		return
	}
	c.JSON(http.StatusOK, customer2NotificationConfig(customer))
}

func customer2NotificationConfig(customer *types.Customer) *armotypes.NotificationsConfig {
	if customer == nil {
		return nil
	}
	if customer.NotificationsConfig == nil {
		return &armotypes.NotificationsConfig{}
	}
	return customer.NotificationsConfig
}

func decodeNotificationConfig(c *gin.Context) ([]*types.Customer, error) {
	var notificationConfig *armotypes.NotificationsConfig
	//notificationConfig do not support bulk update - so we do not expect array
	if err := c.ShouldBindJSON(&notificationConfig); err != nil {
		handlers.ResponseFailedToBindJson(c, err)
		return nil, err
	}
	customerGuid := c.GetString(consts.CustomerGUID)
	if customerGuid == "" {
		handlers.ResponseInternalServerError(c, "failed to read customer guid from context", nil)
		return nil, fmt.Errorf("failed to read customer guid from context")
	}
	customer := &types.Customer{}
	customer.GUID = customerGuid
	customer.NotificationsConfig = notificationConfig
	return []*types.Customer{customer}, nil
}
