package main

import (
	"config-service/types"
	"encoding/json"
	"fmt"
	"slices"
	"time"

	"github.com/armosec/armoapi-go/armotypes"
	"github.com/aws/smithy-go/ptr"
	"github.com/go-faker/faker/v4"
	"github.com/go-faker/faker/v4/pkg/options"
	uuid "github.com/satori/go.uuid"

	"net/http"
	"net/http/httptest"
	"sort"
	"strings"

	"github.com/google/go-cmp/cmp"
)

// //////////////////////////////////////// Test scenarios //////////////////////////////////////////
type testOptions[T any] struct {
	uniqueName             bool
	mandatoryName          bool
	renameAllowed          bool
	customGUID             bool
	skipPutTests           bool
	clondeDocFunc          *func(T) T
	allGUIDsAsInnerFilters bool
}

func commonTestWithOptions[T types.DocContent](suite *MainTestSuite, path string, testDocs []T, modifyFunc func(T) T, testOptions testOptions[T], compareNewOpts ...cmp.Option) {
	if len(testDocs) < 3 {
		suite.FailNow("commonTest: need at least 3 documents")
	}
	doc1 := testDocs[0]
	doc2 := testDocs[1]
	documents := testDocs[1:]

	var cloneDocFun func(T) T
	if testOptions.clondeDocFunc != nil {
		cloneDocFun = *testOptions.clondeDocFunc
	} else {
		cloneDocFun = Clone[T]
	}

	//POST
	//create doc
	doc1.SetGUID("my custom guid")
	doc1Guid := doc1.GetGUID()
	createTime, _ := time.Parse(time.RFC3339, time.Now().UTC().Format(time.RFC3339))
	doc1 = testPostDoc(suite, path, doc1, compareNewOpts...)
	if testOptions.customGUID {
		suite.Equal(doc1Guid, doc1.GetGUID(), "GUID should be the same")
	} else {
		suite.NotEqual(doc1Guid, doc1.GetGUID(), "GUID should be generated")
		_, err := uuid.FromString(doc1.GetGUID())
		suite.NoError(err, "GUID should be a valid uuid")
	}
	//check creation time
	suite.NotNil(doc1.GetCreationTime(), "creation time should not be nil")
	suite.True(createTime.Before(*doc1.GetCreationTime()) || createTime.Equal(*doc1.GetCreationTime()), "creation time is not recent")

	sameNameDoc := cloneDocFun(doc1)
	sameNameDoc.SetGUID("")

	if testOptions.uniqueName {
		//post doc with same name should fail
		testBadRequest(suite, http.MethodPost, path, errorNameExist(sameNameDoc.GetName()), sameNameDoc, http.StatusBadRequest)
	} else {
		//post doc with same name should succeed
		newDoc := testPostDoc(suite, path, sameNameDoc, compareNewOpts...)
		//make sure GUID is generated
		if testOptions.customGUID {
			suite.NotEmpty(newDoc.GetGUID(), "GUID should be defined")
		} else {
			_, err := uuid.FromString(newDoc.GetGUID())
			suite.NoError(err, "GUID should be a valid uuid")
		}
		//delete the new doc
		testDeleteDocByGUIDWithOptions(suite, path, newDoc, &testOptions, compareNewOpts...)
	}
	noNameDoc := cloneDocFun(doc1)
	noNameDoc.SetGUID("")

	noNameDoc.SetName("")
	if testOptions.mandatoryName {
		//post doc with no name should fail
		testBadRequest(suite, http.MethodPost, path, errorMissingName, &noNameDoc, http.StatusBadRequest)
	} else {
		//post doc with no name should succeed
		newDoc := testPostDoc(suite, path, noNameDoc, compareNewOpts...)
		//make sure GUID is generated
		_, err := uuid.FromString(newDoc.GetGUID())
		suite.NoError(err, "GUID should be a valid uuid")
		//delete the new doc
		testDeleteDocByGUIDWithOptions(suite, path, newDoc, &testOptions, compareNewOpts...)
	}
	//bulk post documents
	updateTime, _ := time.Parse(time.RFC3339, time.Now().UTC().Format(time.RFC3339))
	documents = testBulkPostDocs(suite, path, documents, compareNewOpts...)
	//check updated time and fill names
	var names []string
	for _, doc := range documents {
		suite.NotNil(doc.GetUpdatedTime(), "updated time should not be nil")
		//check the the customer update date is updated
		suite.True(updateTime.Before(*doc.GetUpdatedTime()) || updateTime.Equal(*doc.GetUpdatedTime()), "update time is not recent")
		names = append(names, doc.GetName())
	}
	if testOptions.uniqueName {
		//bulk post documents with same name should fail
		sort.Strings(names)
		testBadRequest(suite, http.MethodPost, path, errorNameExist(names...), documents, http.StatusBadRequest)
	}
	if testOptions.allGUIDsAsInnerFilters {
		guids := []string{}
		for _, doc := range documents {
			guids = append(guids, doc.GetGUID())
		}
		innerFilters := []map[string]string{
			{
				"guid": strings.Join(guids, ","),
			},
		}
		v2Req := armotypes.V2ListRequest{
			InnerFilters: innerFilters,
			PageSize:     ptr.Int(0), //only count
			PageNum:      ptr.Int(1),
		}

		w := suite.doRequest(http.MethodPost, fmt.Sprintf("%s/query", path), v2Req)
		suite.Equal(http.StatusOK, w.Code)
		var result types.SearchResult[T]
		err := json.Unmarshal(w.Body.Bytes(), &result)
		suite.NoError(err)
		suite.Equal(len(guids), result.Total.Value)
	}

	//PUT
	if !testOptions.skipPutTests {

		oldDoc1 := cloneDocFun(doc1)
		doc1 = modifyFunc(doc1)
		//check that the doc update date is updated
		updateTime, _ = time.Parse(time.RFC3339, time.Now().UTC().Format(time.RFC3339))
		doc1 = testPutDoc(suite, path, oldDoc1, doc1, compareNewOpts...)

		suite.NotNil(doc1.GetUpdatedTime(), "updated time should not be nil")
		suite.True(updateTime.Before(*doc1.GetUpdatedTime()) || updateTime.Equal(*doc1.GetUpdatedTime()), "update time is not recent")

		//test changed name
		doc1name := doc1.GetName()
		changedNamedDoc := cloneDocFun(doc1)
		changedNamedDoc.SetName("new_name")

		w := suite.doRequest(http.MethodPut, path, changedNamedDoc)
		suite.Equal(http.StatusOK, w.Code)
		response, err := decodeResponseArray[T](w)
		if err != nil {
			suite.FailNow(err.Error())
		}
		var expectedResponse []T
		if !testOptions.renameAllowed {
			//rename should be ignored
			expectedResponse = []T{doc1, doc1}
		} else { //rename allowed
			expectedResponse = []T{doc1, changedNamedDoc}
			doc1 = changedNamedDoc
		}
		diff := cmp.Diff(response, expectedResponse, compareNewOpts...)
		suite.Equal("", diff)

		if testOptions.renameAllowed && testOptions.uniqueName {
			changedNamedDoc.SetName(doc2.GetName())
			//put doc with existing name should fail
			testBadRequest(suite, http.MethodPut, path, errorNameExist(doc2.GetName()), &changedNamedDoc, http.StatusBadRequest)
			changedNamedDoc.SetName("")
			testBadRequest(suite, http.MethodPut, path, errorMissingName, &changedNamedDoc, http.StatusBadRequest)
			//restore name
			doc1.SetName(doc1name)
			changedNamedDoc.SetName("new_name")
			doc1 = testPutDoc(suite, path, changedNamedDoc, doc1, compareNewOpts...)
		}

		//test put with guid in path
		oldDoc1 = cloneDocFun(doc1)
		doc1 = modifyFunc(doc1)
		doc1.SetGUID("")
		testPutDocWGuid(suite, path, oldDoc1, doc1, compareNewOpts...)

		//test put with no guid should fail
		noGuidDoc := cloneDocFun(doc1)
		noGuidDoc.SetGUID("")
		testBadRequest(suite, http.MethodPut, path, errorMissingGUID, &noGuidDoc, http.StatusBadRequest)

		//not existing doc should fail
		noneExistingDoc := cloneDocFun(doc1)
		noneExistingDoc.SetName("not_exist")
		noneExistingDoc.SetGUID("")
		noneExistingDoc.SetGUID("not_exist")
		testBadRequest(suite, http.MethodPut, path, errorDocumentNotFound, &noneExistingDoc, http.StatusNotFound)
	}

	//GET
	//test get by guid
	pathWGuid := fmt.Sprintf("%s/%s", path, doc1.GetGUID())
	testGetDocWithOptions(suite, pathWGuid, doc1, &testOptions, compareNewOpts...)
	//test get all
	docs := []T{doc1}
	docs = append(docs, documents...)
	testGetDocsWithOptions(suite, path, docs, &testOptions, compareNewOpts...)
	//test get with wrong guid should fail
	testBadRequest(suite, http.MethodGet, fmt.Sprintf("%s/%s", path, "no_exist"), errorDocumentNotFound, nil, http.StatusNotFound)

	//test delete by guid
	testDeleteDocByGUIDWithOptions(suite, path, doc1, &testOptions, compareNewOpts...)
	//test get all after delete
	testGetDocsWithOptions(suite, path, documents, &testOptions, compareNewOpts...)
	//delete the rest of the docs
	for _, doc := range documents {
		testDeleteDocByGUIDWithOptions(suite, path, doc, &testOptions, compareNewOpts...)
	}
	//test get all after delete all
	testGetDocs(suite, path, []T{}, compareNewOpts...)

	//test post and delete bulk
	testDeleteBulkDeleteByGUID(suite, path, documents, compareNewOpts...)

	//test get all after delete all
	testGetDocs(suite, path, []T{}, compareNewOpts...)

	//test delete doc with wrong guid should fail
	testBadRequest(suite, http.MethodDelete, fmt.Sprintf("%s/%s", path, "no_exist"), errorDocumentNotFound, nil, http.StatusNotFound)

}

