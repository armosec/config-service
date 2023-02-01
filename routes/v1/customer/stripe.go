package customer

import (
	"config-service/handlers"
	"config-service/types"
	"config-service/utils/consts"
	"fmt"
	"net/http"

	"github.com/armosec/armoapi-go/armotypes"
	"github.com/gin-gonic/gin"
)

const (
	stripeCustomerField = "stripe_customer"
)

func addStripeRoutes(g *gin.Engine) {
	handlers.AddRoutes(g, handlers.NewRouterOptionsBuilder[*types.Customer]().
		WithDBCollection(consts.CustomersCollection). //same db as customers
		WithPath(consts.StripeCustomerPath).
		WithServeGetWithGUIDOnly(true).                                        //only get single doc by GUID
		WithPutFields([]string{stripeCustomerField, consts.UpdatedTimeField}). //only update stripeCustomer and UpdatedTime fields in customer document
		WithServePost(false).                                                  //no post
		WithServeDelete(false).                                                //no delete
		WithBodyDecoder(decodeStripeCustomer).                                 //custom decoder
		WithResponseSender(stripeCustomerResponseSender).                      //custom response sender
		Get()...)
}

func stripeCustomerResponseSender(c *gin.Context, customer *types.Customer, customers []*types.Customer) {
	//in Put we expect array of customers the old one and the updated one
	if c.Request.Method == http.MethodPut {
		if len(customers) != 2 {
			handlers.ResponseInternalServerError(c, "unexpected nill doc array response in PUT", nil)
			return
		}
		stripeCustomer := []*armotypes.StripeCustomer{}
		for _, customer := range customers {
			stripeCustomer = append(stripeCustomer, customer2SripeCustomer(customer))
		}
		c.JSON(http.StatusOK, stripeCustomer)
		return
	}
	if customer == nil {
		handlers.ResponseInternalServerError(c, "unexpected nil doc response", nil)
		return
	}
	c.JSON(http.StatusOK, customer2SripeCustomer(customer))
}

func customer2SripeCustomer(customer *types.Customer) *armotypes.StripeCustomer {
	if customer == nil {
		return nil
	}

	return customer.StripeCustomer
}

func decodeStripeCustomer(c *gin.Context) ([]*types.Customer, error) {
	var stripeCustomer *armotypes.StripeCustomer
	// do not support bulk update - so we do not expect array
	if err := c.ShouldBindJSON(&stripeCustomer); err != nil {
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
	customer.StripeCustomer = stripeCustomer
	return []*types.Customer{customer}, nil
}
