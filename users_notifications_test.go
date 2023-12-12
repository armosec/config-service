package main

import (
	"config-service/types"
	"config-service/utils/consts"
	_ "embed"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/armosec/armoapi-go/armotypes"
	"github.com/aws/smithy-go/ptr"
	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"
)

//go:embed test_data/vulnerabilities_drifts.json
var vulnerabilitiesDriftsBytes []byte

var driftCompareFilter = cmp.FilterPath(func(p cmp.Path) bool {
	return p.String() == "AggregatedVulnerability.GUID" || p.String() == "AggregatedVulnerability.UpdatedTime" || p.String() == "AggregatedVulnerability.CreationTime"
}, cmp.Ignore())

func (suite *MainTestSuite) TestUsersNotificationsVulnerabilities() {
	docs, _ := loadJson[*types.AggregatedVulnerability](vulnerabilitiesDriftsBytes)
	modifyFunc := func(doc *types.AggregatedVulnerability) *types.AggregatedVulnerability {
		newImage := "image-" + uuid.New().String()
		doc.Images = append(doc.Images, newImage)
		return doc
	}

	testOpts := testOptions[*types.AggregatedVulnerability]{
		uniqueName:    false,
		mandatoryName: false,
	}

	commonTestWithOptions(suite, consts.UsersNotificationsVulnerabilitiesPath, docs, modifyFunc, testOpts, ignoreTime, driftCompareFilter)

	uniqueValues := []uniqueValueTest{
		{
			testName: "unique images",
			uniqueValuesRequest: armotypes.UniqueValuesRequestV2{
				Fields: map[string]string{
					"images": "",
				},
			},
			expectedResponse: armotypes.UniqueValuesResponseV2{
				Fields: map[string][]string{
					"images": {"image1", "image2", "image3", "image4", "image5", "image6"},
				},
				FieldsCount: map[string][]armotypes.UniqueValuesResponseFieldsCount{
					"images": {
						{

							Field: "image1",
							Count: 3,
						},
						{
							Field: "image2",
							Count: 3,
						},
						{
							Field: "image3",
							Count: 1,
						},
						{
							Field: "image4",
							Count: 1,
						},
						{
							Field: "image5",
							Count: 1,
						},
						{
							Field: "image6",
							Count: 1,
						},
					},
				},
			},
		},
		{
			testName: "unique images paginated",
			uniqueValuesRequest: armotypes.UniqueValuesRequestV2{
				PageSize: 3,
				PageNum:  ptr.Int(2),
				Fields: map[string]string{
					"images": "",
				},
			},
			expectedResponse: armotypes.UniqueValuesResponseV2{
				Fields: map[string][]string{
					"images": {"image4", "image5", "image6"},
				},
				FieldsCount: map[string][]armotypes.UniqueValuesResponseFieldsCount{
					"images": {
						{
							Field: "image4",
							Count: 1,
						},
						{
							Field: "image5",
							Count: 1,
						},
						{
							Field: "image6",
							Count: 1,
						},
					},
				},
			},
		},
	}

	testUniqueValues(suite, consts.UsersNotificationsVulnerabilitiesPath, docs, uniqueValues, ignoreTime, driftCompareFilter)

	searchQueries := []searchTest{
		//field or match
		{
			testName:        "cluster w images or fix version",
			expectedIndexes: []int{0, 1, 3},
			listRequest: armotypes.V2ListRequest{
				OrderBy: "name:asc",
				InnerFilters: []map[string]string{
					{
						"cluster": "cluster1",
						"images":  "image2,image3",
					},
					{
						"cluster":    "cluster2",
						"fixVersion": "v2",
					},
				},
			},
		},
		{
			testName:        "severity ",
			expectedIndexes: []int{2, 3, 4},
			listRequest: armotypes.V2ListRequest{
				OrderBy: "name:asc",
				InnerFilters: []map[string]string{
					{
						"severity": "3|greater",
					},
				},
			},
		},
	}

	testPostV2ListRequest(suite, consts.UsersNotificationsVulnerabilitiesPath, docs, nil, searchQueries, ignoreTime, driftCompareFilter)

}

