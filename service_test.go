package main

import (
	"config-service/routes/v1/customer_config"
	"config-service/types"
	"config-service/utils"
	"config-service/utils/consts"
	"fmt"
	"net/http"
	"time"

	_ "embed"

	"github.com/armosec/armoapi-go/armotypes"
	rndStr "github.com/dchest/uniuri"

	"github.com/google/go-cmp/cmp"
)

//go:embed test_data/clusters.json
var clustersJson []byte

var newClusterCompareFilter = cmp.FilterPath(func(p cmp.Path) bool {
	switch p.String() {
	case "PortalBase.GUID", "SubscriptionDate", "LastLoginDate", "PortalBase.UpdatedTime", "ExpirationDate":
		return true
	case "PortalBase.Attributes":
		if p.Last().String() == `["alias"]` {
			return true
		}
	}
	return false
}, cmp.Ignore())

func (suite *MainTestSuite) TestCluster() {
	clusters, _ := loadJson[*types.Cluster](clustersJson)

	modifyFunc := func(cluster *types.Cluster) *types.Cluster {
		if cluster.Attributes == nil {
			cluster.Attributes = make(map[string]interface{})
		}
		if _, ok := cluster.Attributes["test"]; ok {
			cluster.Attributes["test"] = cluster.Attributes["test"].(string) + "-modified"
		} else {
			cluster.Attributes["test"] = "test"
		}
		return cluster
	}

	commonTest(suite, consts.ClusterPath, clusters, modifyFunc, newClusterCompareFilter)

	testPartialUpdate(suite, consts.ClusterPath, &types.Cluster{}, newClusterCompareFilter)

	//cluster specific tests

	//put doc without alias - expect the alias not to be deleted
	cluster := testPostDoc(suite, consts.ClusterPath, clusters[0], newClusterCompareFilter)
	alias := cluster.Attributes["alias"].(string)
	suite.NotEmpty(alias)
	delete(cluster.Attributes, "alias")
	w := suite.doRequest(http.MethodPut, consts.ClusterPath, cluster)
	suite.Equal(http.StatusOK, w.Code)
	response, err := decodeResponseArray[*types.Cluster](w)
	if err != nil || len(response) != 2 {
		panic(err)
	}
	suite.Equal(alias, response[1].Attributes["alias"].(string))

	//put doc without alias and wrong doc GUID
	cluster.GUID = "wrongGUID"
	delete(cluster.Attributes, "alias")
	testBadRequest(suite, http.MethodPut, consts.ClusterPath, errorDocumentNotFound, cluster, http.StatusNotFound)

}

//go:embed test_data/posturePolicies.json
var posturePoliciesJson []byte

var commonCmpFilter = cmp.FilterPath(func(p cmp.Path) bool {
	return p.String() == "PortalBase.GUID" || p.String() == "CreationTime" || p.String() == "CreationDate" || p.String() == "PortalBase.UpdatedTime"
}, cmp.Ignore())

func (suite *MainTestSuite) TestPostureException() {
	posturePolicies, _ := loadJson[*types.PostureExceptionPolicy](posturePoliciesJson)

	modifyFunc := func(policy *types.PostureExceptionPolicy) *types.PostureExceptionPolicy {
		if policy.Attributes == nil {
			policy.Attributes = make(map[string]interface{})
		}
		if _, ok := policy.Attributes["test"]; ok {
			policy.Attributes["test"] = policy.Attributes["test"].(string) + "-modified"
		} else {
			policy.Attributes["test"] = "test"
		}
		return policy
	}

	commonTest(suite, consts.PostureExceptionPolicyPath, posturePolicies, modifyFunc, commonCmpFilter)

	getQueries := []queryTest[*types.PostureExceptionPolicy]{
		{
			query:           "posturePolicies.controlName=Allowed hostPath&posturePolicies.controlName=Applications credentials in configuration files",
			expectedIndexes: []int{0, 1},
		},
		{
			query:           "resources.attributes.cluster=cluster1&scope.cluster=cluster3",
			expectedIndexes: []int{0, 2},
		},
		{
			query:           "scope.namespace=armo-system&scope.namespace=test-system&scope.cluster=cluster1&scope.cluster=cluster3",
			expectedIndexes: []int{0, 2},
		},
		{
			query:           "scope.namespace=armo-system&posturePolicies.frameworkName=MITRE",
			expectedIndexes: []int{1, 2},
		},
		{
			query:           "namespaceOnly=true",
			expectedIndexes: []int{1, 2},
		},
		{
			query:           "resources.attributes.cluster=cluster1",
			expectedIndexes: []int{2},
		},
		{
			query:           "posturePolicies.frameworkName=MITRE&posturePolicies.frameworkName=NSA",
			expectedIndexes: []int{0, 1, 2},
		},
		{
			query:           "posturePolicies.frameworkName=MITRE",
			expectedIndexes: []int{1, 2},
		},
		{
			query:           "posturePolicies.frameworkName=NSA",
			expectedIndexes: []int{0},
		},
	}
	testGetDeleteByNameAndQuery(suite, consts.PostureExceptionPolicyPath, consts.PolicyNameParam, posturePolicies, getQueries)
	testPartialUpdate(suite, consts.PostureExceptionPolicyPath, &types.PostureExceptionPolicy{}, commonCmpFilter)
}

//go:embed test_data/vulnerabilityPolicies.json
var vulnerabilityPoliciesJson []byte