// common test for (almost) all documents
func commonTest[T types.DocContent](suite *MainTestSuite, path string, testDocs []T, modifyFunc func(T) T, compareNewOpts ...cmp.Option) {
	testOptions := testOptions[T]{
		mandatoryName: true,
		uniqueName:    true,
		customGUID:    false,
	}
	commonTestWithOptions[T](suite, path, testDocs, modifyFunc, testOptions, compareNewOpts...)
}

func testPartialUpdate[T types.DocContent](suite *MainTestSuite, path string, emptyDoc T, compareOpts ...cmp.Option) {
	fullDoc := Clone(emptyDoc)
	partialDoc := Clone(emptyDoc)
	err := faker.FakeData(fullDoc, options.WithIgnoreInterface(true), options.WithGenerateUniqueValues(false),
		options.WithRandomMapAndSliceMaxSize(3), options.WithRandomMapAndSliceMinSize(2), options.WithNilIfLenIsZero(false), options.WithRecursionMaxDepth(5))
	if err != nil {
		suite.FailNow(err.Error())
	}
	fullDoc = Clone(fullDoc)
	if err != nil {
		suite.FailNow(err.Error())
	}
	fullAttr := fullDoc.GetAttributes()
	if fullAttr == nil {
		fullAttr = map[string]interface{}{}
	}
	fullAttr["alias"] = "new_alias"
	fullDoc.SetAttributes(fullAttr)
	fullDoc = testPostDoc(suite, path, fullDoc, compareOpts...)
	partialDoc.SetGUID(fullDoc.GetGUID())
	expectedFullDoc := Clone(fullDoc)
	partialCmpOpts := []cmp.Option{newClusterCompareFilter}
	partialCmpOpts = append(partialCmpOpts, compareOpts...)
	updatedDoc := testPutPartialDoc(suite, path, fullDoc, partialDoc, expectedFullDoc, partialCmpOpts...)
	testDeleteDocByGUID(suite, path, updatedDoc, compareOpts...)
}

