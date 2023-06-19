package main

import (
	"config-service/types"
	"encoding/json"
	"fmt"
	"time"

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
// tests for documents which have name as unique
func testDocNameUnique[T types.DocContent](suite *MainTestSuite, doc1 T, path string, documents []T, compareNewOpts ...cmp.Option) {
	//post doc with same name should fail
	sameNameDoc := clone(doc1)
	testBadRequest(suite, http.MethodPost, path, errorNameExist(sameNameDoc.GetName()), sameNameDoc, http.StatusBadRequest)

	//post doc with no name should fail
	noNameDoc := clone(doc1)
	noNameDoc.SetName("")
	testBadRequest(suite, http.MethodPost, path, errorMissingName, &noNameDoc, http.StatusBadRequest)

	names := []string{documents[0].GetName(), documents[1].GetName()}
	sort.Strings(names)
	testBadRequest(suite, http.MethodPost, path, errorNameExist(names...), documents, http.StatusBadRequest)

	//test changed name - should be ignored
	changedNamedDoc := clone(doc1)
	changedNamedDoc.SetName("new_name")
	w := suite.doRequest(http.MethodPut, path, changedNamedDoc)
	suite.Equal(http.StatusOK, w.Code)
	response, err := decodeResponseArray[T](w)
	expectedResponse := []T{doc1, doc1}
	if err != nil {
		suite.FailNow(err.Error())
	}
	diff := cmp.Diff(response, expectedResponse, compareNewOpts...)
	suite.Equal("", diff)

}

// common test for (almost) all documents
func commonTest[T types.DocContent](suite *MainTestSuite, path string, doc1 T, documents []T, modifyFunc func(T) T, createTime, updateTime time.Time, compareNewOpts ...cmp.Option) {
	//check creation time
	suite.NotNil(doc1.GetCreationTime(), "creation time should not be nil")
	suite.True(createTime.Before(*doc1.GetCreationTime()) || createTime.Equal(*doc1.GetCreationTime()), "creation time is not recent")

	//check updated time
	for _, doc := range documents {
		suite.NotNil(doc.GetUpdatedTime(), "updated time should not be nil")
		//check the the customer update date is updated
		suite.True(updateTime.Before(*doc.GetUpdatedTime()) || updateTime.Equal(*doc.GetUpdatedTime()), "update time is not recent")
	}

	//PUT
	oldDoc1 := clone(doc1)
	doc1 = modifyFunc(doc1)
	updateTime, _ = time.Parse(time.RFC3339, time.Now().UTC().Format(time.RFC3339))
	testPutDoc(suite, path, oldDoc1, doc1, compareNewOpts...)
	suite.NotNil(doc1.GetUpdatedTime(), "updated time should not be nil")

	//check the the customer update date is updated
	suite.True(updateTime.Before(*doc1.GetUpdatedTime()) || updateTime.Equal(*doc1.GetUpdatedTime()), "update time is not recent")

	//test put with guid in path
	oldDoc1 = clone(doc1)
	doc1 = modifyFunc(doc1)
	testPutDocWGuid(suite, path, oldDoc1, doc1, compareNewOpts...)

	//GET
	//test get by guid
	pathWGuid := fmt.Sprintf("%s/%s", path, doc1.GetGUID())
	testGetDoc(suite, pathWGuid, doc1, compareNewOpts...)

	//test get with wrong guid should fail
	testBadRequest(suite, http.MethodGet, fmt.Sprintf("%s/%s", path, "no_exist"), errorDocumentNotFound, nil, http.StatusNotFound)

	//test delete doc with wrong guid should fail
	testBadRequest(suite, http.MethodDelete, fmt.Sprintf("%s/%s", path, "no_exist"), errorDocumentNotFound, nil, http.StatusNotFound)
}

// tests for documents which have a non-empty set guid implementation
func testWithSetGUID[T types.DocContent](suite *MainTestSuite, doc1 T, path string, compareNewOpts ...cmp.Option) {
	//create doc
	newDoc := clone(doc1)
	newDoc.SetGUID("some bad value")
	newDoc = testPostDoc(suite, path, doc1, compareNewOpts...)
	_, err := uuid.FromString(newDoc.GetGUID())
	suite.NoError(err, "GUID should be a valid uuid")
	testDeleteDocByGUID(suite, path, newDoc, compareNewOpts...)

	// test put with no guid should fail
	noGuidDoc := clone(newDoc)
	noGuidDoc.SetGUID("")
	testBadRequest(suite, http.MethodPut, path, errorMissingGUID, &noGuidDoc, http.StatusBadRequest)

	// not existing doc should fail
	noneExistingDoc := clone(newDoc)
	noneExistingDoc.SetGUID("no_exist")
	testBadRequest(suite, http.MethodPut, path, errorDocumentNotFound, &noneExistingDoc, http.StatusNotFound)
}

func testGetAndDeleteAll[T types.DocContent](suite *MainTestSuite, doc1 T, path string, documents []T, compareNewOpts ...cmp.Option) {
	//test get all
	docs := []T{doc1}
	docs = append(docs, documents...)
	testGetDocs(suite, path, docs, compareNewOpts...)

	//test delete by guid
	testDeleteDocByGUID(suite, path, doc1, compareNewOpts...)

	//test get all after delete
	testGetDocs(suite, path, documents, compareNewOpts...)

	//delete the rest of the docs
	for _, doc := range documents {
		testDeleteDocByGUID(suite, path, doc, compareNewOpts...)
	}

	//test get all after delete all
	testGetDocs(suite, path, []T{}, compareNewOpts...)

}

func testDeleteDocs[T types.DocContent](suite *MainTestSuite, path string, doc1 T, documents []T, compareNewOpts ...cmp.Option) {
	//test delete by guid
	testDeleteDocByGUID(suite, path, doc1, compareNewOpts...)
	//test get all after delete
	testGetDocs(suite, path, documents, compareNewOpts...)
	//delete the rest of the docs
	for _, doc := range documents {
		testDeleteDocByGUID(suite, path, doc, compareNewOpts...)
	}
}

func testPartialUpdate[T types.DocContent](suite *MainTestSuite, path string, emptyDoc T, compareOpts ...cmp.Option) {
	fullDoc := clone(emptyDoc)
	partialDoc := clone(emptyDoc)
	err := faker.FakeData(fullDoc, options.WithIgnoreInterface(true), options.WithGenerateUniqueValues(false),
		options.WithRandomMapAndSliceMaxSize(3), options.WithRandomMapAndSliceMinSize(2), options.WithNilIfLenIsZero(false), options.WithRecursionMaxDepth(5))
	if err != nil {
		suite.FailNow(err.Error())
	}
	fullDoc = clone(fullDoc)
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
	expectedFullDoc := clone(fullDoc)
	updatedDoc := testPutPartialDoc(suite, path, fullDoc, partialDoc, expectedFullDoc, newClusterCompareFilter)
	testDeleteDocByGUID(suite, path, updatedDoc, compareOpts...)
}

type queryTest[T types.DocContent] struct {
	query           string
	expectedIndexes []int
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

func testGetByQuery[T types.DocContent](suite *MainTestSuite, basePath string, testDocs []T, getQueries []queryTest[T], compareOpts ...cmp.Option) {
	newDocs := testBulkPostDocs(suite, basePath, testDocs, commonCmpFilter)
	suite.Equal(len(testDocs), len(newDocs))
	docNames := []string{}
	for i := range newDocs {
		docNames = append(docNames, testDocs[i].GetName())
	}
	//get Docs by query params
	testGetWithQuery(suite, basePath, getQueries, newDocs)
	for _, doc := range newDocs {
		testDeleteDocByGUID(suite, basePath, doc, compareOpts...)
	}

}

func testGetDeleteByNameAndQuery[T types.DocContent](suite *MainTestSuite, basePath, nameParam string, testDocs []T, getQueries []queryTest[T], compareOpts ...cmp.Option) {
	testGetByName(suite, basePath, nameParam, testDocs, compareOpts...)

	testGetByQuery(suite, basePath, testDocs, getQueries, compareOpts...)

	testDeleteByName(suite, basePath, nameParam, testDocs, compareOpts...)

}

func testGetWithQuery[T types.DocContent](suite *MainTestSuite, basePath string, getQueries []queryTest[T], expected []T, compareOpts ...cmp.Option) {
	//get Docs by query params
	for _, query := range getQueries {
		path := fmt.Sprintf("%s?%s", basePath, query.query)
		var expectedDocs []T
		for _, index := range query.expectedIndexes {
			expectedDocs = append(expectedDocs, expected[index])
		}
		testGetDocs(suite, path, expectedDocs)
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

func testBadRequest(suite *MainTestSuite, method, path, expectedResponse string, body interface{}, expectedCode int) {
	w := suite.doRequest(method, path, body)
	suite.Equal(expectedCode, w.Code)
	suite.Equal(expectedResponse, w.Body.String())
}

// //////////////////////////////////////// GET //////////////////////////////////////////
func testGetDoc[T any](suite *MainTestSuite, path string, expectedDoc T, compareOpts ...cmp.Option) T {
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

func testGetDocs[T types.DocContent](suite *MainTestSuite, path string, expectedDocs []T, compareOpts ...cmp.Option) (actualDocs []T) {
	w := suite.doRequest(http.MethodGet, path, nil)
	suite.Equal(http.StatusOK, w.Code)
	docs, err := decodeResponseArray[T](w)
	if err != nil {
		suite.FailNow(err.Error())
	}
	sort.Slice(docs, func(i, j int) bool {
		return docs[i].GetName() < docs[j].GetName()
	})
	sort.Slice(expectedDocs, func(i, j int) bool {
		return expectedDocs[i].GetName() < expectedDocs[j].GetName()
	})
	diff := cmp.Diff(docs, expectedDocs, compareOpts...)
	suite.Equal("", diff)
	return docs
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
func testPutDoc[T any](suite *MainTestSuite, path string, oldDoc, newDoc T, compareNewOpts ...cmp.Option) {
	w := suite.doRequest(http.MethodPut, path, newDoc)
	suite.Equal(http.StatusOK, w.Code)
	response, err := decodeResponseArray[T](w)
	expectedResponse := []T{oldDoc, newDoc}
	if err != nil {
		suite.FailNow(err.Error())
	}
	diff := cmp.Diff(response, expectedResponse, compareNewOpts...)
	suite.Equal("", diff)
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
	guid := newDoc.GetGUID()
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

// //////////////////////////////////////// DELETE //////////////////////////////////////////
func testDeleteDocByGUID[T types.DocContent](suite *MainTestSuite, path string, doc2Delete T, compareOpts ...cmp.Option) {
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

func clone[T any](orig T) T {
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