func (suite *MainTestSuite) TestUsersNotificationsCache() {
	toJson := func(i interface{}) json.RawMessage {
		b, err := json.Marshal(i)
		if err != nil {
			suite.FailNow(err.Error())
		}
		return json.RawMessage(b)
	}
	fromJson := func(data json.RawMessage) interface{} {
		var i interface{}
		err := json.Unmarshal(data, &i)
		if err != nil {
			suite.FailNow(err.Error())
		}
		return i
	}

	now := time.Now().UTC()

	docs := []*types.Cache{
		{
			GUID:       "test-guid-1",
			Name:       "test-name-1",
			Data:       toJson(float64(1)),
			DataType:   "test-data-type-1",
			ExpiryTime: now.Add(1 * time.Hour),
		},
		{
			GUID:       "test-guid-2",
			Name:       "test-name-2",
			Data:       toJson("data-value-string"),
			DataType:   "test-data-type-2",
			ExpiryTime: now.Add(2 * time.Hour),
		},
		{
			GUID:       "test-guid-3",
			Name:       "test-name-3*.?7*",
			Data:       toJson([]interface{}{"test-*value*?-3", "test-value-4"}),
			ExpiryTime: now.Add(3 * time.Hour),
		},
		{
			GUID:       "test-guid-4",
			Name:       "test-name-4",
			Data:       toJson(float64(5)),
			DataType:   "test-data-type-4",
			ExpiryTime: now.Add(4 * time.Hour),
		},
		{
			GUID:       "test-guid-5",
			Name:       "test-name-5",
			Data:       toJson("role,bind,clusterrole"),
			DataType:   "test-data-type-5",
			ExpiryTime: now.Add(5 * time.Hour),
		},
	}

	modifyFunc := func(doc *types.Cache) *types.Cache {
		i := fromJson(doc.Data)
		switch i.(type) {
		case float64:
			doc.Data = toJson(i.(float64) + 1)
		case string:
			doc.Data = toJson(i.(string) + "-updated")
		case []interface{}:
			doc.Data = toJson(append(i.([]interface{}), "test-value-5"))
		}
		return doc
	}
	testOpts := testOptions[*types.Cache]{
		uniqueName:    false,
		mandatoryName: false,
		customGUID:    true,
	}

	commonTestWithOptions(suite, consts.UsersNotificationsCachePath, docs, modifyFunc, testOpts, commonCmpFilter, ignoreTime)

	projectedDocs := []*types.Cache{
		{
			Name: "test-name-1",
		},
		{

			Name: "test-name-2",
		},
		{
			GUID: "test-guid-3",
			Data: toJson([]interface{}{"test-value-3", "test-value-4"}),
		},
	}

	//Until we will support schema info queries can only match string, boolean, and numbers
	//Types like slices, time.Time, and json.RawMessage fields are not supported yet
	searchQueries := []searchTest{
		//field or match
		{
			testName:        "field or match",
			expectedIndexes: []int{0, 1},
			listRequest: armotypes.V2ListRequest{
				OrderBy: "name:asc",
				InnerFilters: []map[string]string{
					{
						"dataType": "test-data-type-1,test-data-type-2",
					},
				},
			},
		},
		//same field or match in descending order
		{
			testName:        "same field or match in descending order",
			expectedIndexes: []int{1, 0},
			listRequest: armotypes.V2ListRequest{
				OrderBy: "name:desc",
				InnerFilters: []map[string]string{
					{
						"dataType":  "test-data-type-1,test-data-type-2",
						"someField": "", //test ignore empty field
					},
				},
			},
		},
		//fields and match
		{
			testName:        "fields and match",
			expectedIndexes: []int{0},
			listRequest: armotypes.V2ListRequest{
				OrderBy: "name:asc",
				InnerFilters: []map[string]string{
					{
						"dataType": "test-data-type-1,test-data-type-2|equal",
						"name":     "test-name-1",
					},
				},
			},
		},
		//filters exist operator
		{
			testName:        "filters exist operator",
			expectedIndexes: []int{0, 1, 3, 4},
			listRequest: armotypes.V2ListRequest{
				OrderBy: "name:asc",
				InnerFilters: []map[string]string{
					{
						"dataType": "|exists",
					},
				},
			},
		},
		//filters or match with missing operator
		{
			testName:        "filters or match with missing operator",
			expectedIndexes: []int{0, 1, 2},
			listRequest: armotypes.V2ListRequest{
				OrderBy: "name:asc",
				InnerFilters: []map[string]string{
					{
						"dataType": "test-data-type-1|match,test-data-type-2",
					},
					{
						"dataType": "|missing",
					},
				},
			},
		},
		//greater than equal on one field
		{
			testName:        "greater than equal on one field",
			expectedIndexes: []int{1, 2, 3, 4},
			listRequest: armotypes.V2ListRequest{
				OrderBy: "name:asc",
				InnerFilters: []map[string]string{
					{
						"name": "test-name-2|greater",
					},
				},
			},
		},
		//like match
		{
			testName:        "like ignorecase match",
			expectedIndexes: []int{4},
			listRequest: armotypes.V2ListRequest{
				OrderBy: "name:asc",
				InnerFilters: []map[string]string{
					{
						"dataType": "TEST-data-Type-5|like&ignorecase",
					},
				},
			},
		},
		{
			testName:        "like with multi results",
			expectedIndexes: []int{0, 1, 3, 4},
			listRequest: armotypes.V2ListRequest{
				OrderBy: "name:asc",
				InnerFilters: []map[string]string{
					{
						"dataType": "test-data-type-|like",
					},
				},
			},
		},
		{
			testName:        "like with special chars",
			expectedIndexes: []int{2},
			listRequest: armotypes.V2ListRequest{
				OrderBy: "name:asc",
				InnerFilters: []map[string]string{
					{
						"name": "test-name-3*.?|like&ignorecase",
					},
				},
			},
		},
		//regex match
		{
			testName:        "regex ignorecase match",
			expectedIndexes: []int{4},
			listRequest: armotypes.V2ListRequest{
				OrderBy: "name:asc",
				InnerFilters: []map[string]string{
					{
						"dataType": ".*Dat.-type-5|regex&ignorecase",
					},
				},
			},
		},
		{
			testName:        "range  strings",
			expectedIndexes: []int{0, 1},
			listRequest: armotypes.V2ListRequest{
				OrderBy: "name:asc",
				InnerFilters: []map[string]string{
					{
						"name": "test-name&test-name-3|range",
					},
				},
			},
		},
		{
			testName:        "range ids",
			expectedIndexes: []int{0, 1, 2},
			listRequest: armotypes.V2ListRequest{
				OrderBy: "name:asc",
				InnerFilters: []map[string]string{
					{
						"guid": "test-guid-0&test-guid-3|range",
					},
				},
			},
		},
		{
			testName:        "range string dates",
			expectedIndexes: []int{0, 1, 2, 3, 4},
			listRequest: armotypes.V2ListRequest{
				OrderBy: "name:asc",
				InnerFilters: []map[string]string{
					{
						"creationTime": fmt.Sprintf("%s&%s|range", time.Now().UTC().Add(-1*time.Hour).Format(time.RFC3339), time.Now().UTC().Add(1*time.Hour).Format(time.RFC3339)),
					},
				},
			},
		},
		//lower than equal on one field
		{
			testName:        "lower than equal on one field",
			expectedIndexes: []int{0, 1},
			listRequest: armotypes.V2ListRequest{
				OrderBy: "name:asc",
				InnerFilters: []map[string]string{
					{
						"name": "test-name-2|lower",
					},
				},
			},
		},
		//greater than equal on multiple values
		{
			testName:        "greater than equal on multiple values",
			expectedIndexes: []int{0, 2, 3, 4},
			listRequest: armotypes.V2ListRequest{
				OrderBy: "name:asc",
				InnerFilters: []map[string]string{
					{
						"guid": "test-guid-1|lower,test-guid-4|greater,.*-gui.-3|regex",
					},
				},
			},
		},
		{
			testName:        "greater than equal expiry time",
			expectedIndexes: []int{3, 4},
			listRequest: armotypes.V2ListRequest{
				OrderBy: "name:asc",
				InnerFilters: []map[string]string{
					{
						"expiryTime": time.Now().UTC().Add(4*time.Hour).Format(time.RFC3339) + "|greater",
					},
				},
			},
		},
		//paginated tests
		{
			testName:        "paginated with query 1",
			expectedIndexes: []int{0, 1},
			listRequest: armotypes.V2ListRequest{
				PageSize: ptr.Int(2),
				OrderBy:  "name:asc",
				InnerFilters: []map[string]string{
					{
						"dataType": "test-data-type-1,test-data-type-2",
					},
					{
						"dataType": "|missing",
					},
				},
			},
		},
		{
			testName:        "paginated with query 2",
			expectedIndexes: []int{2},
			listRequest: armotypes.V2ListRequest{
				PageSize: ptr.Int(2),
				PageNum:  ptr.Int(2),
				OrderBy:  "name:asc",
				InnerFilters: []map[string]string{
					{
						"dataType": "test-data-type-1,test-data-type-2",
					},
					{
						"dataType": ",|missing",
					},
				},
			},
		},
		{
			testName:        "paginated with query 2",
			expectedIndexes: []int{2},
			listRequest: armotypes.V2ListRequest{
				PageSize: ptr.Int(2),
				PageNum:  ptr.Int(2),
				OrderBy:  "name:asc",
				InnerFilters: []map[string]string{
					{
						"dataType": "test-data-type-1,test-data-type-2",
					},
					{
						"dataType": ",|missing",
					},
				},
			},
		},
		{
			testName:        "paginated all 1",
			expectedIndexes: []int{0, 1},
			listRequest: armotypes.V2ListRequest{
				PageSize: ptr.Int(2),
				PageNum:  ptr.Int(1),
				OrderBy:  "name:asc",
			},
		},
		{
			testName:        "paginated all 2",
			expectedIndexes: []int{2, 3},
			listRequest: armotypes.V2ListRequest{
				PageSize: ptr.Int(2),
				PageNum:  ptr.Int(2),
				OrderBy:  "name:asc",
			},
		},
		{
			testName:        "paginated all 3",
			expectedIndexes: []int{4},
			listRequest: armotypes.V2ListRequest{
				PageSize: ptr.Int(2),
				PageNum:  ptr.Int(3),
				OrderBy:  "name:asc",
			},
		},
		//projection test
		{
			testName:         "projection test",
			expectedIndexes:  []int{0, 1},
			projectedResults: true,
			listRequest: armotypes.V2ListRequest{
				OrderBy:    "name:asc",
				FieldsList: []string{"name"},
				InnerFilters: []map[string]string{
					{
						"dataType": "test-data-type-1,test-data-type-2",
					},
				},
			},
		},
	}

	testPostV2ListRequest(suite, consts.UsersNotificationsCachePath, docs, projectedDocs, searchQueries, commonCmpFilter, ignoreTime)

	//add some docs for unique values tests
	moreDocs := []*types.Cache{
		{
			GUID:     "test-guid-6",
			Name:     "test-name-1",
			DataType: "test-data-type-1",
		},
		{
			GUID:     "test-guid-7",
			Name:     "test-name-2",
			DataType: "test-data-type-2",
		},
		{
			GUID:     "test-guid-8",
			Name:     "test-name-2",
			DataType: "test-data-type-2",
		},
	}
	docs = append(docs, moreDocs...)

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
					"name": {"test-name-1", "test-name-2", "test-name-3*.?7*", "test-name-4", "test-name-5"},
				},
				FieldsCount: map[string][]armotypes.UniqueValuesResponseFieldsCount{
					"name": {
						{

							Field: "test-name-1",
							Count: 2,
						},
						{
							Field: "test-name-2",
							Count: 3,
						},
						{
							Field: "test-name-3*.?7*",
							Count: 1,
						},
						{
							Field: "test-name-4",
							Count: 1,
						},
						{
							Field: "test-name-5",
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
						"dataType": "test-data-type-1,test-data-type-2",
					},
				},
			},
			expectedResponse: armotypes.UniqueValuesResponseV2{
				Fields: map[string][]string{
					"name": {"test-name-1", "test-name-2"},
				},
				FieldsCount: map[string][]armotypes.UniqueValuesResponseFieldsCount{
					"name": {
						{

							Field: "test-name-1",
							Count: 2,
						},
						{

							Field: "test-name-2",
							Count: 3,
						},
					},
				},
			},
		},
		{
			testName: "unique names and datatypes",
			uniqueValuesRequest: armotypes.UniqueValuesRequestV2{
				Fields: map[string]string{
					"name":     "",
					"dataType": "",
				},
				InnerFilters: []map[string]string{
					{
						"someField": "", //test ignore empty field
					},
				},
			},
			expectedResponse: armotypes.UniqueValuesResponseV2{
				Fields: map[string][]string{
					"name":     {"test-name-1", "test-name-2", "test-name-3*.?7*", "test-name-4", "test-name-5"},
					"dataType": {"test-data-type-1", "test-data-type-2", "test-data-type-4", "test-data-type-5"},
				},
				FieldsCount: map[string][]armotypes.UniqueValuesResponseFieldsCount{
					"name": {
						{
							Field: "test-name-1",
							Count: 2,
						},
						{
							Field: "test-name-2",
							Count: 3,
						},
						{
							Field: "test-name-3*.?7*",
							Count: 1,
						},
						{
							Field: "test-name-4",
							Count: 1,
						},
						{
							Field: "test-name-5",
							Count: 1,
						},
					},
					"dataType": {
						{
							Field: "test-data-type-1",
							Count: 2,
						},
						{
							Field: "test-data-type-2",
							Count: 3,
						},
						{
							Field: "test-data-type-4",
							Count: 1,
						},
						{
							Field: "test-data-type-5",
							Count: 1,
						},
					},
				},
			},
		},
	}

	testUniqueValues(suite, consts.UsersNotificationsCachePath, docs, uniqueValues, commonCmpFilter, ignoreTime)

	cachedDuration := armotypes.PortalCache[time.Duration]{
		GUID:     "test-duration-guid",
		Name:     "test-duration-name",
		Data:     time.Hour,
		DataType: "test-duration-data-type",
	}

	w := suite.doRequest(http.MethodPost, consts.UsersNotificationsCachePath, cachedDuration)
	suite.Equal(http.StatusCreated, w.Code)
	newDoc, err := decodeResponse[armotypes.PortalCache[time.Duration]](w)
	if err != nil {
		suite.FailNow(err.Error())
	}
	diff := cmp.Diff(cachedDuration, newDoc, commonCmpFilter, ignoreTime)
	suite.Equal("", diff)

	w = suite.doRequest(http.MethodGet, consts.UsersNotificationsCachePath+"/test-duration-guid", nil)
	suite.Equal(http.StatusOK, w.Code)
	newDoc, err = decodeResponse[armotypes.PortalCache[time.Duration]](w)
	if err != nil {
		suite.FailNow(err.Error())
	}
	diff = cmp.Diff(cachedDuration, newDoc, commonCmpFilter, ignoreTime)
	suite.Equal("", diff)

	ttlDoc := &types.Cache{
		GUID:     "test-ttl-guid",
		Name:     "test-ttl-name",
		Data:     toJson("test-ttl-data"),
		DataType: "test-ttl-data-type",
	}

	ttlDoc = testPostDoc(suite, consts.UsersNotificationsCachePath, ttlDoc, commonCmpFilter, ignoreTime)
	//check that default ttl is set to 90 days from now
	suite.Equal(time.Now().UTC().Add(time.Hour*24*90).Format(time.RFC3339), ttlDoc.ExpiryTime.Format(time.RFC3339), "default ttl is not set correctly")
	//set ttl to more than 90 days - should be ignored
	expirationTime := ttlDoc.ExpiryTime
	ttlUpdate := Clone(ttlDoc)
	ttlDoc.ExpiryTime = time.Now().UTC().Add(time.Hour * 24 * 100)
	ttlUpdate = testPutDoc(suite, consts.UsersNotificationsCachePath, ttlDoc, ttlUpdate, commonCmpFilter, ignoreTime)
	suite.Equal(expirationTime.UTC().Format(time.RFC3339), ttlUpdate.ExpiryTime.Format(time.RFC3339), "ttl above max allowed is not ignored")
	//set ttl to time in past
	ttlInPast := Clone(ttlDoc)
	ttlInPast.ExpiryTime = time.Now().UTC().Add(time.Hour * -24)
	expirationTime = ttlInPast.ExpiryTime
	ttlInPast = testPutDoc(suite, consts.UsersNotificationsCachePath, ttlDoc, ttlInPast, commonCmpFilter, ignoreTime)
	suite.Equal(expirationTime.UTC().Format(time.RFC3339), ttlInPast.ExpiryTime.Format(time.RFC3339), "ttl in past is not set")
	//wait for the document to expire and check that it is deleted - this can take up to one minute
	deleted := false
	for i := 0; i < 62; i++ {
		time.Sleep(time.Second)
		w := suite.doRequest(http.MethodGet, consts.UsersNotificationsCachePath+"/test-ttl-guid", nil)
		if w.Code == http.StatusNotFound {
			deleted = true
			break
		}
		suite.T().Logf("waiting for document to expire, for %d seconds", i)
	}
	suite.True(deleted, "document was not deleted after ttl expired")
}