func (suite *MainTestSuite) TestVulnerabilityPolicies() {
	vulnerabilities, _ := loadJson[*types.VulnerabilityExceptionPolicy](vulnerabilityPoliciesJson)

	modifyFunc := func(policy *types.VulnerabilityExceptionPolicy) *types.VulnerabilityExceptionPolicy {
		if policy.Attributes == nil {
			policy.Attributes = make(map[string]interface{})
		}
		if _, ok := policy.Attributes["test"]; ok {
			policy.Attributes["test"] = policy.Attributes["test"].(string) + "-modified"
		} else {
			policy.Attributes["test"] = "test"
		}
		return policy
	}

	commonTest(suite, consts.VulnerabilityExceptionPolicyPath, vulnerabilities, modifyFunc, commonCmpFilter)

	getQueries := []queryTest[*types.VulnerabilityExceptionPolicy]{
		{
			query:           "vulnerabilities.name=CVE-2005-2541&scope.cluster=dwertent",
			expectedIndexes: []int{2},
		},
		{
			query:           "scope.containerName=nginx&vulnerabilities.name=CVE-2009-5155",
			expectedIndexes: []int{0, 1},
		},
		{
			query:           "scope.containerName=nginx&vulnerabilities.name=CVE-2005-2541",
			expectedIndexes: []int{0, 2},
		},
		{
			query:           "scope.containerName=nginx&vulnerabilities.name=CVE-2005-2541&vulnerabilities.name=CVE-2005-2555",
			expectedIndexes: []int{0, 1, 2},
		},
		{
			query:           "scope.namespace=systest-ns-xpyz&designators.attributes.namespace=systest-ns-zao6",
			expectedIndexes: []int{1, 2},
		},
		{
			query:           "scope.namespace=systest-ns-xpyz&designators.attributes.namespace=systest-ns-9uqv&scope.containerName=nginx&vulnerabilities.name=CVE-2010-4756",
			expectedIndexes: []int{0},
		},
	}
	testGetDeleteByNameAndQuery(suite, consts.VulnerabilityExceptionPolicyPath, consts.PolicyNameParam, vulnerabilities, getQueries, commonCmpFilter)
	testPartialUpdate(suite, consts.VulnerabilityExceptionPolicyPath, &types.VulnerabilityExceptionPolicy{}, commonCmpFilter)
}

//go:embed test_data/customer_config/customerConfig.json
var customerConfigJson []byte

//go:embed test_data/customer_config/customerConfigMerged.json
var customerConfigMergedJson []byte

//go:embed test_data/customer_config/cluster1Config.json
var cluster1ConfigJson []byte

//go:embed test_data/customer_config/cluster1ConfigMerged.json
var cluster1ConfigMergedJson []byte

//go:embed test_data/customer_config/cluster1ConfigMergedWithDefault.json
var cluster1ConfigMergedWithDefaultJson []byte

//go:embed test_data/customer_config/cluster2Config.json
var cluster2ConfigJson []byte

//go:embed test_data/customer_config/cluster2ConfigMerged.json
var cluster2ConfigMergedJson []byte

