package admin

import (
	"config-service/db"
	"config-service/handlers"
	"config-service/types"
	"config-service/utils"
	"config-service/utils/consts"
	"config-service/utils/log"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/exp/slices"
)

func AddRoutes(g *gin.Engine) {
	admin := g.Group(consts.AdminPath)

	//add middleware to check if user is admin
	adminUsers := utils.GetConfig().AdminUsers
	adminAuthMiddleware := func(c *gin.Context) {
		//check if admin access granted by auth middleware or if user is in the configuration admin users list
		if c.GetBool(consts.AdminAccess) {
			c.Next()
		} else if slices.Contains(adminUsers, c.GetString(consts.CustomerGUID)) {
			c.Next()
		} else {
			//not admin
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized - not an admin user"})
		}
	}

	admin.Use(adminAuthMiddleware)
	//add routes
	//get active customers (with scans in between dates)
	admin.GET("/activeCustomers", getActiveCustomers)
	//get customers with query params
	admin.GET("/customers", handlers.DBContextMiddleware(consts.CustomersCollection), getCustomers)
	//add delete customers data route
	admin.DELETE("/customers", deleteAllCustomerData)

	admin.PUT("/updateVulnerabilityExceptionsSeverity",
		handlers.DBContextMiddleware(consts.VulnerabilityExceptionPolicyCollection),
		updateVulnerabilityExceptionsSeverity)

	admin.PUT("/updatePostureExceptionsSeverity",
		handlers.DBContextMiddleware(consts.PostureExceptionPolicyCollection),
		updatePostureExceptionsSeverity)

	//Post V2 list query on other collections
	admin.POST("/:path/query", adminSearchCollection)
	//DELETE V2 by query on other collections
	admin.DELETE("/:path/query", adminDeleteCollection)
	//uniqueValues
	admin.POST("/:path/uniqueValues", adminAggregateCollection)
}

func updateVulnerabilityExceptionsSeverity(c *gin.Context) {
	var updateReq types.VulnerabilityExceptionsSeverityUpdate
	if err := c.ShouldBindJSON(&updateReq); err != nil {
		handlers.ResponseFailedToBindJson(c, err)
		return
	}

	filter := db.NewFilterBuilder().WithIn("vulnerabilities.name", updateReq.Cves)
	update := db.GetUpdateSetFieldCommand("vulnerabilities.$.severityScore", updateReq.SeverityScore)
	updatedCount, err := db.AdminUpdateMany(c, filter, update)
	if err != nil {
		handlers.ResponseInternalServerError(c, "failed to update vulnerability exceptions severity", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"updatedCount": updatedCount})
}

func updatePostureExceptionsSeverity(c *gin.Context) {
	var updateReq types.PostureExceptionsSeverityUpdate
	if err := c.ShouldBindJSON(&updateReq); err != nil {
		handlers.ResponseFailedToBindJson(c, err)
		return
	}

	filter := db.NewFilterBuilder().WithIn("posturePolicies.controlID", updateReq.ControlIDS)
	update := db.GetUpdateSetFieldCommand("posturePolicies.$.severityScore", updateReq.SeverityScore)
	updatedCount, err := db.AdminUpdateMany(c, filter, update)
	if err != nil {
		handlers.ResponseInternalServerError(c, "failed to update posture exceptions severity", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"updatedCount": updatedCount})
}

func adminDeleteCollection(c *gin.Context) {
	path := "/" + c.Param("path")
	apiInfo := types.GetAPIInfo(path)
	if apiInfo == nil {
		handlers.ResponseBadRequest(c, fmt.Sprintf("unknown path %s - available paths are %v", path, types.GetAllPaths()))
		return
	}
	deleteHandler := handlers.GetAdminDeleteHandler(apiInfo.DBCollection)
	if deleteHandler == nil {
		handlers.ResponseInternalServerError(c, "delete handler is nil", fmt.Errorf("no delete handler for path %s", path))
		return
	}
	//set the path collection
	c.Set(consts.Collection, apiInfo.DBCollection)
	deleteHandler(c)
}

func adminSearchCollection(c *gin.Context) {
	path := "/" + c.Param("path")
	apiInfo := types.GetAPIInfo(path)
	if apiInfo == nil {
		handlers.ResponseBadRequest(c, fmt.Sprintf("unknown path %s - available paths are %v", path, types.GetAllPaths()))
		return
	}
	queryHandler := handlers.GetAdminQueryHandler(apiInfo.DBCollection)
	if queryHandler == nil {
		handlers.ResponseInternalServerError(c, "query handler is nil", fmt.Errorf("no query handler for path %s", path))
		return
	}
	//set the path collection
	c.Set(consts.Collection, apiInfo.DBCollection)
	queryHandler(c)
}

func adminAggregateCollection(c *gin.Context) {
	path := "/" + c.Param("path")
	apiInfo := types.GetAPIInfo(path)
	if apiInfo == nil {
		handlers.ResponseBadRequest(c, fmt.Sprintf("unknown path %s - available paths are %v", path, types.GetAllPaths()))
		return
	}
	//set the path collection
	c.Set(consts.Collection, apiInfo.DBCollection)
	handlers.HandleAdminPostUniqueValuesRequestV2(c)
}