type queryTest[T types.DocContent] struct {
	query           string
	expectedIndexes []int
}

type searchTest struct {
	testName         string
	listRequest      armotypes.V2ListRequest
	expectedIndexes  []int
	projectedResults bool
}

type uniqueValueTest struct {
	testName            string
	uniqueValuesRequest armotypes.UniqueValuesRequestV2
	expectedResponse    armotypes.UniqueValuesResponseV2
}

func testGetByName[T types.DocContent](suite *MainTestSuite, basePath, nameParam string, testDocs []T, compareOpts ...cmp.Option) {
	w := suite.doRequest(http.MethodPost, basePath, testDocs)
	suite.Equal(http.StatusCreated, w.Code)
	newDocs, err := decodeResponseArray[T](w)
	if err != nil {
		suite.FailNow(err.Error())
	}
	suite.Equal(len(testDocs), len(newDocs))
	docNames := []string{}
	for i := range newDocs {
		docNames = append(docNames, testDocs[i].GetName())
	}

	//test get name list
	testGetNameList(suite, basePath, docNames)
	//test get by name
	path := fmt.Sprintf("%s?%s=%s", basePath, nameParam, newDocs[0].GetName())
	testGetDoc(suite, path, newDocs[0], compareOpts...)
	//test get by not existing name
	path = fmt.Sprintf("%s?%s=%s", basePath, nameParam, "notExistingName")
	testBadRequest(suite, http.MethodGet, path, errorDocumentNotFound, nil, http.StatusNotFound)

	for _, doc := range newDocs {
		testDeleteDocByGUID(suite, basePath, doc, compareOpts...)
	}

}