func (suite *MainTestSuite) TestCustomerConfiguration() {

	//load test data
	defaultCustomerConfig := decode[*types.CustomerConfig](suite, defaultCustomerConfigJson)
	defaultCustomerConfig2 := decode[*types.CustomerConfig](suite, defaultCustomerConfigJson)
	customerConfig := decode[*types.CustomerConfig](suite, customerConfigJson)
	customerConfigMerged := decode[*types.CustomerConfig](suite, customerConfigMergedJson)
	cluster1Config := decode[*types.CustomerConfig](suite, cluster1ConfigJson)
	cluster1MergedConfig := decode[*types.CustomerConfig](suite, cluster1ConfigMergedJson)
	cluster1MergedWithDefaultConfig := decode[*types.CustomerConfig](suite, cluster1ConfigMergedWithDefaultJson)
	cluster2Config := decode[*types.CustomerConfig](suite, cluster2ConfigJson)
	cluster2MergedConfig := decode[*types.CustomerConfig](suite, cluster2ConfigMergedJson)

	//create compare options
	compareFilter := cmp.FilterPath(func(p cmp.Path) bool {
		return p.String() == "CreationTime" || p.String() == "GUID" || p.String() == "UpdatedTime" || p.String() == "PortalBase.UpdatedTime"
	}, cmp.Ignore())

	//TESTS

	//get all customer configs - expect only the default one
	defaultCustomerConfig = testGetDocs(suite, consts.CustomerConfigPath, []*types.CustomerConfig{defaultCustomerConfig}, compareFilter)[0]
	//post new customer config
	customerConfig = testPostDoc(suite, consts.CustomerConfigPath, customerConfig, compareFilter)
	//post cluster configs
	cluster1Config.CreationTime = ""
	cluster2Config.CreationTime = ""
	clusterConfigs := testBulkPostDocs(suite, consts.CustomerConfigPath, []*types.CustomerConfig{cluster1Config, cluster2Config}, compareFilter)
	cluster1Config = clusterConfigs[0]
	cluster2Config = clusterConfigs[1]
	suite.NotNil(cluster1Config.GetCreationTime(), "creation time should not be nil")
	suite.True(time.Since(*cluster1Config.GetCreationTime()) < time.Second, "creation time is not recent")
	suite.NotNil(cluster2Config.GetCreationTime(), "creation time should not be nil")
	suite.True(time.Since(*cluster2Config.GetCreationTime()) < time.Second, "creation time is not recent")
	//test get names list
	configNames := []string{defaultCustomerConfig.Name, customerConfig.Name, cluster1Config.Name, cluster2Config.Name}
	testGetNameList(suite, consts.CustomerConfigPath, configNames)

	// test get default config (from var)
	// set default config variable
	customer_config.SetDefaultConfigForTest(defaultCustomerConfig2)
	defaultCustomerConfig2.CustomerConfig.Settings.PostureScanConfig.ScanFrequency = "12345h"
	//by name
	path := fmt.Sprintf("%s?%s=%s", consts.CustomerConfigPath, consts.ConfigNameParam, consts.GlobalConfigName)
	testGetDoc(suite, path, defaultCustomerConfig2, compareFilter)
	//by scope
	path = fmt.Sprintf("%s?%s=%s", consts.CustomerConfigPath, consts.ScopeParam, consts.DefaultScope)
	testGetDoc(suite, path, defaultCustomerConfig2, compareFilter)
	// unset default config var
	customer_config.SetDefaultConfigForTest(nil)

	//test get default config (from cached db doc)
	//by name
	path = fmt.Sprintf("%s?%s=%s", consts.CustomerConfigPath, consts.ConfigNameParam, consts.GlobalConfigName)
	testGetDoc(suite, path, defaultCustomerConfig, compareFilter)
	//by scope
	path = fmt.Sprintf("%s?%s=%s", consts.CustomerConfigPath, consts.ScopeParam, consts.DefaultScope)
	testGetDoc(suite, path, defaultCustomerConfig, compareFilter)

	//test get merged customer config
	//by name
	path = fmt.Sprintf("%s?%s=%s", consts.CustomerConfigPath, consts.ConfigNameParam, consts.CustomerConfigName)
	testGetDoc(suite, path, customerConfigMerged, compareFilter)
	//by scope
	path = fmt.Sprintf("%s?%s=%s", consts.CustomerConfigPath, consts.ScopeParam, consts.CustomerScope)
	testGetDoc(suite, path, customerConfigMerged, compareFilter)
	//test get unmerged customer config
	//by name
	path = fmt.Sprintf("%s?%s=%s&unmerged=true", consts.CustomerConfigPath, consts.ConfigNameParam, consts.CustomerConfigName)
	testGetDoc(suite, path, customerConfig, compareFilter, compareFilter)
	//by scope
	path = fmt.Sprintf("%s?%s=%s&unmerged=true", consts.CustomerConfigPath, consts.ScopeParam, consts.CustomerScope)
	testGetDoc(suite, path, customerConfig, compareFilter)

	//test get merged cluster config by name
	//cluster1
	path = fmt.Sprintf("%s?%s=%s", consts.CustomerConfigPath, consts.ClusterNameParam, cluster1Config.GetName())
	testGetDoc(suite, path, cluster1MergedConfig, compareFilter)
	//cluster2
	path = fmt.Sprintf("%s?%s=%s", consts.CustomerConfigPath, consts.ClusterNameParam, cluster2Config.GetName())
	testGetDoc(suite, path, cluster2MergedConfig, compareFilter)
	//test get unmerged cluster config by name
	//cluster1
	path = fmt.Sprintf("%s?%s=%s&unmerged=true", consts.CustomerConfigPath, consts.ClusterNameParam, cluster1Config.GetName())
	testGetDoc(suite, path, cluster1Config, compareFilter)
	//cluster2
	path = fmt.Sprintf("%s?%s=%s&unmerged=true", consts.CustomerConfigPath, consts.ClusterNameParam, cluster2Config.GetName())
	testGetDoc(suite, path, cluster2Config, compareFilter)

	//delete customer config
	testDeleteDocByName(suite, consts.CustomerConfigPath, consts.ConfigNameParam, customerConfig)
	//get unmerged customer config - expect error 404
	path = fmt.Sprintf("%s?%s=%s&unmerged=true", consts.CustomerConfigPath, consts.ConfigNameParam, consts.CustomerConfigName)
	testBadRequest(suite, http.MethodGet, path, errorDocumentNotFound, nil, http.StatusNotFound)
	//get merged customer config - expect default config
	path = fmt.Sprintf("%s?%s=%s", consts.CustomerConfigPath, consts.ConfigNameParam, consts.CustomerConfigName)
	testGetDoc(suite, path, defaultCustomerConfig, compareFilter)
	//get merged cluster1 - expect merge with default config
	path = fmt.Sprintf("%s?%s=%s", consts.CustomerConfigPath, consts.ClusterNameParam, cluster1Config.GetName())
	testGetDoc(suite, path, cluster1MergedWithDefaultConfig, compareFilter)
	//delete cluster1 config
	testDeleteDocByName(suite, consts.CustomerConfigPath, consts.ClusterNameParam, cluster1Config)
	//get merged cluster1 - expect default config
	testGetDoc(suite, path, defaultCustomerConfig, compareFilter)
	//tets delete without name - expect error 400
	testBadRequest(suite, http.MethodDelete, consts.CustomerConfigPath, errorMissingName, nil, http.StatusBadRequest)

	//test put cluster2 config by cluster name
	oldCluster2 := clone(cluster2Config)
	cluster2Config.Settings.PostureScanConfig.ScanFrequency = "100h"
	path = fmt.Sprintf("%s?%s=%s", consts.CustomerConfigPath, consts.ClusterNameParam, cluster2Config.GetName())
	testPutDoc(suite, path, oldCluster2, cluster2Config)
	// put cluster2 config by config name
	oldCluster2 = clone(cluster2Config)
	cluster2Config.Settings.PostureControlInputs["allowedContainerRepos"] = []string{"repo1", "repo2"}
	path = fmt.Sprintf("%s?%s=%s", consts.CustomerConfigPath, consts.ConfigNameParam, cluster2Config.GetName())
	testPutDoc(suite, path, oldCluster2, cluster2Config, compareFilter)

	//put config with wrong name - expect error 400
	path = fmt.Sprintf("%s?%s=%s", consts.CustomerConfigPath, consts.ConfigNameParam, "notExist")
	testBadRequest(suite, http.MethodPut, path, errorDocumentNotFound, cluster2Config, http.StatusNotFound)
	//test put with wrong config name param - expect error 400
	path = fmt.Sprintf("%s?%s=%s", consts.CustomerConfigPath, "wrongParamName", "someName")
	c2Name := cluster2Config.Name
	cluster2Config.Name = ""
	testBadRequest(suite, http.MethodPut, path, errorMissingName, cluster2Config, http.StatusBadRequest)
	//test put with no name in path but with name in config
	cluster2Config.Name = c2Name
	testPutDoc(suite, path, cluster2Config, cluster2Config, compareFilter)

	//post costumer config again
	customerConfig = testPostDoc(suite, consts.CustomerConfigPath, customerConfig, compareFilter)
	//update it by scope param
	oldCustomerConfig := clone(customerConfig)
	customerConfig.Settings.PostureScanConfig.ScanFrequency = "11h"
	path = fmt.Sprintf("%s?%s=%s", consts.CustomerConfigPath, consts.ScopeParam, consts.CustomerScope)
	testPutDoc(suite, path, oldCustomerConfig, customerConfig, compareFilter)

}

