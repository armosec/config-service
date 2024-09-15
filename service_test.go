package main

import (
	"config-service/db/mongo"
	"config-service/routes/v1/customer_config"
	"config-service/types"
	"config-service/utils"
	"config-service/utils/consts"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/aws/smithy-go/ptr"
	"go.uber.org/zap"

	_ "embed"

	"github.com/armosec/armoapi-go/armotypes"
	"github.com/armosec/armoapi-go/identifiers"
	"github.com/armosec/armoapi-go/notifications"
	"github.com/armosec/armosec-infra/kdr"
	"github.com/armosec/armosec-infra/workflows"

	rndStr "github.com/dchest/uniuri"

	"github.com/google/go-cmp/cmp"
)

//go:embed test_data/clusters.json
var clustersJson []byte

//go:embed test_data/cloud-credentials.json
var cloudAccountsJson []byte

//go:embed test_data/runtimeIncidentPolicyReq1.json
var runtimeIncidentPolicyReq1 []byte

//go:embed test_data/runtimeIncidentPolicyReq2.json
var runtimeIncidentPolicyReq2 []byte

//go:embed test_data/runtimeIncidentPolicyReq3.json
var runtimeIncidentPolicyReq3 []byte

//go:embed test_data/workflows.json
var workflowsJson []byte

//go:embed test_data/workflowsSortReq.json
var workflowsSortReq []byte

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

	projectedDocs := []*types.Cluster{
		{
			PortalBase: armotypes.PortalBase{
				Name: "arn-aws-eks-eu-west-1-221581667315-cluster-deel-dev-test",
			},
		},
		{
			PortalBase: armotypes.PortalBase{
				Name: "bez",
			},
		},
		{
			PortalBase: armotypes.PortalBase{
				Name: "moshe-super-cluster",
			},
		},
	}

	searchQueries := []searchTest{
		{
			testName:        "get all",
			expectedIndexes: []int{0, 1, 2},
			listRequest:     armotypes.V2ListRequest{},
		},
		{
			testName:        "get first page",
			expectedIndexes: []int{0, 1},
			listRequest: armotypes.V2ListRequest{
				PageSize: ptr.Int(2),
				PageNum:  ptr.Int(0),
			},
		},
		{
			testName:        "get multiple names",
			expectedIndexes: []int{1, 2},
			listRequest: armotypes.V2ListRequest{
				OrderBy: "name:asc",
				InnerFilters: []map[string]string{
					{
						"name": "bez,moshe-super-cluster",
					},
				},
			},
		},

		//field or match
		{
			testName:        "field or match",
			expectedIndexes: []int{1},
			listRequest: armotypes.V2ListRequest{
				OrderBy: "name:asc",
				InnerFilters: []map[string]string{
					{
						"name": "bez",
					},
				},
			},
		},
		//fields and match
		{
			testName:        "fields and match",
			expectedIndexes: []int{1},
			listRequest: armotypes.V2ListRequest{
				OrderBy: "name:asc",
				InnerFilters: []map[string]string{
					{
						"attributes.kind": "k8s",
						"name":            "bez",
					},
				},
			},
		},

		//filters exist operator
		{
			testName:        "filters exist operator",
			expectedIndexes: []int{0, 1, 2},
			listRequest: armotypes.V2ListRequest{
				OrderBy: "name:asc",
				InnerFilters: []map[string]string{
					{
						"attributes.kind": "|exists",
					},
				},
			},
		},
		//like match
		{
			testName:        "like ignorecase match",
			expectedIndexes: []int{1},
			listRequest: armotypes.V2ListRequest{
				OrderBy: "name:asc",
				InnerFilters: []map[string]string{
					{
						"name": "BeZ|like&ignorecase",
					},
				},
			},
		},
		{
			testName:        "like with multi results",
			expectedIndexes: []int{0, 1, 2},
			listRequest: armotypes.V2ListRequest{
				OrderBy: "name:asc",
				InnerFilters: []map[string]string{
					{
						"attributes.kind": "k8|like",
					},
				},
			},
		},
		//projection test
		{
			testName:         "projection test",
			expectedIndexes:  []int{0, 1, 2},
			projectedResults: true,
			listRequest: armotypes.V2ListRequest{
				OrderBy:    "name:asc",
				FieldsList: []string{"name"},
				InnerFilters: []map[string]string{
					{
						"attributes.kind": "k8s",
					},
				},
			},
		},
		//field or match
		{
			testName:        "attributes.workerNodes.max = 6 query match test",
			expectedIndexes: []int{1},
			listRequest: armotypes.V2ListRequest{
				OrderBy: "name:asc",
				InnerFilters: []map[string]string{
					{
						"attributes.workerNodes.max": "6",
					},
				},
			},
		},
		{
			testName:        "attributes.clusterAPIServerInfo.platform = linux/amd64 query match test",
			expectedIndexes: []int{1, 2},
			listRequest: armotypes.V2ListRequest{
				OrderBy: "name:asc",
				InnerFilters: []map[string]string{
					{
						"attributes.clusterAPIServerInfo.platform": "linux/amd64",
					},
				},
			},
		},
	}

	testPostV2ListRequest(suite, consts.ClusterPath, clusters, projectedDocs, searchQueries, newClusterCompareFilter, ignoreTime)

	uniqueValues := []uniqueValueTest{
		{
			testName: "unique names",
			uniqueValuesRequest: armotypes.UniqueValuesRequestV2{
				Fields: map[string]string{
					"name": "",
				},
			},
			expectedResponse: armotypes.UniqueValuesResponseV2{
				Fields: map[string][]string{
					"name": {"arn-aws-eks-eu-west-1-221581667315-cluster-deel-dev-test", "bez", "moshe-super-cluster"},
				},
				FieldsCount: map[string][]armotypes.UniqueValuesResponseFieldsCount{
					"name": {
						{

							Field: "arn-aws-eks-eu-west-1-221581667315-cluster-deel-dev-test",
							Count: 1,
						},
						{
							Field: "bez",
							Count: 1,
						},
						{
							Field: "moshe-super-cluster",
							Count: 1,
						},
					},
				},
			},
		},
		{
			testName: "unique names with filter",
			uniqueValuesRequest: armotypes.UniqueValuesRequestV2{
				Fields: map[string]string{
					"name": "",
				},
				InnerFilters: []map[string]string{
					{
						"attributes.clusterAPIServerInfo.platform": "linux/amd64",
					},
				},
			},
			expectedResponse: armotypes.UniqueValuesResponseV2{
				Fields: map[string][]string{
					"name": {"bez", "moshe-super-cluster"},
				},
				FieldsCount: map[string][]armotypes.UniqueValuesResponseFieldsCount{
					"name": {
						{
							Field: "bez",
							Count: 1,
						},
						{
							Field: "moshe-super-cluster",
							Count: 1,
						},
					},
				},
			},
		},
	}

	testUniqueValues(suite, consts.ClusterPath, clusters, uniqueValues, newClusterCompareFilter, ignoreTime)

	testPartialUpdate(suite, consts.ClusterPath, &types.Cluster{}, newClusterCompareFilter)

	testGetByName(suite, consts.ClusterPath, "name", clusters, newClusterCompareFilter, ignoreTime)

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
	return p.String() == "PortalBase.GUID" || p.String() == "GUID" || p.String() == "CreationTime" || p.String() == "CreationDate" || p.String() == "PortalBase.UpdatedTime" || p.String() == "UpdatedTime"
}, cmp.Ignore())