func testDeleteByName[T types.DocContent](suite *MainTestSuite, basePath, nameParam string, testDocs []T, compareOpts ...cmp.Option) {
	newDocs := testBulkPostDocs(suite, basePath, testDocs, commonCmpFilter)
	suite.Equal(len(testDocs), len(newDocs))
	docNames := []string{}
	for i := range newDocs {
		docNames = append(docNames, testDocs[i].GetName())
	}
	//test delete by name
	testDeleteDocByName(suite, basePath, nameParam, newDocs[0], compareOpts...)
	//test bulk delete by name
	docNames2 := docNames[1:]
	testBulkDeleteByName(suite, basePath, nameParam, docNames2)
	//test delete by name with not existing name
	path := fmt.Sprintf("%s?%s=%s", basePath, nameParam, "notExistingName")
	testBadRequest(suite, http.MethodDelete, path, errorDocumentNotFound, nil, http.StatusNotFound)
	//deleteDoc by name with empty name
	path = fmt.Sprintf("%s?%s", basePath, nameParam)
	testBadRequest(suite, http.MethodDelete, path, errorMissingName, nil, http.StatusBadRequest)
	//test bulk delete with body
	testBulkPostDocs(suite, basePath, testDocs, commonCmpFilter)
	testBulkDeleteByNameWithBody(suite, basePath, nameParam, docNames)
}

func testDeleteBulkDeleteByGUID[T types.DocContent](suite *MainTestSuite, basePath string, testDocs []T, compareOpts ...cmp.Option) {
	//test bulk delete by Ids in body
	newDocs := testBulkPostDocs(suite, basePath, testDocs, compareOpts...)
	docsIds := make([]string, 0, len(newDocs))
	for i := range newDocs {
		docsIds = append(docsIds, newDocs[i].GetGUID())
	}
	testBulkDeleteByGUIDWithBody(suite, basePath, docsIds)
	//test bulk delete by Ids in query
	newDocs = testBulkPostDocs(suite, basePath, testDocs, compareOpts...)
	docsIds = make([]string, 0, len(newDocs))
	for i := range newDocs {
		docsIds = append(docsIds, newDocs[i].GetGUID())
	}
	testBulkDeleteByGUIDWithQuery(suite, basePath, docsIds)

	//test delete by v2 list query
	newDocs = testBulkPostDocs(suite, basePath, testDocs, compareOpts...)
	docsIds = make([]string, 0, len(newDocs))
	for i := range newDocs {
		docsIds = append(docsIds, newDocs[i].GetGUID())
	}
	v2Req := armotypes.V2ListRequest{
		InnerFilters: []map[string]string{
			{
				"guid": strings.Join(docsIds, ","),
			},
		}}
	testDeleteByQuery(suite, basePath, v2Req, len(docsIds))

	//test delete all with empty query
	newDocs = testBulkPostDocs(suite, basePath, testDocs, compareOpts...)
	v2Req = armotypes.V2ListRequest{}
	testDeleteByQuery(suite, basePath, v2Req, len(newDocs))

}

func testGetByQuery[T types.DocContent](suite *MainTestSuite, basePath string, testDocs []T, getQueries []queryTest[T], compareOpts ...cmp.Option) {
	newDocs := testBulkPostDocs(suite, basePath, testDocs, commonCmpFilter)
	suite.Equal(len(testDocs), len(newDocs))
	docNames := []string{}
	for i := range newDocs {
		docNames = append(docNames, testDocs[i].GetName())
	}
	//get Docs by query params
	testGetWithQuery(suite, basePath, getQueries, newDocs, compareOpts...)
	for _, doc := range newDocs {
		testDeleteDocByGUID(suite, basePath, doc, compareOpts...)
	}

}