var customerCompareFilter = cmp.FilterPath(func(p cmp.Path) bool {
	return p.String() == "SubscriptionDate" || p.String() == "PortalBase.UpdatedTime"
}, cmp.Ignore())

func (suite *MainTestSuite) TestCustomer() {
	customer := &types.Customer{
		PortalBase: armotypes.PortalBase{
			Name: "customer1",
			GUID: "new-customer-guid",
			Attributes: map[string]interface{}{
				"customer1-attr1": "customer1-attr1-value",
				"customer1-attr2": "customer1-attr2-value",
			},
		},
		Description:        "customer1 description",
		Email:              "customer1@customers.org",
		LicenseType:        "kubescape",
		InitialLicenseType: "kubescape",
	}

	//create compare options

	//create customer is public so - remove auth cookie
	suite.authCookie = ""
	//post new customer
	newCustomer := testPostDoc(suite, "/customer_tenant", customer, customerCompareFilter)
	//check creation time
	suite.NotNil(newCustomer.GetCreationTime(), "creation time should not be nil")
	suite.True(time.Since(*newCustomer.GetCreationTime()) < time.Second, "creation time is not recent")
	//check that the guid stays the same
	suite.Equal(customer.GUID, newCustomer.GUID, "customer GUID should be preserved")
	//test get customer with current customer logged in - expect error 404
	suite.login(defaultUserGUID)
	testBadRequest(suite, http.MethodGet, "/customer", errorDocumentNotFound, nil, http.StatusNotFound)
	//login new customer
	testCustomerGUID := suite.authCustomerGUID
	suite.login("new-customer-guid")
	testGetDoc(suite, "/customer", newCustomer, nil)
	//test put customer
	oldCustomer := clone(newCustomer)
	newCustomer.LicenseType = "$$$$$$"
	newCustomer.Description = "new description"
	testPutDoc(suite, "/customer", oldCustomer, newCustomer, customerCompareFilter)
	oldCustomer = clone(newCustomer)
	partialCustomer := &types.Customer{LicenseType: "partial"}
	newCustomer.LicenseType = "partial"
	testPutPartialDoc(suite, "/customer", oldCustomer, partialCustomer, newCustomer, customerCompareFilter)
	//test post with existing guid - expect error 400
	testBadRequest(suite, http.MethodPost, "/customer_tenant", errorGUIDExists, customer, http.StatusBadRequest)
	//test post customer without GUID
	customer.GUID = ""
	testBadRequest(suite, http.MethodPost, "/customer_tenant", errorMissingGUID, customer, http.StatusBadRequest)
	//restore login
	suite.login(testCustomerGUID)
}

//go:embed test_data/frameworks.json
var frameworksJson []byte
var fwCmpFilter = cmp.FilterPath(func(p cmp.Path) bool {
	return p.String() == "PortalBase.GUID" || p.String() == "CreationTime" || p.String() == "Controls" || p.String() == "PortalBase.UpdatedTime"
}, cmp.Ignore())

func (suite *MainTestSuite) TestFrameworks() {
	frameworks, _ := loadJson[*types.Framework](frameworksJson)

	modifyFunc := func(fw *types.Framework) *types.Framework {
		if fw.ControlsIDs == nil {
			fw.ControlsIDs = &[]string{}
		}
		*fw.ControlsIDs = append(*fw.ControlsIDs, "new-control"+rndStr.NewLen(5))
		return fw
	}

	commonTest(suite, consts.FrameworkPath, frameworks, modifyFunc, fwCmpFilter)

	fwCmpIgnoreControls := cmp.FilterPath(func(p cmp.Path) bool {
		return p.String() == "Controls"
	}, cmp.Ignore())

	testGetDeleteByNameAndQuery(suite, consts.FrameworkPath, consts.FrameworkNameParam, frameworks, nil, fwCmpIgnoreControls)

	//testPartialUpdate(suite, consts.FrameworkPath, &types.Framework{}, fwCmpFilter, fwCmpIgnoreControls)
}