var ignoreName = cmp.FilterPath(func(p cmp.Path) bool {
	return p.String() == "PortalBase.Name"
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
	testOptions := testOptions[*types.PostureExceptionPolicy]{
		mandatoryName:          true,
		uniqueName:             true,
		customGUID:             false,
		allGUIDsAsInnerFilters: true,
	}

	commonTestWithOptions(suite, consts.PostureExceptionPolicyPath, posturePolicies, modifyFunc, testOptions, commonCmpFilter)

	getQueries := []queryTest[*types.PostureExceptionPolicy]{
		{
			query:           "posturePolicies.controlName=Allowed hostPath&posturePolicies.controlName=Applications credentials in configuration files",
			expectedIndexes: []int{0, 1},
		},
		{
			query:           "resources.attributes.cluster=cluster1&scope.cluster=cluster3",
			expectedIndexes: []int{0, 2, 3},
		},
		{
			query:           "scope.namespace=armo-system&scope.namespace=test-system&scope.cluster=cluster1&scope.cluster=cluster3",
			expectedIndexes: []int{0, 2, 3},
		},
		{
			query:           "scope.namespace=armo-system&posturePolicies.frameworkName=MITRE",
			expectedIndexes: []int{1, 2, 3},
		},
		{
			query:           "namespaceOnly=true",
			expectedIndexes: []int{1, 2, 3},
		},
		{
			query:           "resources.attributes.cluster=cluster1",
			expectedIndexes: []int{2, 3},
		},
		{
			query:           "posturePolicies.frameworkName=MITRE&posturePolicies.frameworkName=NSA",
			expectedIndexes: []int{0, 1, 2, 3},
		},
		{
			query:           "posturePolicies.frameworkName=MITRE",
			expectedIndexes: []int{1, 2, 3},
		},
		{
			query:           "posturePolicies.frameworkName=NSA",
			expectedIndexes: []int{0},
		},
	}
	testGetDeleteByNameAndQuery(suite, consts.PostureExceptionPolicyPath, consts.PolicyNameParam, posturePolicies, getQueries)
	testPartialUpdate(suite, consts.PostureExceptionPolicyPath, &types.PostureExceptionPolicy{}, commonCmpFilter)

	uniqueValues := []uniqueValueTest{
		{
			testName: "unique control names",
			uniqueValuesRequest: armotypes.UniqueValuesRequestV2{
				Fields: map[string]string{
					"posturePolicies.controlName": "",
				},
			},
			expectedResponse: armotypes.UniqueValuesResponseV2{
				Fields: map[string][]string{
					"posturePolicies.controlName": {"Allowed hostPath", "Applications credentials in configuration files", "List Kubernetes secrets"},
				},
				FieldsCount: map[string][]armotypes.UniqueValuesResponseFieldsCount{
					"posturePolicies.controlName": {
						{

							Field: "Allowed hostPath",
							Count: 1,
						},
						{
							Field: "Applications credentials in configuration files",
							Count: 1,
						},
						{
							Field: "List Kubernetes secrets",
							Count: 2,
						},
					},
				},
			},
		},
	}

	testUniqueValues(suite, consts.PostureExceptionPolicyPath, posturePolicies, uniqueValues, commonCmpFilter)

	searchtests := []searchTest{
		{
			testName: "test resources array search",
			listRequest: armotypes.V2ListRequest{
				InnerFilters: []map[string]string{
					{
						"resources.attributes.cluster": "cluster1",
					},
				},
			},
			expectedIndexes: []int{2, 3},
		},
		{
			testName: "test OR score with missing date",
			listRequest: armotypes.V2ListRequest{
				InnerFilters: []map[string]string{
					{
						"posturePolicies.severityScore": "1,2,3",
						"expirationDate":                "|missing",
					},
				},
			},
			expectedIndexes: []int{0, 2},
		},
		{
			testName: "test missing date",
			listRequest: armotypes.V2ListRequest{
				InnerFilters: []map[string]string{
					{
						"expirationDate": "|missing",
					},
				},
			},
			expectedIndexes: []int{0, 1, 2, 4},
		},
		{
			testName: "test OR score with existing date",
			listRequest: armotypes.V2ListRequest{
				InnerFilters: []map[string]string{
					{
						"posturePolicies.severityScore": "1,2,3",
						"expirationDate":                "|exists",
					},
				},
			},
			expectedIndexes: []int{3},
		},
	}
	oldException := &types.PostureExceptionPolicy{
		PortalBase: armotypes.PortalBase{
			GUID: "1c00292c-713d-4be7-9bff-3f5e14fa4068",
			Name: "exception_old_bba5f2df536533fb348631207c85ed39",
		},
		PolicyType:   "postureExceptionPolicy",
		CreationTime: "2022-01-19T11:56:06.740591",
		PosturePolicies: []armotypes.PosturePolicy{
			{
				FrameworkName: "ArmoBest",
				ControlName:   "Allowed hostPath",
				SeverityScore: 5,
			},
		},
	}
	collection := mongo.GetWriteCollection(consts.PostureExceptionPolicyCollection)
	if _, err := collection.InsertOne(context.Background(), oldException); err != nil {
		suite.FailNow("Failed to insert posturePolicyException", err.Error())
	}
	posturePolicies = append(posturePolicies, oldException)
	testPostV2ListRequest(suite, consts.PostureExceptionPolicyPath, posturePolicies, nil, searchtests, commonCmpFilter)

	//test add exception with expiry date then put it back to null
	exceptionPolicy := posturePolicies[3]
	w := suite.doRequest(http.MethodPost, consts.PostureExceptionPolicyPath, exceptionPolicy)
	suite.Equal(http.StatusCreated, w.Code)
	response, err := decodeResponse[*types.PostureExceptionPolicy](w)
	if err != nil {
		panic(err)
	}
	suite.NotNil(response.ExpirationDate)
	policyGuid := response.GUID
	exceptionPolicy.ExpirationDate = nil
	exceptionPolicy.GUID = policyGuid
	w = suite.doRequest(http.MethodPut, consts.PostureExceptionPolicyPath, exceptionPolicy)
	suite.Equal(http.StatusOK, w.Code)
	w = suite.doRequest(http.MethodGet, path.Join(consts.PostureExceptionPolicyPath, policyGuid), nil)
	suite.Equal(http.StatusOK, w.Code)
	response, err = decodeResponse[*types.PostureExceptionPolicy](w)
	if err != nil {
		panic(err)
	}
	suite.Nil(response.ExpirationDate)
}

//go:embed test_data/collaborationConfigs.json
var collaborationConfigsJson []byte

func (suite *MainTestSuite) TestCollaborationConfigs() {
	collaborations, _ := loadJson[*types.CollaborationConfig](collaborationConfigsJson)

	modifyFunc := func(policy *types.CollaborationConfig) *types.CollaborationConfig {
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

	testOptions := testOptions[*types.CollaborationConfig]{
		mandatoryName: true,
		uniqueName:    true,
		renameAllowed: true,
		customGUID:    false,
	}

	commonTestWithOptions(suite, consts.CollaborationConfigPath, collaborations, modifyFunc, testOptions, commonCmpFilter)

	getQueries := []queryTest[*types.CollaborationConfig]{
		{
			query:           "provider=slack&provider=ms-teams",
			expectedIndexes: []int{0, 3},
		},
		{
			query:           "context.cloud.name=example-io&context.cloud.name=cyberarmor-io",
			expectedIndexes: []int{1, 2},
		},
		{
			query:           "name=collab2",
			expectedIndexes: []int{2},
		},
	}
	testGetDeleteByNameAndQuery(suite, consts.CollaborationConfigPath, consts.PolicyNameParam, collaborations, getQueries, commonCmpFilter)
	testPartialUpdate(suite, consts.CollaborationConfigPath, &types.CollaborationConfig{PortalBase: armotypes.PortalBase{Name: "collabPartial"}}, commonCmpFilter, ignoreName)

	// test case for delete by provider
	testBulkPostDocs(suite, consts.CollaborationConfigPath, collaborations, commonCmpFilter)

	v2Req := armotypes.V2ListRequest{
		InnerFilters: []map[string]string{
			{
				"provider": "slack,jira,not-existing",
			},
		}}
	w := suite.doRequest(http.MethodDelete, consts.CollaborationConfigPath+"/query", v2Req)
	suite.Equal(http.StatusOK, w.Code)
	res, err := decodeResponse[map[string]int](w)
	suite.NoError(err)
	suite.Equal(3, res["deletedCount"])
	// verify deletion happened
	w = suite.doRequest(http.MethodPost, consts.CollaborationConfigPath+"/query", v2Req)
	suite.Equal(http.StatusOK, w.Code)
	var searchRes types.SearchResult[types.CollaborationConfig]
	searchRes, err = decodeResponse[types.SearchResult[types.CollaborationConfig]](w)
	suite.NoError(err)
	suite.Equal(0, len(searchRes.Response))
	suite.Equal(0, searchRes.Total.Value)
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

	commonTest(suite, consts.VulnerabilityExceptionPolicyPath, vulnerabilities, modifyFunc, commonCmpFilter, ignoreTime)

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
	testGetDeleteByNameAndQuery(suite, consts.VulnerabilityExceptionPolicyPath, consts.PolicyNameParam, vulnerabilities, getQueries, commonCmpFilter, ignoreTime)
	testPartialUpdate(suite, consts.VulnerabilityExceptionPolicyPath, &types.VulnerabilityExceptionPolicy{}, commonCmpFilter, ignoreTime)

	uniqueValues := []uniqueValueTest{
		{
			testName: "unique vulnerabilities name",
			uniqueValuesRequest: armotypes.UniqueValuesRequestV2{
				Fields: map[string]string{
					"vulnerabilities.name": "",
				},
			},
			expectedResponse: armotypes.UniqueValuesResponseV2{
				Fields: map[string][]string{
					"vulnerabilities.name": {"CVE-2005-2541", "CVE-2005-2555", "CVE-2007-5686", "CVE-2009-5155", "CVE-2010-4756"},
				},
				FieldsCount: map[string][]armotypes.UniqueValuesResponseFieldsCount{
					"vulnerabilities.name": {
						{

							Field: "CVE-2005-2541",
							Count: 2,
						},
						{
							Field: "CVE-2005-2555",
							Count: 1,
						},
						{
							Field: "CVE-2007-5686",
							Count: 3,
						},
						{
							Field: "CVE-2009-5155",
							Count: 2,
						},
						{
							Field: "CVE-2010-4756",
							Count: 1,
						},
					},
				},
			},
		},
	}

	testUniqueValues(suite, consts.VulnerabilityExceptionPolicyPath, vulnerabilities, uniqueValues, commonCmpFilter, ignoreTime)

	projectedDocs := []*types.VulnerabilityExceptionPolicy{
		{
			PortalBase: armotypes.PortalBase{
				Name: "1656325224.51881314",
			},
		},
		{
			PortalBase: armotypes.PortalBase{
				Name: "1660467597.8207463",
			},
		},
		{
			PortalBase: armotypes.PortalBase{
				Name: "1660475024.9930612",
			},
		},
	}

	searchQueries := []searchTest{
		{
			testName:         "search in designator attributes array",
			expectedIndexes:  []int{1, 2},
			projectedResults: true,
			listRequest: armotypes.V2ListRequest{
				FieldsList: []string{"name"},
				InnerFilters: []map[string]string{
					{
						"designators.attributes.cluster": "dwertent",
					},
				},
			},
		},
		{
			testName:         "test filter by range of dates",
			expectedIndexes:  []int{1, 2},
			projectedResults: true,
			listRequest: armotypes.V2ListRequest{
				FieldsList: []string{"name"},
				InnerFilters: []map[string]string{
					{
						"expirationDate": "2022-08-10T11:03:45.494851Z&2022-08-11T08:59:58.297650Z|range",
					},
				},
			},
		},
	}

	testPostV2ListRequest(suite, consts.VulnerabilityExceptionPolicyPath, vulnerabilities, projectedDocs, searchQueries, commonCmpFilter, ignoreTime)
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
	createTime, _ := time.Parse(time.RFC3339, time.Now().UTC().Format(time.RFC3339))
	cluster1Config.CreationTime = ""
	cluster2Config.CreationTime = ""
	clusterConfigs := testBulkPostDocs(suite, consts.CustomerConfigPath, []*types.CustomerConfig{cluster1Config, cluster2Config}, compareFilter)
	cluster1Config = clusterConfigs[0]
	cluster2Config = clusterConfigs[1]
	suite.NotNil(cluster1Config.GetCreationTime(), "creation time should not be nil")
	suite.True(createTime.Before(*cluster1Config.GetCreationTime()) || createTime.Equal(*cluster1Config.GetCreationTime()), "creation time is not recent")
	suite.NotNil(cluster2Config.GetCreationTime(), "creation time should not be nil")
	suite.True(createTime.Before(*cluster2Config.GetCreationTime()) || createTime.Equal(*cluster2Config.GetCreationTime()), "creation time is not recent")
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
	oldCluster2 := Clone(cluster2Config)
	cluster2Config.Settings.PostureScanConfig.ScanFrequency = "100h"
	path = fmt.Sprintf("%s?%s=%s", consts.CustomerConfigPath, consts.ClusterNameParam, cluster2Config.GetName())
	testPutDoc(suite, path, oldCluster2, cluster2Config)
	// put cluster2 config by config name
	oldCluster2 = Clone(cluster2Config)
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
	oldCustomerConfig := Clone(customerConfig)
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
	createTime, _ := time.Parse(time.RFC3339, time.Now().UTC().Format(time.RFC3339))
	newCustomer := testPostDoc(suite, "/customer_tenant", customer, customerCompareFilter)
	//check creation time
	suite.NotNil(newCustomer.GetCreationTime(), "creation time should not be nil")
	suite.True(createTime.Before(*newCustomer.GetCreationTime()) || createTime.Equal(*newCustomer.GetCreationTime()), "creation time is not recent")
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
	oldCustomer := Clone(newCustomer)
	newCustomer.LicenseType = "$$$$$$"
	newCustomer.Description = "new description"
	testPutDoc(suite, "/customer", oldCustomer, newCustomer, customerCompareFilter)
	oldCustomer = Clone(newCustomer)
	partialCustomer := &types.Customer{LicenseType: "partial"}
	newCustomer.LicenseType = "partial"
	testPutPartialDoc(suite, "/customer", oldCustomer, partialCustomer, newCustomer, customerCompareFilter)
	//test post with existing guid - expect error 400
	testBadRequest(suite, http.MethodPost, "/customer_tenant", errorGUIDExists, customer, http.StatusConflict)
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
	repo1 := Clone(repo)
	repo1.GUID = "wrongGUID"
	delete(repo1.Attributes, "alias")
	testBadRequest(suite, http.MethodPut, consts.RepositoryPath, errorDocumentNotFound, repo1, http.StatusNotFound)

	//change fields (used to be read only)
	repo1 = Clone(repo)
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
	suite.Equal(repo1.Owner, newDoc.Owner)
	suite.Equal(repo1.Provider, newDoc.Provider)
	suite.Equal(repo1.BranchName, newDoc.BranchName)
	suite.Equal(repo1.RepoName, newDoc.RepoName)

	req := armotypes.V2ListRequest{
		OrderBy: "name:asc",
		InnerFilters: []map[string]string{
			{
				"repoName": "new-repo",
			},
		},
	}
	w = suite.doRequest(http.MethodPost, consts.RepositoryPath+"/query", req)
	suite.Equal(http.StatusOK, w.Code)
	var result types.SearchResult[types.Repository]
	err = json.Unmarshal(w.Body.Bytes(), &result)
	if err != nil {
		suite.FailNow(err.Error())
	}
	suite.Equal(1, len(result.Response))
	if len(result.Response) > 0 {
		suite.Equal("new-repo", result.Response[0].RepoName)
	}

	w = suite.doRequest(http.MethodGet, consts.RepositoryPath+"?name=my-repo", nil)
	suite.Equal(http.StatusOK, w.Code)
	byNameRepo := types.Repository{}
	err = json.Unmarshal(w.Body.Bytes(), &byNameRepo)
	if err != nil {
		suite.FailNow(err.Error())
	}
	suite.Equal(repo.GUID, byNameRepo.GUID)
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
	notificationConfig := &notifications.NotificationsConfig{}
	configPath := consts.NotificationConfigPath + "/" + testCustomerGUID
	testGetDoc(suite, configPath, notificationConfig, nil)

	//get customer notification config without guid in path - expect 404
	testBadRequest(suite, http.MethodGet, consts.NotificationConfigPath, "404 page not found", nil, http.StatusNotFound)
	//get notification config on unknown customer - expect 404
	testBadRequest(suite, http.MethodGet, consts.NotificationConfigPath+"/unknown-customer-guid", errorDocumentNotFound, nil, http.StatusNotFound)

	//Post is not served on notification config - expect 404
	testBadRequest(suite, http.MethodPost, consts.NotificationConfigPath, "404 page not found", notificationConfig, http.StatusNotFound)

	//put new notification config
	notificationConfig.UnsubscribedUsers = make(map[string][]notifications.NotificationConfigIdentifier)
	notificationConfig.UnsubscribedUsers["user1"] = []notifications.NotificationConfigIdentifier{{NotificationType: notifications.NotificationTypeVulnerabilityNewFix}}
	notificationConfig.UnsubscribedUsers["user2"] = []notifications.NotificationConfigIdentifier{{NotificationType: notifications.NotificationTypePush}}
	prevConfig := &notifications.NotificationsConfig{}
	testPutDoc(suite, configPath, prevConfig, notificationConfig, nil)
	//update notification config
	prevConfig = Clone(notificationConfig)
	notificationConfig.UnsubscribedUsers = make(map[string][]notifications.NotificationConfigIdentifier)
	notificationConfig.UnsubscribedUsers["user3"] = []notifications.NotificationConfigIdentifier{{NotificationType: notifications.NotificationTypeWeekly}}
	testPutDoc(suite, configPath, prevConfig, notificationConfig, nil)

	//test unsubscribe user
	notify := notifications.NotificationConfigIdentifier{NotificationType: notifications.NotificationTypeWeekly}
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
	notifyAll := notifications.NotificationConfigIdentifier{NotificationType: notifications.NotificationTypeVulnerabilityNewFix}
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
	notificationConfig.UnsubscribedUsers["user3"] = []notifications.NotificationConfigIdentifier{}
	notificationConfig.UnsubscribedUsers["user6"] = []notifications.NotificationConfigIdentifier{{NotificationType: notifications.NotificationTypeVulnerabilityNewFix}}
	notificationConfig.UnsubscribedUsers["user5"] = []notifications.NotificationConfigIdentifier{{NotificationType: notifications.NotificationTypeWeekly}, {NotificationType: notifications.NotificationTypeVulnerabilityNewFix}}

	//test put delete multiple elements
	notifyPush := notifications.NotificationConfigIdentifier{NotificationType: notifications.NotificationTypePush}
	//add 2 elements to user10
	unsubscribePath = fmt.Sprintf("%s/%s/%s", consts.NotificationConfigPath, "unsubscribe", "user10")
	w = suite.doRequest(http.MethodPut, unsubscribePath, []notifications.NotificationConfigIdentifier{notify, notifyPush})
	suite.Equal(http.StatusOK, w.Code)
	res, err = decodeResponse[map[string]int](w)
	suite.NoError(err)
	suite.Equal(1, res["added"])
	//remove non existing element from user10
	w = suite.doRequest(http.MethodDelete, unsubscribePath, []notifications.NotificationConfigIdentifier{notifyAll})
	suite.Equal(http.StatusOK, w.Code)
	res, err = decodeResponse[map[string]int](w)
	suite.NoError(err)
	suite.Equal(0, res["removed"])
	// add 3 elements to user11
	unsubscribePath = fmt.Sprintf("%s/%s/%s", consts.NotificationConfigPath, "unsubscribe", "user11")
	w = suite.doRequest(http.MethodPut, unsubscribePath, []notifications.NotificationConfigIdentifier{notify, notifyPush, notifyAll})
	suite.Equal(http.StatusOK, w.Code)
	res, err = decodeResponse[map[string]int](w)
	suite.NoError(err)
	suite.Equal(1, res["added"])
	// remove 2 elements from user11
	w = suite.doRequest(http.MethodDelete, unsubscribePath, []notifications.NotificationConfigIdentifier{notifyPush, notifyAll})
	suite.Equal(http.StatusOK, w.Code)
	res, err = decodeResponse[map[string]int](w)
	suite.NoError(err)
	suite.Equal(1, res["removed"])
	//set expected state for the notification config
	notificationConfig.UnsubscribedUsers["user10"] = []notifications.NotificationConfigIdentifier{{NotificationType: notifications.NotificationTypeWeekly}, {NotificationType: notifications.NotificationTypePush}}
	notificationConfig.UnsubscribedUsers["user11"] = []notifications.NotificationConfigIdentifier{{NotificationType: notifications.NotificationTypeWeekly}}

	//update just one field in the configuration
	notificationConfigWeekly := &notifications.NotificationsConfig{LatestWeeklyReport: &notifications.WeeklyReport{ClustersScannedThisWeek: 1}}
	prevConfig = Clone(notificationConfig)
	notificationConfig.LatestWeeklyReport = &notifications.WeeklyReport{ClustersScannedThisWeek: 1}
	//test partial update
	updateTime, _ := time.Parse(time.RFC3339, time.Now().UTC().Format(time.RFC3339))
	testPutPartialDoc(suite, configPath, prevConfig, notificationConfigWeekly, notificationConfig, nil)
	//make sure not other customer fields are changed
	updatedCustomer := Clone(testCustomer)
	updatedCustomer.NotificationsConfig = notificationConfig
	updatedCustomer = testGetDoc(suite, "/customer", updatedCustomer, customerCompareFilter)
	//check the the customer update date is updated
	suite.NotNil(updatedCustomer.GetUpdatedTime(), "update time should not be nil")
	suite.True(updateTime.Before(*updatedCustomer.GetUpdatedTime()) || updateTime.Equal(*updatedCustomer.GetUpdatedTime()), "update time is not recent")
	//test add push report
	pushTime := time.Now().UTC()
	pushReport := &notifications.PushReport{Timestamp: pushTime, ReportGUID: "push-guid", Cluster: "cluster1"}
	pushReportPath := fmt.Sprintf("%s/%s/%s", consts.NotificationConfigPath, "latestPushReport", "cluster1")
	w = suite.doRequest(http.MethodPut, pushReportPath, pushReport)
	suite.Equal(http.StatusOK, w.Code)
	res, err = decodeResponse[map[string]int](w)
	suite.NoError(err)
	suite.Equal(1, res["modified"])
	notificationConfig.LatestPushReports = map[string]*notifications.PushReport{}
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
	prevState := Clone(state)
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
	prevState = Clone(state)
	state.Onboarding.Completed = utils.BoolPointer(true)
	expectState := Clone(state)
	state.GettingStarted = nil
	testPutPartialDoc(suite, statePath, prevState, state, expectState)
	state = Clone(expectState)

	//make sure not other customer fields are changed
	updatedCustomer := Clone(testCustomer)
	updatedCustomer.State = state
	updatedCustomer = testGetDoc(suite, "/customer", updatedCustomer, customerCompareFilter)
	//check the the customer update date is updated
	suite.NotNil(updatedCustomer.GetUpdatedTime(), "update time should not be nil")
	suite.Truef(updatedCustomer.GetUpdatedTime().After(timeBeforeUpdate), "update time should be updated")

	// try updating state with false value
	prevState = Clone(state)
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
	prevActiveSubscription := Clone(activeSubscription)
	activeSubscription.SubscriptionStatus = "canceled"
	expectActiveSubscription := Clone(activeSubscription)
	activeSubscription.StripeSubscriptionID = "test-subscription-id"
	testPutPartialDoc(suite, activeSubscriptionPath, prevActiveSubscription, activeSubscription, expectActiveSubscription)
	activeSubscription = Clone(expectActiveSubscription)

	// make sure no other customer fields are changed
	updatedCustomer := Clone(testCustomer)
	updatedCustomer.ActiveSubscription = activeSubscription
	updatedCustomer = testGetDoc(suite, "/customer", updatedCustomer, customerCompareFilter)

	// check the the customer update date is updated
	suite.NotNil(updatedCustomer.GetUpdatedTime(), "update time should not be nil")
	suite.Truef(updatedCustomer.GetUpdatedTime().After(timeBeforeUpdate), "update time should be updated")
}

//go:embed test_data/attack-chain-states.json
var attackChainStatesJson []byte

func (suite *MainTestSuite) TestAttackChainsStates() {
	attackChainStates, _ := loadJson[*types.ClusterAttackChainState](attackChainStatesJson)

	modifyDocFunc := func(doc *types.ClusterAttackChainState) *types.ClusterAttackChainState {
		docCloned := Clone(doc)
		currentTime := time.Now().UTC()
		docCloned.LastPostureScanTriggered = currentTime.Format(time.RFC3339)
		return docCloned
	}

	testOpts := testOptions[*types.ClusterAttackChainState]{
		uniqueName:    true,
		mandatoryName: true,
		customGUID:    false,
		skipPutTests:  false,
	}

	commonTestWithOptions(suite, consts.AttackChainsPath, attackChainStates, modifyDocFunc, testOpts, commonCmpFilter, ignoreTime)

	projectedDocs := []*types.ClusterAttackChainState{
		{
			ClusterName:              "aaa",
			LastPostureScanTriggered: "2022-04-28T14:00:00.147901",
			LastTimeEngineCompleted:  "2022-04-28T14:59:44.147901",
		},
		{
			ClusterName:              "bbb",
			LastPostureScanTriggered: "2022-04-28T14:00:00.147901",
			LastTimeEngineCompleted:  "2022-04-28T14:59:44.147901",
		},
		{
			ClusterName:              "ccc",
			LastPostureScanTriggered: "2022-04-28T14:00:00.147901",
			LastTimeEngineCompleted:  "2022-04-28T14:59:44.147901",
		},
	}

	searchQueries := []searchTest{
		{
			testName:        "search by clusterName",
			expectedIndexes: []int{2},
			listRequest: armotypes.V2ListRequest{
				InnerFilters: []map[string]string{
					{
						"clusterName": "ccc",
					},
				},
			},
		},
		{
			testName:        "get all (max 150)",
			expectedIndexes: []int{0, 1, 2},
			listRequest:     armotypes.V2ListRequest{},
		},
		{
			testName:         "get all (max 150) with selected field only",
			expectedIndexes:  []int{0, 1, 2},
			projectedResults: true,
			listRequest: armotypes.V2ListRequest{
				FieldsList: []string{"clusterName", "lastPostureScanTriggered", "lastTimeEngineCompleted"},
			},
		},
	}

	testPostV2ListRequest(suite, consts.AttackChainsPath, attackChainStates, projectedDocs, searchQueries, commonCmpFilter, ignoreTime)

	//add some docs for unique values tests
	moreDocs := []*types.ClusterAttackChainState{
		{
			PortalBase: armotypes.PortalBase{
				GUID: "11111",
				Name: "minikube-aaa",
			},
			ClusterName: "minikube-a",
		},
		{
			PortalBase: armotypes.PortalBase{
				GUID: "11111121",
				Name: "minikube-bbb",
			},
			ClusterName: "minikube-b",
		},
		{
			PortalBase: armotypes.PortalBase{
				GUID: "1132111",
				Name: "minikube-a",
			},
			ClusterName: "minikube-a",
		},
		{
			PortalBase: armotypes.PortalBase{
				GUID: "223xx",
				Name: "minikube-b",
			},
			ClusterName: "minikube-b",
		},

		{
			PortalBase: armotypes.PortalBase{
				GUID: "2234",
				Name: "minikube-d",
			},
			ClusterName: "minikube-d",
		},
	}
	attackChainStates = append(attackChainStates, moreDocs...)

	uniqueValues := []uniqueValueTest{
		{
			testName: "unique cluster names",
			uniqueValuesRequest: armotypes.UniqueValuesRequestV2{
				Fields: map[string]string{
					"clusterName": "",
				},
			},
			expectedResponse: armotypes.UniqueValuesResponseV2{
				Fields: map[string][]string{
					"clusterName": {"aaa", "bbb", "ccc", "minikube-a", "minikube-b", "minikube-d"},
				},
				FieldsCount: map[string][]armotypes.UniqueValuesResponseFieldsCount{
					"clusterName": {
						{

							Field: "aaa",
							Count: 1,
						},
						{
							Field: "bbb",
							Count: 1,
						},
						{
							Field: "ccc",
							Count: 1,
						},
						{
							Field: "minikube-a",
							Count: 2,
						}, {
							Field: "minikube-b",
							Count: 2,
						}, {
							Field: "minikube-d",
							Count: 1,
						},
					},
				},
			},
		},
		{
			testName: "unique lastPostureScanTriggered and lastTimeEngineCompleted",
			uniqueValuesRequest: armotypes.UniqueValuesRequestV2{
				Fields: map[string]string{
					"lastPostureScanTriggered": "",
					"lastTimeEngineCompleted":  "",
				},
			},
			expectedResponse: armotypes.UniqueValuesResponseV2{
				Fields: map[string][]string{
					"lastPostureScanTriggered": {"2022-04-28T14:00:00.147901"},
					"lastTimeEngineCompleted":  {"2022-04-28T14:59:44.147901"},
				},
				FieldsCount: map[string][]armotypes.UniqueValuesResponseFieldsCount{
					"lastPostureScanTriggered": {
						{

							Field: "2022-04-28T14:00:00.147901",
							Count: 3,
						},
					},
					"lastTimeEngineCompleted": {
						{

							Field: "2022-04-28T14:59:44.147901",
							Count: 3,
						},
					},
				},
			},
		},
	}

	testUniqueValues(suite, consts.AttackChainsPath, attackChainStates, uniqueValues, commonCmpFilter, ignoreTime)
}

func getIncidentsMocks() []*types.RuntimeIncident {
	ts, err := time.Parse(time.RFC3339, "2022-04-28T14:00:00.147000Z")
	if err != nil {
		panic(err)
	}
	tsNanos := ts.UnixNano()
	runtimeIncidents := []*types.RuntimeIncident{
		{
			RuntimeIncident: kdr.RuntimeIncident{
				PortalBase: armotypes.PortalBase{
					Name: "incident1",
					GUID: "1c0e9d28-7e71-4370-999e-9b3e8f69a648",
				},
				RuntimeAlert: armotypes.RuntimeAlert{
					RuntimeAlertK8sDetails: armotypes.RuntimeAlertK8sDetails{
						ClusterName: "cluster1",
						Namespace:   "namespace1",
					},
				},
				Severity: "low",
				RuntimeIncidentResource: kdr.RuntimeIncidentResource{
					Designators: identifiers.PortalDesignator{
						DesignatorType: identifiers.DesignatorAttributes,
						Attributes:     map[string]string{},
					},
				},
			},
		},
		{
			RuntimeIncident: kdr.RuntimeIncident{
				PortalBase: armotypes.PortalBase{
					Name: "incident2",
					GUID: "1c0e9d28-7e71-4370-999e-9b3e8f69a647",
				},
				RuntimeIncidentResource: kdr.RuntimeIncidentResource{
					Designators: identifiers.PortalDesignator{
						DesignatorType: identifiers.DesignatorAttributes,
						Attributes:     map[string]string{},
					},
				},
				Severity: "high",
			},
		},
		{
			RuntimeIncident: kdr.RuntimeIncident{
				PortalBase: armotypes.PortalBase{
					Name: "incident3",
					GUID: "1c0e9d28-7e71-4370-999e-9b3e8f69a646",
				},
				RuntimeAlert: armotypes.RuntimeAlert{
					BaseRuntimeAlert: armotypes.BaseRuntimeAlert{
						Timestamp: ts,
					},
				},
				Severity: "medium",
				RelatedAlerts: []kdr.RuntimeAlert{
					{
						RuntimeAlert: armotypes.RuntimeAlert{
							Message:  "msg1",
							HostName: "host1",
							BaseRuntimeAlert: armotypes.BaseRuntimeAlert{
								Timestamp:   ts,
								Nanoseconds: uint64(tsNanos) + 200,
							},
						},
					},
					{
						RuntimeAlert: armotypes.RuntimeAlert{
							Message:  "msg2",
							HostName: "host2",
							BaseRuntimeAlert: armotypes.BaseRuntimeAlert{
								Timestamp:   ts,
								Nanoseconds: uint64(tsNanos) + 100,
							},
						},
					},
					{
						RuntimeAlert: armotypes.RuntimeAlert{
							Message:  "msg3",
							HostName: "host3",
							BaseRuntimeAlert: armotypes.BaseRuntimeAlert{
								Nanoseconds: uint64(tsNanos),
								Timestamp:   ts,
							},
						},
					},
				},
				RuntimeIncidentResource: kdr.RuntimeIncidentResource{
					Designators: identifiers.PortalDesignator{
						DesignatorType: identifiers.DesignatorAttributes,
						Attributes:     map[string]string{},
					},
				},
			},
		},
	}
	return runtimeIncidents
}

func (suite *MainTestSuite) TestRuntimeIncidents() {
	runtimeIncidents := getIncidentsMocks()
	modifyDocFunc := func(doc *types.RuntimeIncident) *types.RuntimeIncident {
		docCloned := Clone(doc)
		docCloned.RelatedAlerts = append(docCloned.RelatedAlerts, kdr.RuntimeAlert{
			RuntimeAlert: armotypes.RuntimeAlert{
				Message: "msg" + rndStr.New(),
			},
		})
		return docCloned
	}

	testOpts := testOptions[*types.RuntimeIncident]{
		mandatoryName: false,
		customGUID:    true,
		skipPutTests:  false,
	}
	cmpFilters := cmp.FilterPath(func(p cmp.Path) bool {
		// "RuntimeIncident.RuntimeAlert.RuntimeAlertK8sDetails.HostNetwork"
		fieldPath := p.String()
		return fieldPath == "RuntimeIncident.PortalBase.GUID" || fieldPath == "RuntimeIncident.GUID" || fieldPath == "RuntimeIncident.CreationTime" || fieldPath == "RuntimeIncident.CreationDate" || fieldPath == "RuntimeIncident.PortalBase.UpdatedTime" || fieldPath == "RuntimeIncident.UpdatedTime" || fieldPath == "CreationDayDate" || fieldPath == "ResolveDayDate" || strings.HasPrefix(fieldPath, "RuntimeIncident.RelatedAlerts")
	}, cmp.Ignore())
	commonTestWithOptions(suite, consts.RuntimeIncidentPath, runtimeIncidents, modifyDocFunc,
		testOpts, cmpFilters, ignoreTime)

	zeroTimeIntAsStr := "-62135596800000"

	uniqueValues := []uniqueValueTest{
		{
			testName: "unique cluster names",
			uniqueValuesRequest: armotypes.UniqueValuesRequestV2{
				Fields: map[string]string{
					"clusterName": "",
				},
			},
			expectedResponse: armotypes.UniqueValuesResponseV2{
				Fields: map[string][]string{
					"clusterName": {"", "cluster1"},
				},
				FieldsCount: map[string][]armotypes.UniqueValuesResponseFieldsCount{
					"clusterName": {
						{
							Field: "",
							Count: 2,
						},
						{

							Field: "cluster1",
							Count: 1,
						},
					},
				},
			},
		},
		{
			testName: "unique incidents severities",
			uniqueValuesRequest: armotypes.UniqueValuesRequestV2{
				Fields: map[string]string{
					"incidentSeverity": "",
				},
			},
			expectedResponse: armotypes.UniqueValuesResponseV2{
				Fields: map[string][]string{
					"incidentSeverity": {"high", "low", "medium"},
				},
				FieldsCount: map[string][]armotypes.UniqueValuesResponseFieldsCount{
					"incidentSeverity": {
						{

							Field: "high",
							Count: 1,
						},
						{
							Field: "low",
							Count: 1,
						},
						{
							Field: "medium",
							Count: 1,
						},
					},
				},
			},
		},
		{
			testName: "unique timestamps",
			uniqueValuesRequest: armotypes.UniqueValuesRequestV2{
				Fields: map[string]string{
					"timestamp": "",
				},
			},
			expectedResponse: armotypes.UniqueValuesResponseV2{
				Fields: map[string][]string{
					"timestamp": {zeroTimeIntAsStr, "1651154400147"},
				},
				FieldsCount: map[string][]armotypes.UniqueValuesResponseFieldsCount{
					"timestamp": {
						{
							Field: zeroTimeIntAsStr, // zero time
							Count: 2,
						},
						{

							Field: "1651154400147", // 2022-04-28T14:00:00.147901
							Count: 1,
						},
					},
				},
			},
		},
	}

	testUniqueValues(suite, consts.RuntimeIncidentPath, runtimeIncidents, uniqueValues, cmpFilters, ignoreTime)
}

func (suite *MainTestSuite) TestRuntimeIncidentsDismiss() {
	runtimeIncidents := getIncidentsMocks()
	// test put is dismissed
	w := suite.doRequest(http.MethodPost, consts.RuntimeIncidentPath, runtimeIncidents)
	suite.Equal(http.StatusCreated, w.Code)
	// get it first:
	w = suite.doRequest(http.MethodGet, consts.RuntimeIncidentPath, nil)
	suite.Equal(http.StatusOK, w.Code)
	docs, err := decodeResponseArray[types.RuntimeIncident](w)
	if err != nil {
		suite.FailNow(err.Error())
	}
	incident := docs[0]
	incident.RelatedAlerts = nil
	incident.IsDismissed = true
	w = suite.doRequest(http.MethodPut, consts.RuntimeIncidentPath+"/"+incident.GUID, incident)
	suite.Equal(http.StatusOK, w.Code)
	updateIncidents, err := decodeResponse[[]types.RuntimeIncident](w)
	suite.NoError(err)
	updateIncident := updateIncidents[1]
	suite.True(updateIncident.IsDismissed)
	nowDate := time.Now().UTC().Format(time.RFC3339[:10])
	suite.Equal(nowDate, updateIncident.ResolveDayDate.Format(time.RFC3339[:10]))
	suite.Equal(nowDate, updateIncident.CreationDayDate.Format(time.RFC3339[:10]))
	nowDateUnix, _ := time.Parse(time.RFC3339[:10], nowDate)
	nowDate = fmt.Sprintf("%d000", nowDateUnix.Unix())
	// test unique values of dismissed/created incidents
	uniqueValuesTests := []uniqueValueTest{
		{
			testName: "unique incidents created",
			uniqueValuesRequest: armotypes.UniqueValuesRequestV2{
				Fields: map[string]string{
					"creationDayDate": "",
				},
			},
			expectedResponse: armotypes.UniqueValuesResponseV2{
				Fields: map[string][]string{
					"creationDayDate": {nowDate},
				},
				FieldsCount: map[string][]armotypes.UniqueValuesResponseFieldsCount{
					"creationDayDate": {
						{
							Field: nowDate,
							Count: 3,
						},
					},
				},
			},
		},
		{
			testName: "unique incidents resolved date",
			uniqueValuesRequest: armotypes.UniqueValuesRequestV2{
				Fields: map[string]string{
					"resolveDayDate": "",
				},
			},
			expectedResponse: armotypes.UniqueValuesResponseV2{
				Fields: map[string][]string{
					"resolveDayDate": {nowDate},
				},
				FieldsCount: map[string][]armotypes.UniqueValuesResponseFieldsCount{
					"resolveDayDate": {
						{
							Field: nowDate,
							Count: 1,
						},
					},
				},
			},
		},
	}
	for _, test := range uniqueValuesTests {
		w := suite.doRequest(http.MethodPost, consts.RuntimeIncidentPath+"/uniqueValues", test.uniqueValuesRequest)
		suite.Equal(http.StatusOK, w.Code)
		var result armotypes.UniqueValuesResponseV2
		err := json.Unmarshal(w.Body.Bytes(), &result)
		if err != nil {
			suite.FailNow(err.Error())
		}
		if test.expectedResponse.FieldsCount == nil {
			//skipping fields count comparison
			test.expectedResponse.FieldsCount = result.FieldsCount
		}
		suite.Equal(test.expectedResponse, result, "Unexpected result: %s", test.testName)
	}

	//bulk delete all docs
	guids := []string{}
	for _, doc := range docs {
		guids = append(guids, doc.GetGUID())
	}
	testBulkDeleteByGUIDWithBody(suite, consts.RuntimeIncidentPath, guids)
}
func (suite *MainTestSuite) TestRuntimeAlerts() {
	// feed incidents with nested alerts
	runtimeIncidents := getIncidentsMocks()
	w := suite.doRequest(http.MethodPost, consts.RuntimeIncidentPath, runtimeIncidents)
	suite.Equal(http.StatusCreated, w.Code)
	// assure no alerts returned in any incident
	w = suite.doRequest(http.MethodGet, consts.RuntimeIncidentPath, nil)
	suite.Equal(http.StatusOK, w.Code)
	docs, err := decodeResponseArray[types.RuntimeIncident](w)
	if err != nil {
		suite.FailNow(err.Error())
	}
	for _, doc := range docs {
		suite.Nil(doc.RelatedAlerts)
	}
	// get incident with alert3 by query
	incidentRequest := armotypes.V2ListRequest{
		PageSize: ptr.Int(1),
		PageNum:  ptr.Int(0),
		InnerFilters: []map[string]string{
			{
				"relatedAlerts.message": "msg3",
			},
		},
	}
	w = suite.doRequest(http.MethodPost, consts.RuntimeIncidentPath+"/query", incidentRequest)
	suite.Equal(http.StatusOK, w.Code)
	resp, err := decodeResponse[armotypes.V2ListResponseGeneric[[]types.RuntimeIncident]](w)
	if err != nil {
		suite.FailNow(err.Error())
	}
	suite.Equal(1, resp.Total.Value)
	suite.Len(resp.Response, 1)
	// assuer it has "no alerts"
	suite.Len(resp.Response[0].RelatedAlerts, 0)
	// assuer the calling PUT for this incident (with empty alerts) will not change the alerts
	w = suite.doRequest(http.MethodPut, consts.RuntimeIncidentPath+"/"+resp.Response[0].GUID, resp.Response[0])
	suite.Equal(http.StatusOK, w.Code)
	w = suite.doRequest(http.MethodPut, consts.RuntimeIncidentPath, resp.Response[0])
	suite.Equal(http.StatusOK, w.Code)
	// get alerts of this incident guid paginated
	alertRequest := armotypes.V2ListRequest{
		PageSize: ptr.Int(1),
		PageNum:  ptr.Int(0),
	}
	w = suite.doRequest(http.MethodPost, consts.RuntimeAlertPath+"/"+resp.Response[0].GUID+"/query", alertRequest)
	suite.Equal(http.StatusOK, w.Code)
	alerts, err := decodeResponse[armotypes.V2ListResponseGeneric[[]types.RuntimeAlert]](w)
	if err != nil {
		suite.FailNow(err.Error())
	}
	suite.Len(alerts.Response, 1)
	suite.Equal(alerts.Total.Value, 3)
	suite.Equal(runtimeIncidents[2].RelatedAlerts[0], alerts.Response[0].RuntimeAlert)
	// get next page
	alertRequest.PageNum = ptr.Int(2)
	w = suite.doRequest(http.MethodPost, consts.RuntimeAlertPath+"/"+resp.Response[0].GUID+"/query", alertRequest)
	suite.Equal(http.StatusOK, w.Code)
	alerts, err = decodeResponse[armotypes.V2ListResponseGeneric[[]types.RuntimeAlert]](w)
	if err != nil {
		suite.FailNow(err.Error())
	}
	suite.Len(alerts.Response, 1)
	suite.Equal(3, alerts.Total.Value)
	suite.Equal(runtimeIncidents[2].RelatedAlerts[1], alerts.Response[0].RuntimeAlert)
	// filter alerts by message
	alertRequest = armotypes.V2ListRequest{
		PageSize: ptr.Int(1),
		PageNum:  ptr.Int(0),
		InnerFilters: []map[string]string{
			{
				"message": "msg3",
			},
		},
	}
	w = suite.doRequest(http.MethodPost, consts.RuntimeAlertPath+"/"+resp.Response[0].GUID+"/query", alertRequest)
	suite.Equal(http.StatusOK, w.Code)
	alerts, err = decodeResponse[armotypes.V2ListResponseGeneric[[]types.RuntimeAlert]](w)
	if err != nil {
		suite.FailNow(err.Error())
	}
	suite.Len(alerts.Response, 1)
	suite.Equal(1, alerts.Total.Value)
	suite.Equal(runtimeIncidents[2].RelatedAlerts[2], alerts.Response[0].RuntimeAlert)
	// test alerts sort by timestamp (with nanoseconds)
	alertRequest = armotypes.V2ListRequest{
		PageSize: ptr.Int(50),
		PageNum:  ptr.Int(0),
		OrderBy:  "timestamp:asc",
	}
	w = suite.doRequest(http.MethodPost, consts.RuntimeAlertPath+"/"+resp.Response[0].GUID+"/query", alertRequest)
	suite.Equal(http.StatusOK, w.Code)
	alerts, err = decodeResponse[armotypes.V2ListResponseGeneric[[]types.RuntimeAlert]](w)
	if err != nil {
		suite.FailNow(err.Error())
	}
	suite.Len(alerts.Response, 3)
	suite.Equal(alerts.Total.Value, 3)
	suite.Equal(runtimeIncidents[2].RelatedAlerts[2], alerts.Response[0].RuntimeAlert)
	suite.Equal(runtimeIncidents[2].RelatedAlerts[1], alerts.Response[1].RuntimeAlert)
	suite.Equal(runtimeIncidents[2].RelatedAlerts[0], alerts.Response[2].RuntimeAlert)

}

func (suite *MainTestSuite) TestRuntimeIncidentPolicies() {
	defaultPolicies := kdr.GetDefaultPolicies()
	defaultPoliciesPtr := []*types.IncidentPolicy{}
	for _, policy := range defaultPolicies {
		defaultPoliciesPtr = append(defaultPoliciesPtr, &types.IncidentPolicy{
			IncidentPolicy: policy,
		})
	}
	for _, policy := range defaultPolicies { // double the policies for testing
		defaultPoliciesPtr = append(defaultPoliciesPtr, &types.IncidentPolicy{
			IncidentPolicy: policy,
		})
	}
	modifyDocFunc := func(doc *types.IncidentPolicy) *types.IncidentPolicy {
		docCloned := Clone(doc)
		return docCloned
	}
	testOpts := testOptions[*types.IncidentPolicy]{
		mandatoryName: true,
		renameAllowed: true,
		uniqueName:    false,
	}
	ignore := cmp.FilterPath(func(p cmp.Path) bool {
		return p.String() == "IncidentPolicy.PortalBase.GUID" || p.String() == "GUID" || p.String() == "CreationTime" ||
			p.String() == "CreationDate" || p.String() == "IncidentPolicy.PortalBase.UpdatedTime" || p.String() == "UpdatedTime"
	}, cmp.Ignore())
	commonTestWithOptions(suite, consts.RuntimeIncidentPolicyPath, defaultPoliciesPtr, modifyDocFunc, testOpts, ignore, ignoreTime)

	// test request example from event ingester
	for _, policy := range defaultPolicies { // triple the policies for testing
		policy.Scope.RiskFactors = []armotypes.RiskFactor{"risk1", "risk2"}
		policy.Name += "-riskFactors"
		defaultPoliciesPtr = append(defaultPoliciesPtr, &types.IncidentPolicy{
			IncidentPolicy: policy,
		})
	}
	testBulkPostDocs(suite, consts.RuntimeIncidentPolicyPath, defaultPoliciesPtr, ignore, ignoreTime)
	time.Sleep(3 * time.Second)
	w := suite.doRequest(http.MethodPost, consts.RuntimeIncidentPolicyPath+"/query", runtimeIncidentPolicyReq1)
	suite.Equal(http.StatusOK, w.Code)
	newDoc, err := decodeResponse[armotypes.V2ListResponseGeneric[[]*kdr.IncidentPolicy]](w)
	if err != nil {
		suite.FailNow(err.Error())
	}
	suite.Len(newDoc.Response, newDoc.Total.Value)
	suite.Len(newDoc.Response, 2)
	for _, doc := range newDoc.Response {
		suite.NotNil(doc.GUID)
		suite.Equal("Anomaly", doc.Name)
	}
	w = suite.doRequest(http.MethodPost, consts.RuntimeIncidentPolicyPath+"/query", runtimeIncidentPolicyReq2)
	suite.Equal(http.StatusOK, w.Code)
	newDoc, err = decodeResponse[armotypes.V2ListResponseGeneric[[]*kdr.IncidentPolicy]](w)
	if err != nil {
		suite.FailNow(err.Error())
	}
	suite.Len(newDoc.Response, newDoc.Total.Value)
	suite.Len(newDoc.Response, 3)
	for _, doc := range newDoc.Response {
		suite.NotNil(doc.GUID)
		suite.Equal("Anomaly", doc.Name[:7])
	}
	// test sort by scope
	w = suite.doRequest(http.MethodPost, consts.RuntimeIncidentPolicyPath+"/query", runtimeIncidentPolicyReq3)
	suite.Equal(http.StatusOK, w.Code)
	_, err = decodeResponse[armotypes.V2ListResponseGeneric[[]*kdr.IncidentPolicy]](w)
	if err != nil {
		suite.FailNow(err.Error())
	}
	// test update scope designators
	docWithScope := newDoc.Response[0]
	docWithScope.Scope.Designators = []kdr.PolicyDesignators{
		{
			Cluster:   ptr.String("cluster1"),
			Kind:      ptr.String("kind1"),
			Name:      ptr.String("name1"),
			Namespace: ptr.String("namespace1"),
		},
	}
	w = suite.doRequest(http.MethodPut, consts.RuntimeIncidentPolicyPath+"/"+docWithScope.GUID, docWithScope)
	suite.Equal(http.StatusOK, w.Code)
	upDoc, err := decodeResponse[[]types.IncidentPolicy](w)
	if err != nil {
		suite.FailNow(err.Error())
	}
	suite.Equal(docWithScope.Scope.Designators, upDoc[1].Scope.Designators)
	// empty designators
	docWithScope.Scope.Designators = []kdr.PolicyDesignators{}
	w = suite.doRequest(http.MethodPut, consts.RuntimeIncidentPolicyPath+"/"+docWithScope.GUID, docWithScope)
	suite.Equal(http.StatusOK, w.Code)
	upDoc, err = decodeResponse[[]types.IncidentPolicy](w)
	if err != nil {
		suite.FailNow(err.Error())
	}
	suite.Equal(len(docWithScope.Scope.Designators), len(upDoc[1].Scope.Designators))
}

func (suite *MainTestSuite) TestIntegrationReference() {
	getTestCase := func() []*types.IntegrationReference {
		return []*types.IntegrationReference{
			{
				PortalBase: armotypes.PortalBase{
					Name: "incident0",
					GUID: "1c0e9d28-7e71-4370-999e-9b3e8f69a648",
				},
				Provider:     "jira",
				Type:         "ticket:cve",
				ProviderData: map[string]string{"key": "value"},
				Owner: &notifications.EntityIdentifiers{
					ResourceHash: "hash1",
					Cluster:      "cluster1",
					Namespace:    "namespace1",
					Kind:         "kind1",
					Name:         "name1",
				},
				RelatedObjects: []notifications.EntityIdentifiers{
					{
						CVEID:            "cve1",
						Severity:         "high",
						Component:        "component1",
						ComponentVersion: "version1",
					},
					{
						CVEID:            "cve2",
						Severity:         "critical",
						Component:        "component1",
						ComponentVersion: "version2",
					},
				},
			},
			{
				PortalBase: armotypes.PortalBase{
					Name: "incident1",
					GUID: "a-7e71-4370-999e-9b3e8f69a648",
				},
				Provider:     "jira",
				Type:         "ticket:cve:layer",
				ProviderData: map[string]string{"key": "value"},
				Owner: &notifications.EntityIdentifiers{
					ResourceHash: "hash3",
					Cluster:      "cluster1",
					Namespace:    "namespace1",
					Kind:         "kind1",
					Name:         "name1",
				},
				RelatedObjects: []notifications.EntityIdentifiers{
					{
						CVEID:            "cve1",
						Severity:         "high",
						Component:        "component1",
						ComponentVersion: "version1",
						LayerHash:        "layer1",
					},
					{
						CVEID:            "cve2",
						Severity:         "critical",
						Component:        "component1",
						ComponentVersion: "version1",
						LayerHash:        "layer",
					},
				},
			},
			{
				PortalBase: armotypes.PortalBase{
					Name: "incident2",
					GUID: "b-7e71-4370-999e-9b3e8f69a648",
				},
				Provider:     "jira",
				Type:         "ticket:cve",
				ProviderData: map[string]string{"key": "value"},
				Owner: &notifications.EntityIdentifiers{
					ResourceHash: "hash2",
					Cluster:      "cluster2",
					Namespace:    "namespace2",
					Kind:         "kind2",
					Name:         "name2",
				},
				RelatedObjects: []notifications.EntityIdentifiers{
					{
						CVEID:            "cve1",
						Severity:         "high",
						Component:        "component1",
						ComponentVersion: "version1",
					},
					{
						CVEID:            "cve2",
						Severity:         "critical",
						Component:        "component1",
						ComponentVersion: "version2",
					},
				},
			},
			{
				PortalBase: armotypes.PortalBase{
					Name: "incident3",
					GUID: "z-7e71-4370-999e-9b3e8f69a648",
				},
				Provider:     "jira",
				Type:         "ticket:cve",
				ProviderData: map[string]string{"key": "value"},
				RelatedObjects: []notifications.EntityIdentifiers{
					{
						CVEID:            "cve3",
						Severity:         "high",
						Component:        "component2",
						ComponentVersion: "version1",
					},
					{
						CVEID:            "cve2",
						Severity:         "critical",
						Component:        "component2",
						ComponentVersion: "version1",
						LayerHash:        "layer1",
					},
				},
			},
			{
				PortalBase: armotypes.PortalBase{
					Name: "incident4",
					GUID: "1c0e9d28-7e71-4370-999e-9b3e8f69a648",
				},
				Provider:     "jira",
				Type:         "ticket:cve",
				ProviderData: map[string]string{"key": "value"},
				Owner: &notifications.EntityIdentifiers{
					ResourceHash: "hash5",
					Cluster:      "cluster1",
					Namespace:    "namespace1",
					Kind:         "kind1",
					Name:         "name1",
				},
				RelatedObjects: []notifications.EntityIdentifiers{
					{
						CVEID:            "cve1",
						Severity:         "critical",
						Component:        "component2",
						ComponentVersion: "version2",
					},
					{
						CVEID:            "cve2",
						Severity:         "critical",
						Component:        "component1",
						ComponentVersion: "version2",
					},
					{
						CVEID: "123456",
					},
				},
			},
		}
	}

	modifyDocFunc := func(doc *types.IntegrationReference) *types.IntegrationReference {
		docCloned := Clone(doc)
		if docCloned.Attributes == nil {
			docCloned.Attributes = map[string]interface{}{}
		}
		docCloned.Attributes[rndStr.NewLen(5)] = rndStr.NewLen(5)
		return docCloned
	}

	testOpts := testOptions[*types.IntegrationReference]{
		mandatoryName: false,
		customGUID:    false,
		skipPutTests:  false,
	}
	commonTestWithOptions(suite, consts.IntegrationReferencePath, getTestCase(), modifyDocFunc,
		testOpts, commonCmpFilter, ignoreTime)

	searchTestCases := []searchTest{
		{
			testName:        "element match search with a number as string",
			expectedIndexes: []int{4},
			listRequest: armotypes.V2ListRequest{
				InnerFilters: []map[string]string{
					{
						"relatedObjects.cveID|elemMatch": "123456",
					},
				},
			},
		},
		{
			testName:        "search by component and version",
			expectedIndexes: []int{0, 1, 2},
			listRequest: armotypes.V2ListRequest{
				InnerFilters: []map[string]string{
					{
						"relatedObjects.component":        "component1",
						"relatedObjects.componentVersion": "version1",
					},
				},
			},
		},
		{
			testName:        "search by component and version not exist",
			expectedIndexes: []int{},
			listRequest: armotypes.V2ListRequest{
				InnerFilters: []map[string]string{
					{
						"relatedObjects.component":        "component1",
						"relatedObjects.componentVersion": "version66",
					},
				},
			},
		},
		{
			testName:        "search by missing owner",
			expectedIndexes: []int{3},
			listRequest: armotypes.V2ListRequest{
				InnerFilters: []map[string]string{
					{
						"owner": "|missing",
					},
				},
			},
		},
	}

	testPostV2ListRequest(suite, consts.IntegrationReferencePath, getTestCase(), nil, searchTestCases, commonCmpFilter, ignoreTime)

	uniqueValueTestCases := []uniqueValueTest{
		{
			testName: "unique values with elem match operator",
			uniqueValuesRequest: armotypes.UniqueValuesRequestV2{
				Fields: map[string]string{
					"relatedObjects.severity|relatedObjects.component": "",
				},
				InnerFilters: []map[string]string{{
					"relatedObjects.component|elemMatch": "component1,component2",
					"relatedObjects.severity|elemMatch":  "critical,high",
				},
					{
						"relatedObjects.component|elemMatch": "component1,component2",
						"relatedObjects.severity|elemMatch":  "critical,high",
					}},
			},
			expectedResponse: armotypes.UniqueValuesResponseV2{
				Fields: map[string][]string{
					"relatedObjects.severity|relatedObjects.component": {
						"critical|component1", "critical|component2", "high|component1", "high|component2"},
				},
				FieldsCount: nil,
			},
		},
		{
			testName: "unique values with elem match operator",
			uniqueValuesRequest: armotypes.UniqueValuesRequestV2{
				Fields: map[string]string{
					"relatedObjects.severity": "",
				},
				InnerFilters: []map[string]string{
					{
						"relatedObjects.component|elemMatch":        "component1,component2",
						"relatedObjects.componentVersion|elemMatch": "version1,version2",
						"owner": "|exists",
					},
				},
			},
			expectedResponse: armotypes.UniqueValuesResponseV2{
				Fields: map[string][]string{
					"relatedObjects.severity": {"critical", "high"},
				},
				FieldsCount: nil,
			},
		},
		{
			testName: "resources with image layer",
			uniqueValuesRequest: armotypes.UniqueValuesRequestV2{
				Fields: map[string]string{
					"owner.resourceHash": "",
				},
				InnerFilters: []map[string]string{
					{
						"relatedObjects.layerHash": "|exists",
						"owner":                    "|exists",
					},
				},
			},
			expectedResponse: armotypes.UniqueValuesResponseV2{
				Fields: map[string][]string{
					"owner.resourceHash": {"hash3"},
				},
				FieldsCount: nil,
			},
		},
		{
			testName: "search releated objects properties without element match",
			uniqueValuesRequest: armotypes.UniqueValuesRequestV2{
				Fields: map[string]string{
					"owner.resourceHash": "",
				},
				InnerFilters: []map[string]string{
					{
						"relatedObjects.cveID":    "cve1",
						"relatedObjects.severity": "critical",
					},
				},
			},
			expectedResponse: armotypes.UniqueValuesResponseV2{
				Fields: map[string][]string{
					"owner.resourceHash": {"hash1", "hash2", "hash3", "hash5"},
				},
				FieldsCount: nil,
			},
		},
		{
			testName: "search releated objects properties with element match",
			uniqueValuesRequest: armotypes.UniqueValuesRequestV2{
				Fields: map[string]string{
					"owner.resourceHash": "",
				},
				InnerFilters: []map[string]string{
					{
						"owner":                             "|exists",
						"relatedObjects.cveID|elemMatch":    "cve1",
						"relatedObjects.severity|elemMatch": "critical",
					},
				},
			},
			expectedResponse: armotypes.UniqueValuesResponseV2{
				Fields: map[string][]string{
					"owner.resourceHash": {"hash5"},
				},
				FieldsCount: nil,
			},
		},
		{
			testName: "math element in releated objects with missing layer",
			uniqueValuesRequest: armotypes.UniqueValuesRequestV2{
				Fields: map[string]string{
					"owner.resourceHash": "",
				},
				InnerFilters: []map[string]string{
					{
						"relatedObjects.cveID|elemMatch":            "cve1",
						"relatedObjects.severity|elemMatch":         "critical",
						"relatedObjects.component|elemMatch":        "component2",
						"relatedObjects.componentVersion|elemMatch": "version2",
						"relatedObjects.componentLayer|elemMatch":   "|missing",
					},
				},
			},
			expectedResponse: armotypes.UniqueValuesResponseV2{
				Fields: map[string][]string{
					"owner.resourceHash": {"hash5"},
				},
				FieldsCount: nil,
			},
		},
	}

	testUniqueValues(suite, consts.IntegrationReferencePath, getTestCase(), uniqueValueTestCases, commonCmpFilter, ignoreTime)
}

var accountCompareFilter = cmp.FilterPath(func(p cmp.Path) bool {
	switch p.String() {
	//all the fields that are not supposed to be compared because they cannot be empty.
	case "PortalBase.GUID", "PortalBase.UpdatedTime", "PortalBase.Name", "CreationTime":
		zap.L().Info("path", zap.String("path", p.String()))

		return true
	}
	return false
}, cmp.Ignore())

var updateAccountCompareFilter = cmp.FilterPath(func(p cmp.Path) bool {
	switch p.String() {
	//all the fields that are not supposed to be compared because they are cannot be empty.
	case "AccountID", "Provider":
		zap.L().Info("path", zap.String("path", p.String()))

		return true
	}
	return false
}, cmp.Ignore())

func (suite *MainTestSuite) TestCloudAccount() {
	accounts, _ := loadJson[*types.CloudAccount](cloudAccountsJson)

	modifyFunc := func(account *types.CloudAccount) *types.CloudAccount {
		account.UpdatedTime = time.Now().UTC().Format(time.RFC3339)
		return account
	}

	commonTest(suite, consts.CloudAccountPath, accounts, modifyFunc, accountCompareFilter)

	projectedDocs := []*types.CloudAccount{
		{
			PortalBase: armotypes.PortalBase{
				Name: "AWS-Test-Account",
			},
		},
		{
			PortalBase: armotypes.PortalBase{
				Name: "AWS-Test-Account-No-Regions",
			},
		},
		{
			PortalBase: armotypes.PortalBase{
				Name: "Azure-Test-Account",
			},
		},
		{
			PortalBase: armotypes.PortalBase{
				Name: "GCP-Test-Account",
			},
		},
	}

	searchQueries := []searchTest{
		{
			testName:        "get all",
			expectedIndexes: []int{0, 1, 2, 3},
			listRequest:     armotypes.V2ListRequest{},
		},
		{
			testName:        "get first page",
			expectedIndexes: []int{0, 1},
			listRequest: armotypes.V2ListRequest{
				PageSize: ptr.Int(2),
				PageNum:  ptr.Int(0),
			},
		},
		{
			testName:        "get multiple names",
			expectedIndexes: []int{0, 2, 3},
			listRequest: armotypes.V2ListRequest{
				OrderBy: "name:asc",
				InnerFilters: []map[string]string{
					{
						"name": "AWS-Test-Account,Azure-Test-Account,GCP-Test-Account",
					},
				},
			},
		},
		{
			testName:        "field or match",
			expectedIndexes: []int{2},
			listRequest: armotypes.V2ListRequest{
				OrderBy: "name:asc",
				InnerFilters: []map[string]string{
					{
						"name": "Azure-Test-Account",
					},
				},
			},
		},
		{
			testName:        "fields and match",
			expectedIndexes: []int{0},
			listRequest: armotypes.V2ListRequest{
				OrderBy: "name:asc",
				InnerFilters: []map[string]string{
					{
						"provider": "aws",
						"name":     "AWS-Test-Account",
					},
				},
			},
		},
		{
			testName:        "filters exist operator",
			expectedIndexes: []int{0, 1, 2, 3},
			listRequest: armotypes.V2ListRequest{
				OrderBy: "name:asc",
				InnerFilters: []map[string]string{
					{
						"provider": "|exists",
					},
				},
			},
		},
		{
			testName:        "like ignorecase match",
			expectedIndexes: []int{0, 1},
			listRequest: armotypes.V2ListRequest{
				OrderBy: "name:asc",
				InnerFilters: []map[string]string{
					{
						"name": "AWS-Test|like&ignorecase",
					},
				},
			},
		},
		{
			testName:        "like with multi results",
			expectedIndexes: []int{0, 1},
			listRequest: armotypes.V2ListRequest{
				OrderBy: "name:asc",
				InnerFilters: []map[string]string{
					{
						"credentials.awsCredentials.encryptedRoleARN": "arn:aws|like",
					},
				},
			},
		},
		{
			testName:         "projection test",
			expectedIndexes:  []int{0, 1, 2, 3},
			projectedResults: true,
			listRequest: armotypes.V2ListRequest{
				OrderBy:    "name:asc",
				FieldsList: []string{"name"},
			},
		},
		{
			testName:        "credentials.azureCredentials.encryptedTenantID exists",
			expectedIndexes: []int{2},
			listRequest: armotypes.V2ListRequest{
				OrderBy: "name:asc",
				InnerFilters: []map[string]string{
					{
						"credentials.azureCredentials.encryptedTenantID": "|exists",
					},
				},
			},
		},
		{
			testName:        "credentials.encryptedPrincipalID exists",
			expectedIndexes: []int{3},
			listRequest: armotypes.V2ListRequest{
				OrderBy: "name:asc",
				InnerFilters: []map[string]string{
					{
						"credentials.gcpCredentials.encryptedPrincipalID": "|exists",
					},
				},
			},
		},
	}

	zap.L().Info("search test", zap.Any("searchQueries", searchQueries), zap.Any("accounts", projectedDocs))
	testPostV2ListRequest(suite, consts.CloudAccountPath, accounts, projectedDocs, searchQueries, accountCompareFilter, ignoreTime)

	uniqueValues := []uniqueValueTest{
		{
			testName: "unique providers",
			uniqueValuesRequest: armotypes.UniqueValuesRequestV2{
				Fields: map[string]string{
					"provider": "",
				},
			},
			expectedResponse: armotypes.UniqueValuesResponseV2{
				Fields: map[string][]string{
					"provider": {"aws", "azure", "gcp"},
				},
				FieldsCount: map[string][]armotypes.UniqueValuesResponseFieldsCount{
					"provider": {
						{
							Field: "aws",
							Count: 2,
						},
						{
							Field: "azure",
							Count: 1,
						},
						{
							Field: "gcp",
							Count: 1,
						},
					},
				},
			},
		},
		{
			testName: "unique names with aws filter",
			uniqueValuesRequest: armotypes.UniqueValuesRequestV2{
				Fields: map[string]string{
					"name": "",
				},
				InnerFilters: []map[string]string{
					{
						"provider": "aws",
					},
				},
			},
			expectedResponse: armotypes.UniqueValuesResponseV2{
				Fields: map[string][]string{
					"name": {"AWS-Test-Account", "AWS-Test-Account-No-Regions"},
				},
				FieldsCount: map[string][]armotypes.UniqueValuesResponseFieldsCount{
					"name": {
						{
							Field: "AWS-Test-Account",
							Count: 1,
						},
						{
							Field: "AWS-Test-Account-No-Regions",
							Count: 1,
						},
					},
				},
			},
		},
	}
	zap.L().Info("unique values test", zap.Any("uniqueValues", uniqueValues))

	testUniqueValues(suite, consts.CloudAccountPath, accounts, uniqueValues, accountCompareFilter, ignoreTime)

	testPartialUpdate(suite, consts.CloudAccountPath, &types.CloudAccount{}, accountCompareFilter, updateAccountCompareFilter, ignoreTime)

	testGetByName(suite, consts.CloudAccountPath, "name", accounts, accountCompareFilter, ignoreTime)
}

func (suite *MainTestSuite) TestWorkflows() {
	// load workflows from json
	workflowsObj, _ := loadJson[*types.Workflow](workflowsJson)

	modifyDocFunc := func(doc *types.Workflow) *types.Workflow {
		docCloned := Clone(doc)
		return docCloned
	}
	testOpts := testOptions[*types.Workflow]{
		mandatoryName: true,
		renameAllowed: true,
		uniqueName:    false,
	}
	ignore := cmp.FilterPath(func(p cmp.Path) bool {
		return p.String() == "Workflow.PortalBase.GUID" || p.String() == "GUID" || p.String() == "CreationTime" ||
			p.String() == "CreationDate" || p.String() == "Workflow.PortalBase.UpdatedTime" || p.String() == "UpdatedTime"
	}, cmp.Ignore())
	commonTestWithOptions(suite, consts.WorkflowPath, workflowsObj, modifyDocFunc, testOpts, ignore, ignoreTime)

	// test sort and pagination
	testBulkPostDocs(suite, consts.WorkflowPath, workflowsObj, ignore, ignoreTime)
	time.Sleep(3 * time.Second)
	w := suite.doRequest(http.MethodPost, consts.WorkflowPath+"/query", workflowsSortReq)
	suite.Equal(http.StatusOK, w.Code)
	res, err := decodeResponse[armotypes.V2ListResponseGeneric[[]*workflows.Workflow]](w)
	if err != nil {
		suite.FailNow(err.Error())
	}

	suite.Len(res.Response, 2)

}
