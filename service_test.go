package main

import (
	"config-service/types"
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
	case "PortalBase.GUID", "SubscriptionDate", "LastLoginDate", "PortalBase.UpdatedTime":
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

	//test get default config
	//by name
	path := fmt.Sprintf("%s?%s=%s", consts.CustomerConfigPath, consts.ConfigNameParam, consts.GlobalConfigName)
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
	testPutDoc(suite, path, oldCluster2, cluster2Config)

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

func (suite *MainTestSuite) TestAdminAndUsers() {
	const (
		user1 = "user1-guid"
		user2 = "f5f360bb-c233-4c33-a9af-5692e7795d61"
		user3 = "2ce5daf4-e28d-4e6e-a239-03fda048070b"
		admin = "admin-user-guid"
	)

	users := []string{user1, user2, user3}

	clusters, clustersNames := loadJson[*types.Cluster](clustersJson)
	frameworks, frameworksNames := loadJson[*types.Framework](frameworksJson)
	posturePolices, policiesNames := loadJson[*types.PostureExceptionPolicy](posturePoliciesJson)
	vulnerabilityPolicies, vulnerabilityNames := loadJson[*types.VulnerabilityExceptionPolicy](vulnerabilityPoliciesJson)
	repositories, repositoriesNames := loadJson[*types.Repository](repositoriesJson)
	registryCronJobs, registryCronJobNames := loadJson[*types.RegistryCronJob](registryCronJobJson)

	populateUser := func(userGUID string) {
		suite.login(userGUID)
		testBulkPostDocs(suite, consts.ClusterPath, clusters, newClusterCompareFilter)
		testBulkPostDocs(suite, consts.FrameworkPath, frameworks, fwCmpFilter)
		testBulkPostDocs(suite, consts.PostureExceptionPolicyPath, posturePolices, commonCmpFilter)
		testBulkPostDocs(suite, consts.VulnerabilityExceptionPolicyPath, vulnerabilityPolicies, commonCmpFilter)
		testBulkPostDocs(suite, consts.RepositoryPath, repositories, repoCompareFilter)
		testBulkPostDocs(suite, consts.RegistryCronJobPath, registryCronJobs, rCmpFilter)

		customer := &types.Customer{
			PortalBase: armotypes.PortalBase{
				Name: userGUID,
				GUID: userGUID,
			},
		}
		testPostDoc(suite, consts.TenantPath, customer, customerCompareFilter)
	}

	verifyUserData := func(userGUID string) {
		suite.login(userGUID)
		testGetNameList(suite, consts.ClusterPath, clustersNames)
		testGetNameList(suite, consts.FrameworkPath, frameworksNames)
		testGetNameList(suite, consts.PostureExceptionPolicyPath, policiesNames)
		testGetNameList(suite, consts.VulnerabilityExceptionPolicyPath, vulnerabilityNames)
		testGetNameList(suite, consts.RepositoryPath, repositoriesNames)
		testGetNameList(suite, consts.RegistryCronJobPath, registryCronJobNames)

		customer := &types.Customer{
			PortalBase: armotypes.PortalBase{
				Name: userGUID,
				GUID: userGUID,
			},
		}

		testGetDoc(suite, "/customer", customer, customerCompareFilter)
	}

	verifyUserDataDeleted := func(userGUID string) {
		suite.login(userGUID)
		testGetNameList(suite, consts.ClusterPath, nil)
		testGetNameList(suite, consts.FrameworkPath, nil)
		testGetNameList(suite, consts.PostureExceptionPolicyPath, nil)
		testGetNameList(suite, consts.VulnerabilityExceptionPolicyPath, nil)
		testGetNameList(suite, consts.RepositoryPath, nil)
		testGetNameList(suite, consts.RegistryCronJobPath, nil)
		testBadRequest(suite, http.MethodGet, consts.CustomerPath, errorDocumentNotFound, nil, http.StatusNotFound)

	}

	for _, userGUID := range users {
		populateUser(userGUID)
		verifyUserData(userGUID)
	}
	//login as admin
	suite.loginAsAdmin("a-admin-guid")
	//delete users2 and users3 data
	deleteUsersUrls := fmt.Sprintf("%s/customers?%s=%s&%s=%s", consts.AdminPath, consts.CustomersParam, user2, consts.CustomersParam, user3)
	type deletedResponse struct {
		Deleted int64 `json:"deleted"`
	}
	w := suite.doRequest(http.MethodDelete, deleteUsersUrls, nil)
	suite.Equal(http.StatusOK, w.Code)
	response, err := decodeResponse[*deletedResponse](w)
	if err != nil {
		suite.FailNow(err.Error())
	}
	//expect 2 customers doc and all what they have
	deletedCount := 2 * (1 + len(clusters) + len(frameworks) + len(posturePolices) + len(vulnerabilityPolicies) + len(repositories) + len(registryCronJobs))
	suite.Equal(int64(deletedCount), response.Deleted)
	//verify user1 data is still there
	verifyUserData(user1)
	//verify user2 and user3 data is gone
	for _, userGUID := range users[1:] {
		verifyUserDataDeleted(userGUID)
	}

	//make sure regular user can't use admin api
	suite.login(user1)
	testBadRequest(suite, http.MethodDelete, deleteUsersUrls, errorNotAdminUser, nil, http.StatusUnauthorized)

	//populate user2 again
	suite.login(user2)
	populateUser(user2)
	verifyUserData(user2)
	//test customer delete they own data with  DELETE /customer api
	w = suite.doRequest(http.MethodDelete, consts.CustomerPath, nil)
	suite.Equal(http.StatusOK, w.Code)
	response, err = decodeResponse[*deletedResponse](w)
	if err != nil {
		suite.FailNow(err.Error())
	}

	deletedCount = 1 + len(clusters) + len(frameworks) + len(posturePolices) + len(vulnerabilityPolicies) + len(repositories) + len(registryCronJobs)
	suite.Equal(int64(deletedCount), response.Deleted)
	//verify user2 data is gone
	verifyUserDataDeleted(user2)
	//verify user1 data is still there
	verifyUserData(user1)

	//login as admin from the config admins list
	suite.login(admin)
	//delete user1 data
	deleteUsersUrls = fmt.Sprintf("%s/customers?%s=%s", consts.AdminPath, consts.CustomersParam, user1)
	w = suite.doRequest(http.MethodDelete, deleteUsersUrls, nil)
	suite.Equal(http.StatusOK, w.Code)
	response, err = decodeResponse[*deletedResponse](w)
	if err != nil {
		suite.FailNow(err.Error())
	}

	suite.Equal(int64(deletedCount), response.Deleted)
	//verify user1 data is gone
	verifyUserDataDeleted(user1)

	//test bad delete customers request with no users
	suite.loginAsAdmin("other-admin-guid")
	deleteUsersUrls = fmt.Sprintf("%s/customers", consts.AdminPath)
	testBadRequest(suite, http.MethodDelete, deleteUsersUrls, errorMissingQueryParams(consts.CustomersParam), nil, http.StatusBadRequest)

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
	notificationConfig.UnsubscribedUsers = make(map[string]armotypes.NotificationConfigIdentifier)
	notificationConfig.UnsubscribedUsers["user1"] = armotypes.NotificationConfigIdentifier{NotificationType: armotypes.NotificationTypeAll}
	notificationConfig.UnsubscribedUsers["user2"] = armotypes.NotificationConfigIdentifier{NotificationType: armotypes.NotificationTypePush}
	prevConfig := &armotypes.NotificationsConfig{}
	testPutDoc(suite, configPath, prevConfig, notificationConfig, nil)
	//update notification config
	prevConfig = clone(notificationConfig)
	notificationConfig.UnsubscribedUsers = make(map[string]armotypes.NotificationConfigIdentifier)
	notificationConfig.UnsubscribedUsers["user3"] = armotypes.NotificationConfigIdentifier{NotificationType: armotypes.NotificationTypeWeekly}
	testPutDoc(suite, configPath, prevConfig, notificationConfig, nil)

	//make sure not other customer fields are changed
	updatedCustomer := clone(testCustomer)
	updatedCustomer.NotificationsConfig = notificationConfig
	updatedCustomer = testGetDoc(suite, "/customer", updatedCustomer, customerCompareFilter)
	//check the the customer update date is updated
	suite.NotNil(updatedCustomer.GetUpdatedTime(), "update time should not be nil")
	suite.True(time.Since(*updatedCustomer.GetUpdatedTime()) < time.Second, "update time is not recent")
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
	suite.Nil(testCustomer.NotificationsConfig)
	//login as customer
	suite.login(testCustomerGUID)

	//get customer state - should return the default state (onboarding completed)
	state := &armotypes.CustomerState{
		Onboarding: &armotypes.CustomerOnboarding{
			Completed: true,
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
	state.Onboarding.CompanySize = "1000"
	state.Onboarding.Completed = false
	state.Onboarding.Interests = []string{"a", "b"}
	state.GettingStartedDismissed = true
	prevState := &armotypes.CustomerState{
		Onboarding: &armotypes.CustomerOnboarding{
			Completed: true,
		},
	}
	testPutDoc(suite, statePath, prevState, state, nil)

	//update state
	prevState = clone(state)
	state.Onboarding.Completed = true
	testPutDoc(suite, statePath, prevState, state, nil)

	//make sure not other customer fields are changed
	updatedCustomer := clone(testCustomer)
	updatedCustomer.State = state
	updatedCustomer = testGetDoc(suite, "/customer", updatedCustomer, customerCompareFilter)
	//check the the customer update date is updated
	suite.NotNil(updatedCustomer.GetUpdatedTime(), "update time should not be nil")
	suite.True(time.Since(*updatedCustomer.GetUpdatedTime()) < time.Second, "update time is not recent")
}