//go:embed test_data/registryCronJob.json
var registryCronJobJson []byte

var rCmpFilter = cmp.FilterPath(func(p cmp.Path) bool {
	return p.String() == "PortalBase.GUID" || p.String() == "CreationTime" || p.String() == "CreationDate" || p.String() == "PortalBase.UpdatedTime"
}, cmp.Ignore())

func (suite *MainTestSuite) TestRegistryCronJobs() {
	registryCronJobs, _ := loadJson[*types.RegistryCronJob](registryCronJobJson)

	modifyFunc := func(r *types.RegistryCronJob) *types.RegistryCronJob {
		if r.Include == nil {
			r.Include = []string{}
		}
		r.Include = append(r.Include, "new-registry"+rndStr.NewLen(5))
		return r
	}
	commonTest(suite, consts.RegistryCronJobPath, registryCronJobs, modifyFunc, rCmpFilter)

	getQueries := []queryTest[*types.RegistryCronJob]{
		{
			query:           "clusterName=clusterA",
			expectedIndexes: []int{0, 2},
		},
		{
			query:           "registryName=registryA&registryName=registryB",
			expectedIndexes: []int{0, 1, 2},
		},
		{
			query:           "registryName=registryB",
			expectedIndexes: []int{1, 2},
		},
		{
			query:           "registryName=registryA",
			expectedIndexes: []int{0},
		},
		{
			query:           "clusterName=clusterA&registryName=registryB",
			expectedIndexes: []int{2},
		},
	}

	testGetDeleteByNameAndQuery(suite, consts.RegistryCronJobPath, consts.NameField, registryCronJobs, getQueries, rCmpFilter)

	//testPartialUpdate(suite, consts.RegistryCronJobPath, &types.RegistryCronJob{}, rCmpFilter)
}

func modifyAttribute[T types.DocContent](repo T) T {
	attributes := repo.GetAttributes()
	if attributes == nil {
		attributes = make(map[string]interface{})
	}
	if _, ok := attributes["test"]; ok {
		attributes["test"] = attributes["test"].(string) + "-modified"
	} else {
		attributes["test"] = "test"
	}
	repo.SetAttributes(attributes)
	return repo
}

//go:embed test_data/repositories.json
var repositoriesJson []byte

var repoCompareFilter = cmp.FilterPath(func(p cmp.Path) bool {
	switch p.String() {
	case "PortalBase.GUID", "CreationDate", "LastLoginDate", "PortalBase.UpdatedTime":
		return true
	case "PortalBase.Attributes":
		if p.Last().String() == `["alias"]` {
			return true
		}
	}
	return false
}, cmp.Ignore())

func (suite *MainTestSuite) TestRepository() {
	repositories, _ := loadJson[*types.Repository](repositoriesJson)

	commonTest(suite, consts.RepositoryPath, repositories, modifyAttribute[*types.Repository], repoCompareFilter)

	testPartialUpdate(suite, consts.RepositoryPath, &types.Repository{}, repoCompareFilter)

	//put doc without alias - expect the alias not to be deleted
	repo := repositories[0]
	repo.Name = "my-repo"
	repo = testPostDoc(suite, consts.RepositoryPath, repo, repoCompareFilter)
	alias := repo.Attributes["alias"].(string)
	//expect alias to use the first latter of the repo name
	suite.Equal("O", alias, "alias should be the first latter of the repo name")
	suite.NotEmpty(alias)
	delete(repo.Attributes, "alias")
	w := suite.doRequest(http.MethodPut, consts.RepositoryPath, repo)
	suite.Equal(http.StatusOK, w.Code)
	response, err := decodeResponseArray[*types.Repository](w)
	if err != nil || len(response) != 2 {
		panic(err)
	}
	repo = response[1]
	suite.Equal(alias, repo.Attributes["alias"].(string))

	//put doc without alias and wrong doc GUID
	repo1 := clone(repo)
	repo1.GUID = "wrongGUID"
	delete(repo1.Attributes, "alias")
	testBadRequest(suite, http.MethodPut, consts.RepositoryPath, errorDocumentNotFound, repo1, http.StatusNotFound)

	//change read only fields - expect them to be ignored
	repo1 = clone(repo)
	repo1.Owner = "new-owner"
	repo1.Provider = "new-provider"
	repo1.BranchName = "new-branch"
	repo1.RepoName = "new-repo"
	repo1.Attributes = map[string]interface{}{"new-attribute": "new-value"}
	w = suite.doRequest(http.MethodPut, consts.RepositoryPath, repo1)
	suite.Equal(http.StatusOK, w.Code)
	response, err = decodeResponseArray[*types.Repository](w)
	if err != nil {
		suite.FailNow(err.Error())
	}
	newDoc := response[1]
	//check updated field
	suite.Equal(newDoc.Attributes["new-attribute"], "new-value")
	//check read only fields
	suite.Equal(repo.Owner, newDoc.Owner)
	suite.Equal(repo.Provider, newDoc.Provider)
	suite.Equal(repo.BranchName, newDoc.BranchName)
	suite.Equal(repo.RepoName, newDoc.RepoName)
}