func testPostV2ListRequest[T types.DocContent](suite *MainTestSuite, basePath string, testDocs []T, projectionTestDocs []T, tests []searchTest, compareOpts ...cmp.Option) {
	newDocs := testBulkPostDocs(suite, basePath, testDocs, compareOpts...)

	for _, test := range tests {
		w := suite.doRequest(http.MethodPost, basePath+"/query", test.listRequest)
		suite.Equal(http.StatusOK, w.Code)
		var result types.SearchResult[T]
		err := json.Unmarshal(w.Body.Bytes(), &result)
		suite.NoError(err, "Unexpected error: %s", test.testName)
		suite.Equal(len(test.expectedIndexes), len(result.Response), "Unexpected result count: %s", test.testName)
		if test.listRequest.PageSize == nil && test.listRequest.PageNum == nil {
			//not paginated expected all results
			suite.Equal(len(test.expectedIndexes), result.Total.Value, "Unexpected total count: %s", test.testName)
		}
		var expectedDocs []T
		for _, index := range test.expectedIndexes {
			if test.projectedResults {
				expectedDocs = append(expectedDocs, projectionTestDocs[index])
			} else {
				expectedDocs = append(expectedDocs, testDocs[index])
			}
		}
		if len(expectedDocs) != 0 {
			diff := cmp.Diff(result.Response, expectedDocs, compareOpts...)
			suite.Equal("", diff, "Unexpected diff: %s", test.testName)
		}
	}

	//test bad requests for search
	//unsupported operation
	req := armotypes.V2ListRequest{
		InnerFilters: []map[string]string{
			{
				"name": "name|unknownOp",
			},
		},
	}
	testBadRequest(suite, http.MethodPost, basePath+"/query", errorUnsupportedOperator("unknownOp"), req, http.StatusBadRequest)

	//invalid sort direction
	req = armotypes.V2ListRequest{
		OrderBy: "name:unknownOrder",
	}
	testBadRequest(suite, http.MethodPost, basePath+"/query", errorMessage("invalid sort field name:unknownOrder:desc"), req, http.StatusBadRequest)

	//use of inclusive time window
	maxTimeForDocs := time.Now()
	maxTimeForDocs = maxTimeForDocs.Add(time.Hour * 24)

	req = armotypes.V2ListRequest{
		Since: ptr.Time(time.Time{}),
		Until: ptr.Time(maxTimeForDocs),
	}

	w := suite.doRequest(http.MethodPost, basePath+"/query", req)
	suite.Equal(http.StatusOK, w.Code)
	var qResult types.SearchResult[T]
	err := json.Unmarshal(w.Body.Bytes(), &qResult)
	suite.NoError(err, "Unexpected error: %s", "use of supported time window")
	allDocs := qResult.Total.Value
	suite.Equal(len(newDocs), allDocs, "Unexpected result count: %s", "use of supported time window")

	// use of exclusive since
	req = armotypes.V2ListRequest{
		Since: ptr.Time(maxTimeForDocs),
		Until: ptr.Time(maxTimeForDocs),
	}
	w = suite.doRequest(http.MethodPost, basePath+"/query", req)
	suite.Equal(http.StatusOK, w.Code)

	err = json.Unmarshal(w.Body.Bytes(), &qResult)
	suite.NoError(err, "Unexpected error: %s", "use of exclusive since")
	suite.Equal(0, qResult.Total.Value, "Unexpected result count: %s", "use of exclusive since")

	// use of exclusive until
	req = armotypes.V2ListRequest{
		Since: ptr.Time(time.Time{}),
		Until: ptr.Time(time.Time{}),
	}
	w = suite.doRequest(http.MethodPost, basePath+"/query", req)
	suite.Equal(http.StatusOK, w.Code)
	err = json.Unmarshal(w.Body.Bytes(), &qResult)
	suite.NoError(err, "Unexpected error: %s", "use of exclusive until")
	suite.Equal(0, qResult.Total.Value, "Unexpected result count: %s", "use of exclusive until")

	//range with no &
	req = armotypes.V2ListRequest{
		InnerFilters: []map[string]string{
			{
				"name": "something|range",
			},
		},
	}
	testBadRequest(suite, http.MethodPost, basePath+"/query", errorMessage("value missing range separator something"), req, http.StatusBadRequest)

	//range with different data types
	req = armotypes.V2ListRequest{
		InnerFilters: []map[string]string{
			{
				"name": "1&name|range",
			},
		},
	}
	testBadRequest(suite, http.MethodPost, basePath+"/query", errorMessage("invalid range must use same value types found int64 string"), req, http.StatusBadRequest)

	//bulk delete all docs
	guids := []string{}
	for _, doc := range newDocs {
		guids = append(guids, doc.GetGUID())
	}
	testBulkDeleteByGUIDWithBody(suite, basePath, guids)
}

