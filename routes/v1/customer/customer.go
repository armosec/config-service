package customer

import (
	"config-service/db"
	"config-service/handlers"
	"config-service/types"
	"config-service/utils/consts"
	"config-service/utils/log"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
)

func AddPublicRoutes(g *gin.Engine) {
	tenant := g.Group(consts.TenantPath)
	tenant.Use(handlers.DBContextMiddleware(consts.CustomersCollection))
	tenant.POST("", postCustomerTenant)
}

func AddRoutes(g *gin.Engine) {
	customer := g.Group(consts.CustomerPath)
	customer.Use(handlers.DBContextMiddleware(consts.CustomersCollection))
	customer.GET("", getCustomer)
	customer.DELETE("", deleteCustomer)
	customer.PUT("", handlers.HandlePutDocWithGUIDValidation[*types.Customer]()...)

	//add customer's inner files routes
	addInnerFieldsRoutes(g)
}

func addInnerFieldsRoutes(g *gin.Engine) {
	//add customer embedded objects routes
	addNotificationConfigRoutes(g)
	addCustomerStateRoutes(g)
}

func getCustomer(c *gin.Context) {
	defer log.LogNTraceEnterExit("getCustomer", c)()
	_, customerGUID, err := db.ReadContext(c)
	if err != nil {
		handlers.ResponseInternalServerError(c, "failed to read customer guid from context", err)
		return
	}
	//do not filter per customer since old data does not have customer field
	filter := db.NewFilterBuilder().WithGUID(customerGUID)
	if doc, err := db.GetDoc[*types.Customer](c, filter); err != nil {
		handlers.ResponseInternalServerError(c, "failed to read document", err)
		return
	} else if doc == nil {
		handlers.ResponseDocumentNotFound(c)
		return
	} else {
		c.JSON(http.StatusOK, doc)
	}
}

func deleteCustomer(c *gin.Context) {
	defer log.LogNTraceEnterExit("deleteCustomer", c)()
	deletedCount, err := db.DeleteCustomerDocs(c)
	if err != nil {
		handlers.ResponseInternalServerError(c, fmt.Sprintf("failed to delete customer docs. %d docs deleted", deletedCount), err)
		return
	} else {
		c.JSON(http.StatusOK, gin.H{"deleted": deletedCount})
	}
}

func postCustomerTenant(c *gin.Context) {
	defer log.LogNTraceEnterExit("postCustomerTenant", c)()
	var customer *types.Customer
	if err := c.ShouldBindBodyWith(&customer, binding.JSON); err != nil || customer == nil {
		handlers.ResponseFailedToBindJson(c, err)
		return
	}
	if customer.GUID == "" {
		handlers.ResponseMissingGUID(c)
		return
	}
	customer.InitNew()
	dbDoc := types.Document[*types.Customer]{
		ID:        customer.GUID,
		Content:   customer,
		Customers: []string{customer.GUID},
	}
	handlers.PostDBDocumentHandler(c, dbDoc)
}