func (suite *MainTestSuite) TestCustomerNotificationConfig() {
	testCustomerGUID := "test-notification-customer-guid"
	customer := &types.Customer{
		PortalBase: armotypes.PortalBase{
			Name: "customer-test-notification-config",
			GUID: testCustomerGUID,
			Attributes: map[string]interface{}{
				"customer1-attr1": "customer1-attr1-value",
				"customer1-attr2": "customer1-attr2-value",
			},
		},
		Description:        "customer1 description",
		Email:              "customer1@customers.org",
		LicenseType:        "kubescape",
		InitialLicenseType: "kubescape",
	}
	//create customer is public so - remove auth cookie
	suite.authCookie = ""
	//post new customer
	testCustomer := testPostDoc(suite, "/customer_tenant", customer, customerCompareFilter)
	suite.Nil(testCustomer.NotificationsConfig)
	//login as customer
	suite.login(testCustomerGUID)
	//get customer notification config - should be empty
	notificationConfig := &armotypes.NotificationsConfig{}
	configPath := consts.NotificationConfigPath + "/" + testCustomerGUID
	testGetDoc(suite, configPath, notificationConfig, nil)

	//get customer notification config without guid in path - expect 404
	testBadRequest(suite, http.MethodGet, consts.NotificationConfigPath, "404 page not found", nil, http.StatusNotFound)
	//get notification config on unknown customer - expect 404
	testBadRequest(suite, http.MethodGet, consts.NotificationConfigPath+"/unknown-customer-guid", errorDocumentNotFound, nil, http.StatusNotFound)

	//Post is not served on notification config - expect 404
	testBadRequest(suite, http.MethodPost, consts.NotificationConfigPath, "404 page not found", notificationConfig, http.StatusNotFound)

	//put new notification config
	notificationConfig.UnsubscribedUsers = make(map[string][]armotypes.NotificationConfigIdentifier)
	notificationConfig.UnsubscribedUsers["user1"] = []armotypes.NotificationConfigIdentifier{{NotificationType: armotypes.NotificationTypeAll}}
	notificationConfig.UnsubscribedUsers["user2"] = []armotypes.NotificationConfigIdentifier{{NotificationType: armotypes.NotificationTypePush}}
	prevConfig := &armotypes.NotificationsConfig{}
	testPutDoc(suite, configPath, prevConfig, notificationConfig, nil)
	//update notification config
	prevConfig = clone(notificationConfig)
	notificationConfig.UnsubscribedUsers = make(map[string][]armotypes.NotificationConfigIdentifier)
	notificationConfig.UnsubscribedUsers["user3"] = []armotypes.NotificationConfigIdentifier{{NotificationType: armotypes.NotificationTypeWeekly}}
	testPutDoc(suite, configPath, prevConfig, notificationConfig, nil)

	//test unsubscribe user
	notify := armotypes.NotificationConfigIdentifier{NotificationType: armotypes.NotificationTypeWeekly}
	unsubscribePath := fmt.Sprintf("%s/%s/%s", consts.NotificationConfigPath, "unsubscribe", "user5")
	w := suite.doRequest(http.MethodPut, unsubscribePath, notify)
	suite.Equal(http.StatusOK, w.Code)
	res, err := decodeResponse[map[string]int](w)
	suite.NoError(err)
	suite.Equal(1, res["added"])
	//send the same element should update noting
	w = suite.doRequest(http.MethodPut, unsubscribePath, notify)
	suite.Equal(http.StatusOK, w.Code)
	res, err = decodeResponse[map[string]int](w)
	suite.NoError(err)
	suite.Equal(0, res["added"])
	//add another one to the same user
	notifyAll := armotypes.NotificationConfigIdentifier{NotificationType: armotypes.NotificationTypeAll}
	w = suite.doRequest(http.MethodPut, unsubscribePath, notifyAll)
	suite.Equal(http.StatusOK, w.Code)
	res, err = decodeResponse[map[string]int](w)
	suite.NoError(err)
	suite.Equal(1, res["added"])
	//add the same to a different user
	unsubscribePath = fmt.Sprintf("%s/%s/%s", consts.NotificationConfigPath, "unsubscribe", "user6")
	w = suite.doRequest(http.MethodPut, unsubscribePath, notify)
	suite.Equal(http.StatusOK, w.Code)
	res, err = decodeResponse[map[string]int](w)
	suite.NoError(err)
	suite.Equal(1, res["added"])
	//add also the 2nd element to the same user
	w = suite.doRequest(http.MethodPut, unsubscribePath, notifyAll)
	suite.Equal(http.StatusOK, w.Code)
	res, err = decodeResponse[map[string]int](w)
	suite.NoError(err)
	suite.Equal(1, res["added"])
	//remove the first element from user6
	w = suite.doRequest(http.MethodDelete, unsubscribePath, notify)
	suite.Equal(http.StatusOK, w.Code)
	res, err = decodeResponse[map[string]int](w)
	suite.NoError(err)
	suite.Equal(1, res["removed"])
	//remove from user3
	unsubscribePath = fmt.Sprintf("%s/%s/%s", consts.NotificationConfigPath, "unsubscribe", "user3")
	w = suite.doRequest(http.MethodDelete, unsubscribePath, notify)
	suite.Equal(http.StatusOK, w.Code)
	res, err = decodeResponse[map[string]int](w)
	suite.NoError(err)
	suite.Equal(1, res["removed"])
	//remove the non existing element from user3
	w = suite.doRequest(http.MethodDelete, unsubscribePath, notifyAll)
	suite.Equal(http.StatusOK, w.Code)
	res, err = decodeResponse[map[string]int](w)
	suite.NoError(err)
	suite.Equal(0, res["removed"])

	//updated the expected notification config with the changes
	notificationConfig.UnsubscribedUsers["user3"] = []armotypes.NotificationConfigIdentifier{}
	notificationConfig.UnsubscribedUsers["user6"] = []armotypes.NotificationConfigIdentifier{{NotificationType: armotypes.NotificationTypeAll}}
	notificationConfig.UnsubscribedUsers["user5"] = []armotypes.NotificationConfigIdentifier{{NotificationType: armotypes.NotificationTypeWeekly}, {NotificationType: armotypes.NotificationTypeAll}}
	//update just one field in the configuration
	notificationConfigWeekly := &armotypes.NotificationsConfig{LatestWeeklyReport: &armotypes.WeeklyReport{ClustersScannedThisWeek: 1}}
	prevConfig = clone(notificationConfig)
	notificationConfig.LatestWeeklyReport = &armotypes.WeeklyReport{ClustersScannedThisWeek: 1}
	//test partial update
	testPutPartialDoc(suite, configPath, prevConfig, notificationConfigWeekly, notificationConfig, nil)
	//make sure not other customer fields are changed
	updatedCustomer := clone(testCustomer)
	updatedCustomer.NotificationsConfig = notificationConfig
	updatedCustomer = testGetDoc(suite, "/customer", updatedCustomer, customerCompareFilter)
	//check the the customer update date is updated
	suite.NotNil(updatedCustomer.GetUpdatedTime(), "update time should not be nil")
	suite.True(time.Since(*updatedCustomer.GetUpdatedTime()) < time.Second, "update time is not recent")

	//test add push report
	var ignoreTime = cmp.FilterValues(func(x, y time.Time) bool { return true }, cmp.Ignore())
	pushTime := time.Now().UTC()
	pushReport := &armotypes.PushReport{Timestamp: pushTime, ReportGUID: "push-guid", Cluster: "cluster1"}
	pushReportPath := fmt.Sprintf("%s/%s/%s", consts.NotificationConfigPath, "latestPushReport", "cluster1")
	w = suite.doRequest(http.MethodPut, pushReportPath, pushReport)
	suite.Equal(http.StatusOK, w.Code)
	res, err = decodeResponse[map[string]int](w)
	suite.NoError(err)
	suite.Equal(1, res["modified"])
	notificationConfig.LatestPushReports = map[string]*armotypes.PushReport{}
	notificationConfig.LatestPushReports["cluster1"] = pushReport
	testGetDoc(suite, configPath, notificationConfig, ignoreTime)
	//add one for cluster2
	pushReportPath = fmt.Sprintf("%s/%s/%s", consts.NotificationConfigPath, "latestPushReport", "cluster2")
	w = suite.doRequest(http.MethodPut, pushReportPath, pushReport)
	suite.Equal(http.StatusOK, w.Code)
	res, err = decodeResponse[map[string]int](w)
	suite.NoError(err)
	suite.Equal(1, res["modified"])
	notificationConfig.LatestPushReports["cluster2"] = pushReport
	testGetDoc(suite, configPath, notificationConfig, ignoreTime)
	//delete cluster1
	pushReportPath = fmt.Sprintf("%s/%s/%s", consts.NotificationConfigPath, "latestPushReport", "cluster1")
	w = suite.doRequest(http.MethodDelete, pushReportPath, nil)
	suite.Equal(http.StatusOK, w.Code)
	res, err = decodeResponse[map[string]int](w)
	suite.NoError(err)
	suite.Equal(1, res["modified"])
	delete(notificationConfig.LatestPushReports, "cluster1")
	testGetDoc(suite, configPath, notificationConfig, ignoreTime)
}

