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
	customerStateField = "state"
)

func addCustomerStateRoutes(g *gin.Engine) {
	handlers.AddRoutes(g, handlers.NewRouterOptionsBuilder[*types.Customer]().
		WithDBCollection(consts.CustomersCollection). //same db as customers
		WithPath(consts.CustomerStatePath).
		WithServeGetWithGUIDOnly(true).                                       //only get single doc by GUID
		WithPutFields([]string{customerStateField, consts.UpdatedTimeField}). //only update customer state field and UpdatedTime fields in customer document
		WithServePost(false).                                                 //no post
		WithServeDelete(false).                                               //no delete
		WithBodyDecoder(decodeCustomerState).                                 //custom decoder
		WithResponseSender(customerStateResponseSender).                      //custom response sender
		Get()...)
}

func customerStateResponseSender(c *gin.Context, customer *types.Customer, customers []*types.Customer) {
	//in Put we expect array of customers the old one and the updated one
	if c.Request.Method == http.MethodPut {
		if len(customers) != 2 {
			handlers.ResponseInternalServerError(c, "unexpected nill doc array response in PUT", nil)
			return
		}
		states := []*armotypes.CustomerState{}
		for _, customer := range customers {
			states = append(states, customer2State(customer))
		}
		c.JSON(http.StatusOK, states)
		return
	}
	if customer == nil {
		handlers.ResponseInternalServerError(c, "unexpected nil doc response", nil)
		return
	}
	c.JSON(http.StatusOK, customer2State(customer))
}

func customer2State(customer *types.Customer) *armotypes.CustomerState {
	if customer == nil {
		return nil
	}
	if customer.State == nil {
		return defaultCustomerState()
	}
	return customer.State
}

func decodeCustomerState(c *gin.Context) ([]*types.Customer, error) {
	var state *armotypes.CustomerState
	// do not support bulk update - so we do not expect array
	if err := c.ShouldBindJSON(&state); err != nil {
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
	customer.State = state
	return []*types.Customer{customer}, nil
}

func defaultCustomerState() *armotypes.CustomerState {
	return &armotypes.CustomerState{
		Onboarding: &armotypes.CustomerOnboarding{
			Completed: true,
		},
	}
}