func testUniqueValues[T types.DocContent](suite *MainTestSuite, basePath string, testDocs []T, uniqueValuesTests []uniqueValueTest, compareOpts ...cmp.Option) {
	newDocs := testBulkPostDocs(suite, basePath, testDocs, compareOpts...)
	//unique values
	for _, test := range uniqueValuesTests {
		w := suite.doRequest(http.MethodPost, basePath+"/uniqueValues", test.uniqueValuesRequest)
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
	for _, doc := range newDocs {
		guids = append(guids, doc.GetGUID())
	}
	testBulkDeleteByGUIDWithBody(suite, basePath, guids)

	//test uniqueValues bad requests
	req := armotypes.UniqueValuesRequestV2{}
	testBadRequest(suite, http.MethodPost, basePath+"/uniqueValues", errorMessage("fields are required"), req, http.StatusBadRequest)
	//fix the bad request
	req = armotypes.UniqueValuesRequestV2{
		Fields: map[string]string{
			"name": "",
		},
		Since: ptr.Time(time.Time{}),
		Until: ptr.Time(time.Now()),
	}
	w := suite.doRequest(http.MethodPost, basePath+"/uniqueValues", req)
	suite.Equal(http.StatusOK, w.Code)
}

func testGetDeleteByNameAndQuery[T types.DocContent](suite *MainTestSuite, basePath, nameParam string, testDocs []T, getQueries []queryTest[T], compareOpts ...cmp.Option) {
	testGetByName(suite, basePath, nameParam, testDocs, compareOpts...)

	testGetByQuery(suite, basePath, testDocs, getQueries, compareOpts...)

	testDeleteByName(suite, basePath, nameParam, testDocs, compareOpts...)

}

func testGetWithQuery[T types.DocContent](suite *MainTestSuite, basePath string, getQueries []queryTest[T], expected []T, compareOpts ...cmp.Option) {
	//get Docs by query params
	for queryID, query := range getQueries {
		path := fmt.Sprintf("%s?%s", basePath, query.query)
		var expectedDocs []T
		for _, index := range query.expectedIndexes {
			expectedDocs = append(expectedDocs, expected[index])
		}
		testGetDocs(suite, path, expectedDocs, compareOpts...)
		if suite.T().Failed() {
			suite.FailNow("Failed query: %d", queryID)
		}
	}
}

////////////////////////////////////////// Test helpers //////////////////////////////////////////

const (
	//error messages
	errorMissingName      = `{"error":"name is required"}`
	errorMissingGUID      = `{"error":"guid is required"}`
	errorGUIDExists       = `{"error":"guid already exists"}`
	errorDocumentNotFound = `{"error":"document not found"}`
	errorNotAdminUser     = `{"error":"Unauthorized - not an admin user"}`
)

func errorBadTimeParam(paramName string) string {
	return `{"error":"` + paramName + ` must be in RFC3339 format"}`
}

func errorParamType(paramName, typeName string) string {
	return `{"error":"` + paramName + ` must be a ` + typeName + `"}`
}

func errorMissingQueryParams(params ...string) string {
	if len(params) == 1 {
		return `{"error":"` + params[0] + ` query param is required"}`
	} else if len(params) > 1 {
		return `{"error":"` + strings.Join(params, ",") + ` query params are required"}`
	}
	return `{"error":"missing query params"}`
}

func errorNameExist(name ...string) string {
	var msg string
	if len(name) == 0 {
		msg = "name already exists"
	} else if len(name) == 1 {
		msg = fmt.Sprintf("name %s already exists", name[0])
	} else {
		msg = fmt.Sprintf("names %s already exist", strings.Join(name, ","))
	}
	return `{"error":"` + msg + `"}`
}

func errorUnsupportedOperator(op string) string {
	return `{"error":"unsupported operator ` + op + `"}`
}

func errorInvalidSortType(sortType string) string {
	return fmt.Sprintf(`{"error":"invalid sort type %s"}`, sortType)
}
func errorMessage(msg string) string {
	return fmt.Sprintf(`{"error":"%s"}`, msg)
}

func testBadRequest(suite *MainTestSuite, method, path, expectedResponse string, body interface{}, expectedCode int) {
	w := suite.doRequest(method, path, body)
	suite.Equal(expectedCode, w.Code)
	suite.Equal(expectedResponse, w.Body.String())
}

func testGetDocWithOptions[T any](suite *MainTestSuite, path string, expectedDoc T, testOptions *testOptions[T], compareOpts ...cmp.Option) T {
	w := suite.doRequest(http.MethodGet, path, nil)
	suite.Equal(http.StatusOK, w.Code)
	doc, err := decodeResponse[T](w)
	if err != nil {
		suite.FailNow(err.Error())
	}
	diff := cmp.Diff(doc, expectedDoc, compareOpts...)
	suite.Equal("", diff)
	return doc
}

// //////////////////////////////////////// GET //////////////////////////////////////////
func testGetDoc[T any](suite *MainTestSuite, path string, expectedDoc T, compareOpts ...cmp.Option) T {
	return testGetDocWithOptions[T](suite, path, expectedDoc, nil, compareOpts...)
}

func testGetDocsWithOptions[T types.DocContent](suite *MainTestSuite, path string, expectedDocs []T, testOptions *testOptions[T], compareOpts ...cmp.Option) (actualDocs []T) {
	w := suite.doRequest(http.MethodGet, path, nil)
	suite.Equal(http.StatusOK, w.Code)
	docs, err := decodeResponseArray[T](w)
	if err != nil {
		suite.FailNow(err.Error())
	}
	sortFunc := func(a T, b T) int {
		if nameInt := strings.Compare(a.GetName(), b.GetName()); nameInt != 0 {
			return nameInt
		} else {
			return strings.Compare(a.GetGUID(), b.GetGUID())
		}
	}
	slices.SortFunc(docs, sortFunc)
	slices.SortFunc(expectedDocs, sortFunc)
	diff := cmp.Diff(docs, expectedDocs, compareOpts...)
	suite.Equal("", diff)
	return docs
}

func testGetDocs[T types.DocContent](suite *MainTestSuite, path string, expectedDocs []T, compareOpts ...cmp.Option) (actualDocs []T) {
	return testGetDocsWithOptions(suite, path, expectedDocs, nil, compareOpts...)
}

func testGetNameList(suite *MainTestSuite, path string, expectedNames []string) {
	path = fmt.Sprintf("%s?list", path)
	w := suite.doRequest(http.MethodGet, path, nil)
	suite.Equal(http.StatusOK, w.Code)

	names := decodeArray[string](suite, w.Body.Bytes())
	sort.Strings(expectedNames)
	sort.Strings(names)
	diff := cmp.Diff(names, expectedNames)
	suite.Equal("", diff)
}

// //////////////////////////////////////// POST //////////////////////////////////////////
func testPostDoc[T types.DocContent](suite *MainTestSuite, path string, doc T, compareOpts ...cmp.Option) (newDoc T) {
	w := suite.doRequest(http.MethodPost, path, doc)
	suite.Equal(http.StatusCreated, w.Code)
	newDoc, err := decodeResponse[T](w)
	if err != nil {
		suite.FailNow(err.Error())
	}
	diff := cmp.Diff(doc, newDoc, compareOpts...)
	suite.Equal("", diff)
	return newDoc
}

func testBulkPostDocs[T types.DocContent](suite *MainTestSuite, path string, docs []T, compareOpts ...cmp.Option) (newDocs []T) {
	w := suite.doRequest(http.MethodPost, path, docs)
	suite.Equal(http.StatusCreated, w.Code)
	newDocs, err := decodeResponseArray[T](w)
	if err != nil {
		suite.FailNow(err.Error())
	}
	sort.Slice(docs, func(i, j int) bool {
		return docs[i].GetName() < docs[j].GetName()
	})
	sort.Slice(newDocs, func(i, j int) bool {
		return newDocs[i].GetName() < newDocs[j].GetName()
	})
	diff := cmp.Diff(docs, newDocs, compareOpts...)
	suite.Equal("", diff)
	return newDocs
}

// //////////////////////////////////////// PUT //////////////////////////////////////////
func testPutDoc[T any](suite *MainTestSuite, path string, oldDoc, newDoc T, compareNewOpts ...cmp.Option) T {
	w := suite.doRequest(http.MethodPut, path, newDoc)
	suite.Equal(http.StatusOK, w.Code)
	response, err := decodeResponseArray[T](w)
	expectedResponse := []T{oldDoc, newDoc}
	if err != nil {
		suite.FailNow(err.Error())
	}
	diff := cmp.Diff(response, expectedResponse, compareNewOpts...)
	suite.Equal("", diff)
	return response[1]
}

func testPutPartialDoc[T any](suite *MainTestSuite, path string, oldDoc T, newPartialDoc interface{}, expectedFullDoc T, compareNewOpts ...cmp.Option) T {
	w := suite.doRequest(http.MethodPut, path, newPartialDoc)
	suite.Equal(http.StatusOK, w.Code)
	response, err := decodeResponseArray[T](w)
	expectedResponse := []T{oldDoc, expectedFullDoc}
	if err != nil {
		suite.FailNow(err.Error())
	}
	diff := cmp.Diff(response, expectedResponse, compareNewOpts...)
	suite.Equal("", diff)
	return response[1]
}

func testPutDocWGuid[T types.DocContent](suite *MainTestSuite, path string, oldDoc, newDoc T, compareNewOpts ...cmp.Option) {
	guid := oldDoc.GetGUID()
	path = fmt.Sprintf("%s/%s", path, guid)
	newDoc.SetGUID("")
	w := suite.doRequest(http.MethodPut, path, newDoc)
	suite.Equal(http.StatusOK, w.Code)
	response, err := decodeResponseArray[T](w)
	if err != nil {
		suite.FailNow(err.Error())
	}
	newDoc.SetGUID(guid)
	expectedResponse := []T{oldDoc, newDoc}
	sort.Slice(response, func(i, j int) bool {
		return response[i].GetName() < response[j].GetName()
	})
	sort.Slice(expectedResponse, func(i, j int) bool {
		return expectedResponse[i].GetName() < expectedResponse[j].GetName()
	})
	diff := cmp.Diff(response, expectedResponse, compareNewOpts...)
	suite.Equal("", diff)
}

func testDeleteDocByGUIDWithOptions[T types.DocContent](suite *MainTestSuite, path string, doc2Delete T, testOptions *testOptions[T], compareOpts ...cmp.Option) {
	path = fmt.Sprintf("%s/%s", path, doc2Delete.GetGUID())
	w := suite.doRequest(http.MethodDelete, path, nil)
	suite.Equal(http.StatusOK, w.Code)
	deleteDoc, err := decodeResponse[T](w)
	if err != nil {
		suite.FailNow(err.Error())
	}
	diff := cmp.Diff(deleteDoc, doc2Delete, compareOpts...)
	suite.Equal("", diff)
}

// //////////////////////////////////////// DELETE //////////////////////////////////////////
func testDeleteDocByGUID[T types.DocContent](suite *MainTestSuite, path string, doc2Delete T, compareOpts ...cmp.Option) {
	testDeleteDocByGUIDWithOptions(suite, path, doc2Delete, nil, compareOpts...)
}

func testDeleteDocByName[T types.DocContent](suite *MainTestSuite, path string, nameParam string, doc2Delete T, compareOpts ...cmp.Option) {
	path = fmt.Sprintf("%s?%s=%s", path, nameParam, doc2Delete.GetName())
	w := suite.doRequest(http.MethodDelete, path, nil)
	suite.Equal(http.StatusOK, w.Code)
	deleteDoc, err := decodeResponse[T](w)
	if err != nil {
		suite.FailNow(err.Error())
	}
	diff := cmp.Diff(deleteDoc, doc2Delete, compareOpts...)
	suite.Equal("", diff)
}

func testBulkDeleteByName(suite *MainTestSuite, path string, nameParam string, names []string) {
	if len(names) == 0 {
		return
	}
	path = fmt.Sprintf("%s?%s=%s", path, nameParam, names[0])
	for _, name := range names[1:] {
		path = fmt.Sprintf("%s&%s=%s", path, nameParam, name)
	}
	w := suite.doRequest(http.MethodDelete, path, nil)
	suite.Equal(http.StatusOK, w.Code)
	diff := cmp.Diff(fmt.Sprintf(`{"deletedCount":%d}`, len(names)), w.Body.String())
	suite.Equal("", diff)
}

func testBulkDeleteByNameWithBody(suite *MainTestSuite, path string, nameParam string, names []string) {
	if len(names) == 0 {
		return
	}
	namesBody := []map[string]string{}
	for _, name := range names {
		namesBody = append(namesBody, map[string]string{nameParam: name})
	}
	w := suite.doRequest(http.MethodDelete, path, namesBody)
	suite.Equal(http.StatusOK, w.Code)
	diff := cmp.Diff(fmt.Sprintf(`{"deletedCount":%d}`, len(names)), w.Body.String())
	suite.Equal("", diff)
}

func testBulkDeleteByGUIDWithBody(suite *MainTestSuite, path string, guids []string) {
	if len(guids) == 0 {
		return
	}
	w := suite.doRequest(http.MethodDelete, path+"/bulk", guids)
	suite.Equal(http.StatusOK, w.Code)
	diff := cmp.Diff(fmt.Sprintf(`{"deletedCount":%d}`, len(guids)), w.Body.String())
	suite.Equal("", diff)
}

func testDeleteByQuery(suite *MainTestSuite, path string, request armotypes.V2ListRequest, expectedCount int) {
	w := suite.doRequest(http.MethodDelete, path+"/query", request)
	suite.Equal(http.StatusOK, w.Code)
	diff := cmp.Diff(fmt.Sprintf(`{"deletedCount":%d}`, expectedCount), w.Body.String())
	suite.Equal("", diff)
}

func testBulkDeleteByGUIDWithQuery(suite *MainTestSuite, path string, guids []string) {
	if len(guids) == 0 {
		return
	}
	path += "/bulk"
	path = fmt.Sprintf("%s?%s=%s", path, "guid", guids[0])
	for _, name := range guids[1:] {
		path = fmt.Sprintf("%s&%s=%s", path, "guid", name)
	}
	w := suite.doRequest(http.MethodDelete, path, nil)
	suite.Equal(http.StatusOK, w.Code)
	diff := cmp.Diff(fmt.Sprintf(`{"deletedCount":%d}`, len(guids)), w.Body.String())
	suite.Equal("", diff)
}

//helpers

func decodeResponse[T any](w *httptest.ResponseRecorder) (T, error) {
	var content T
	err := json.Unmarshal(w.Body.Bytes(), &content)
	return content, err
}

func decodeResponseArray[T any](w *httptest.ResponseRecorder) ([]T, error) {
	var content []T
	err := json.Unmarshal(w.Body.Bytes(), &content)
	return content, err
}

func decode[T any](suite *MainTestSuite, bytes []byte) T {
	var content T
	if err := json.Unmarshal(bytes, &content); err != nil {
		suite.FailNow("failed to decode", err.Error())
	}
	return content
}

func decodeArray[T any](suite *MainTestSuite, bytes []byte) []T {
	var content []T
	if err := json.Unmarshal(bytes, &content); err != nil {
		suite.FailNow("failed to decode", err.Error())
	}
	return content
}

func Clone[T any](orig T) T {
	origJSON, err := json.Marshal(orig)
	if err != nil {
		panic(err)
	}
	var clone T
	if err = json.Unmarshal(origJSON, &clone); err != nil {
		panic(err)
	}
	return clone
}