func (suite *MainTestSuite) TestCustomerState() {
	testCustomerGUID := "test-state-customer-guid"
	customer := &types.Customer{
		PortalBase: armotypes.PortalBase{
			Name: "customer-test-state",
			GUID: testCustomerGUID,
			Attributes: map[string]interface{}{
				"customer1-attr1": "customer1-attr1-value",
				"customer1-attr2": "customer1-attr2-value",
			},
		},
		Description:        "customer1 description",
		Email:              "customer1@customers.org",
		LicenseType:        "kubescape",
		InitialLicenseType: "kubescape",
	}
	//create customer is public so - remove auth cookie
	suite.authCookie = ""
	//post new customer
	testCustomer := testPostDoc(suite, "/customer_tenant", customer, customerCompareFilter)
	suite.Nil(testCustomer.State)
	//login as customer
	suite.login(testCustomerGUID)

	//get customer state - should return the default state (onboarding completed)
	state := &armotypes.CustomerState{
		Onboarding: &armotypes.CustomerOnboarding{
			Completed: utils.BoolPointer(true),
		},
		GettingStarted: &armotypes.GettingStartedChecklist{
			GettingStartedDismissed: utils.BoolPointer(false),
		},
	}
	statePath := consts.CustomerStatePath + "/" + testCustomerGUID
	testGetDoc(suite, statePath, state, nil)

	//get customer state without guid in path - expect 404
	testBadRequest(suite, http.MethodGet, consts.CustomerStatePath, "404 page not found", nil, http.StatusNotFound)
	//get state on unknown customer - expect 404
	testBadRequest(suite, http.MethodGet, consts.CustomerStatePath+"/unknown-customer-guid", errorDocumentNotFound, nil, http.StatusNotFound)

	//Post is not served on state - expect 404
	testBadRequest(suite, http.MethodPost, consts.CustomerStatePath, "404 page not found", state, http.StatusNotFound)

	//put new state
	prevState := clone(state)
	state.Onboarding.CompanySize = utils.StringPointer("1000")
	state.Onboarding.Completed = utils.BoolPointer(false)
	state.Onboarding.Interests = []string{"a", "b"}
	state.GettingStarted = &armotypes.GettingStartedChecklist{
		GettingStartedDismissed: utils.BoolPointer(true),
	}

	// mongo has a millisecond precision while golang time.Time has nanosecond precision, so we need to wait at least 1 millisecond to reflect the change
	timeBeforeUpdate := time.Now()
	time.Sleep(1000 * time.Millisecond)

	testPutDoc(suite, statePath, prevState, state, nil)

	// update state - "GettingStarted = nil" should not be updated
	// we skip checking it in testPutDoc because it will returned as a non-null object and comparison will fail
	prevState = clone(state)
	state.Onboarding.Completed = utils.BoolPointer(true)
	expectState := clone(state)
	state.GettingStarted = nil
	testPutPartialDoc(suite, statePath, prevState, state, expectState)
	state = clone(expectState)

	//make sure not other customer fields are changed
	updatedCustomer := clone(testCustomer)
	updatedCustomer.State = state
	updatedCustomer = testGetDoc(suite, "/customer", updatedCustomer, customerCompareFilter)
	//check the the customer update date is updated
	suite.NotNil(updatedCustomer.GetUpdatedTime(), "update time should not be nil")
	suite.Truef(updatedCustomer.GetUpdatedTime().After(timeBeforeUpdate), "update time should be updated")

	// try updating state with false value
	prevState = clone(state)
	state.Onboarding.Completed = utils.BoolPointer(false)
	testPutDoc(suite, statePath, prevState, state, nil)
}