func getCustomers(c *gin.Context) {
	query := handlers.QueryParams2Filter(c, c.Request.URL.Query(), handlers.FlatQueryConfig())
	if query == nil {
		handlers.ResponseBadRequest(c, "must provide query params") //TODO: support pagination and return all customers
	}
	findOpts := db.NewFindOptions().WithFilter(query)
	if projectionParam := c.Query(consts.ProjectionParam); projectionParam != "" {
		includeFields := strings.Split(projectionParam, ",")
		if len(includeFields) > 0 {
			findOpts.Projection().Include(includeFields...)
		}
	}
	customers, error := db.AdminFind[*types.Customer](c, findOpts)
	if error != nil {
		log.LogNTraceError("getCustomers completed with errors", error, c)
		handlers.ResponseInternalServerError(c, "getCustomers completed with errors", error)
		return
	}
	handlers.DocsResponse(c, customers)
}

func deleteAllCustomerData(c *gin.Context) {
	customersGUIDs := c.QueryArray(consts.CustomersParam)
	if len(customersGUIDs) == 0 {
		handlers.ResponseMissingQueryParam(c, consts.CustomersParam)
		return
	}
	deleted, err := db.AdminDeleteCustomersDocs(c, customersGUIDs...)
	if err != nil {
		log.LogNTraceError(fmt.Sprintf("deleteAllCustomerData completed with errors. %d documents deleted", deleted), err, c)
		handlers.ResponseInternalServerError(c, fmt.Sprintf("deleted: %d, errors: %v", deleted, err), err)
		return
	}
	log.LogNTrace(fmt.Sprintf("deleteAllCustomerData completed successfully. %d documents of %d users deleted by admin %s ", deleted, len(customersGUIDs), c.GetString(consts.CustomerGUID)), c)
	c.JSON(http.StatusOK, gin.H{"deleted": deleted})

}

func deleteOldRuntimeIncidents(c *gin.Context) {
	customersGUIDs := c.QueryArray(consts.CustomersParam)
	if len(customersGUIDs) == 0 {
		handlers.ResponseMissingQueryParam(c, consts.CustomersParam)
		return
	}
	deleted, err := db.AdminDeleteCustomersDocs(c, customersGUIDs...)
	if err != nil {
		log.LogNTraceError(fmt.Sprintf("deleteAllCustomerData completed with errors. %d documents deleted", deleted), err, c)
		handlers.ResponseInternalServerError(c, fmt.Sprintf("deleted: %d, errors: %v", deleted, err), err)
		return
	}
	log.LogNTrace(fmt.Sprintf("deleteAllCustomerData completed successfully. %d documents of %d users deleted by admin %s ", deleted, len(customersGUIDs), c.GetString(consts.CustomerGUID)), c)
	c.JSON(http.StatusOK, gin.H{"deleted": deleted})

}

func getActiveCustomers(c *gin.Context) {
	defer log.LogNTraceEnterExit("activeCustomers", c)()
	var err error
	var limit, skip = 1000, 0
	if limitStr := c.Query(consts.LimitParam); limitStr != "" {
		limit, err = strconv.Atoi(limitStr)
		if err != nil {
			handlers.ResponseBadRequest(c, consts.LimitParam+" must be a number")
			return
		}
	}
	if skipStr := c.Query(consts.SkipParam); skipStr != "" {
		skip, err = strconv.Atoi(skipStr)
		if err != nil {
			handlers.ResponseBadRequest(c, consts.SkipParam+" must be a number")
			return
		}
	}
	fromDate := c.Query(consts.FromDateParam)
	if fromDate == "" {
		handlers.ResponseMissingQueryParam(c, consts.FromDateParam)
		return
	}
	if fromTime, err := time.Parse(time.RFC3339, fromDate); err != nil {
		handlers.ResponseBadRequest(c, consts.FromDateParam+" must be in RFC3339 format")
		return
	} else {
		fromDate = fromTime.UTC().Format(time.RFC3339)
	}
	toDate := c.Query(consts.ToDateParam)
	if toDate == "" {
		handlers.ResponseMissingQueryParam(c, consts.ToDateParam)
		return
	}
	if dateTime, err := time.Parse(time.RFC3339, toDate); err != nil {
		handlers.ResponseBadRequest(c, consts.ToDateParam+" must be in RFC3339 format")
		return
	} else {
		toDate = dateTime.UTC().Format(time.RFC3339)
	}
	agrs := map[string]interface{}{
		"from": fromDate,
		"to":   toDate,
	}
	result, err := db.AggregateWithTemplate[types.Customer](c, limit, skip,
		consts.ClustersCollection, db.CustomersWithScansBetweenDates, agrs)
	if err != nil {
		handlers.ResponseInternalServerError(c, "error getting active customers", err)
		return
	}
	c.JSON(http.StatusOK, result)
}
