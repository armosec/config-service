package customer

import (
	"config-service/handlers"
	"config-service/types"
	"config-service/utils"
	"config-service/utils/consts"
	"fmt"
	"net/http"

	"github.com/armosec/armoapi-go/armotypes"
	"github.com/gin-gonic/gin"
)

const (
	activeSubscription = "activeSubscription"
)

func addPaymentRoutes(g *gin.Engine) {
	handlers.AddRoutes(g, handlers.NewRouterOptionsBuilder[*types.Customer]().
		WithDBCollection(consts.CustomersCollection). //same db as customers
		WithPath(consts.ActiveSubscriptionPath).
		WithServeGetWithGUIDOnly(true).                                       //only get single doc by GUID
		WithPutFields([]string{activeSubscription, consts.UpdatedTimeField}). //only update stripeCustomer and UpdatedTime fields in customer document
		WithServePost(false).                                                 //no post
		WithServeDelete(false).                                               //no delete
		WithBodyDecoder(decodePaymentCustomer).                               //custom decoder
		WithResponseSender(subscriptionResponseSender).                       //custom response sender
		Get()...)
}

func subscriptionResponseSender(c *gin.Context, customer *types.Customer, customers []*types.Customer) {
	//in Put we expect array of customers the old one and the updated one
	if c.Request.Method == http.MethodPut {
		if len(customers) != 2 {
			handlers.ResponseInternalServerError(c, "unexpected nill doc array response in PUT", nil)
			return
		}
		subscriptionList := []*armotypes.Subscription{}
		for _, customer := range customers {
			subscriptionList = append(subscriptionList, customer2subscription(customer))

		}
		c.JSON(http.StatusOK, subscriptionList)
		return
	}
	if customer == nil {
		handlers.ResponseInternalServerError(c, "unexpected nil doc response", nil)
		return
	}
	c.JSON(http.StatusOK, customer2subscription(customer))
}

func customer2subscription(customer *types.Customer) *armotypes.Subscription {
	if customer == nil {
		return nil
	}

	if customer.ActiveSubscription == nil {
		return nil
	}

	if customer.ActiveSubscription.LicenseType == "" {
		customer.ActiveSubscription.LicenseType = defaultLicenseTypeActiveSubscription()
	}

	if customer.ActiveSubscription.CancelAtPeriodEnd == nil {
		customer.ActiveSubscription.CancelAtPeriodEnd = utils.BoolPointer(false)
	}

	return customer.ActiveSubscription
}

func decodePaymentCustomer(c *gin.Context) ([]*types.Customer, error) {
	var activeSubscription *armotypes.Subscription
	// do not support bulk update - so we do not expect array
	if err := c.ShouldBindJSON(&activeSubscription); err != nil {
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
	customer.ActiveSubscription = activeSubscription

	return []*types.Customer{customer}, nil
}

func defaultLicenseTypeActiveSubscription() armotypes.LicenseType {
	return armotypes.LicenseTypeFree
}