func (suite *MainTestSuite) TestActiveSubscription() {
	testCustomerGUID := "test-stripe-customer-guid"
	customer := &types.Customer{
		PortalBase: armotypes.PortalBase{
			Name: "customer-test-stripe-customer",
			GUID: testCustomerGUID,
			Attributes: map[string]interface{}{
				"customer1-attr1": "customer1-attr1-value",
				"customer1-attr2": "customer1-attr2-value",
			},
		},
		Description:        "customer1 description",
		Email:              "customer1@customers.org",
		LicenseType:        "kubescape",
		InitialLicenseType: "kubescape",
	}
	//create customer is public so - remove auth cookie
	suite.authCookie = ""
	//post new customer
	testCustomer := testPostDoc(suite, "/customer_tenant", customer, customerCompareFilter)
	suite.Nil(testCustomer.ActiveSubscription)

	// login as customer
	suite.login(testCustomerGUID)

	// define activeSubscription with licenseType default value "free"
	activeSubscription := &armotypes.Subscription{LicenseType: armotypes.LicenseTypeFree}

	// construct activeSubscription api path with customer guid
	activeSubscriptionPath := consts.ActiveSubscriptionPath + "/" + testCustomerGUID

	// test getting doc of the customer.
	testGetDoc(suite, activeSubscriptionPath, customer.ActiveSubscription, nil)

	//get activeSubscription without guid in path - expect 404
	testBadRequest(suite, http.MethodGet, consts.ActiveSubscriptionPath, "404 page not found", nil, http.StatusNotFound)

	//get activeSubscription on unknown customer - expect 404
	testBadRequest(suite, http.MethodGet, consts.ActiveSubscriptionPath+"/unknown-customer-guid", errorDocumentNotFound, nil, http.StatusNotFound)

	//Post is not served on activeSubscription - expect 404
	testBadRequest(suite, http.MethodPost, consts.ActiveSubscriptionPath, "404 page not found", activeSubscription, http.StatusNotFound)

	// define new activeSubscription values
	activeSubscription.StripeCustomerID = "test-customer-id"
	activeSubscription.StripeSubscriptionID = "test-subscription-id"
	activeSubscription.SubscriptionStatus = "active"
	activeSubscription.CancelAtPeriodEnd = utils.BoolPointer(false)

	// mongo has a millisecond precision while golang time.Time has nanosecond precision, so we need to wait at least 1 millisecond to reflect the change
	timeBeforeUpdate := time.Now()
	time.Sleep(1000 * time.Millisecond)

	// put new activeSubscription - oldDoc is nil has we haven't configure it yet.
	testPutDoc(suite, activeSubscriptionPath, nil, activeSubscription, nil)

	// update activeSubscription partially
	// we skip checking it in testPutDoc because it will returned as a non-null object and comparison will fail
	prevActiveSubscription := clone(activeSubscription)
	activeSubscription.SubscriptionStatus = "canceled"
	expectActiveSubscription := clone(activeSubscription)
	activeSubscription.StripeSubscriptionID = "test-subscription-id"
	testPutPartialDoc(suite, activeSubscriptionPath, prevActiveSubscription, activeSubscription, expectActiveSubscription)
	activeSubscription = clone(expectActiveSubscription)

	// make sure no other customer fields are changed
	updatedCustomer := clone(testCustomer)
	updatedCustomer.ActiveSubscription = activeSubscription
	updatedCustomer = testGetDoc(suite, "/customer", updatedCustomer, customerCompareFilter)

	// check the the customer update date is updated
	suite.NotNil(updatedCustomer.GetUpdatedTime(), "update time should not be nil")
	suite.Truef(updatedCustomer.GetUpdatedTime().After(timeBeforeUpdate), "update time should be updated")
}
