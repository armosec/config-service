package main

import (
	"bytes"
	"config-service/db/mongo"
	"config-service/types"
	"config-service/utils/consts"
	"context"
	_ "embed"
	"fmt"
	"sort"

	"net/http"
	"net/http/httptest"
	"os/exec"
	"testing"
	"time"

	"encoding/json"

	"github.com/gin-gonic/gin"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/suite"
)

/*
Coverage report cmd
go test -timeout 30s  -coverpkg=./handlers,./db,./types,./routes/prob,./routes/login,./routes/cluster,./routes/posture_exception,./routes/vulnerability_exception,./routes/customer,./routes/customer_config -coverprofile coverage.out  \
&& \
go tool cover -html=coverage.out -o coverage.html
*/

const (
	mongoDockerCommand = `docker run --name=mongo -d -p 27017:27017 -e  "MONGO_INITDB_ROOT_USERNAME=admin" -e "MONGO_INITDB_ROOT_PASSWORD=admin" mongo`
	mongoStopCommand   = "docker stop mongo && docker rm mongo"
	defaultUserGUID    = "test-customer-guid"
)

//go:embed test_data/customer_config/defaultConfig.json
var defaultCustomerConfigJson []byte

func TestConfigServiceWithMongoImage(t *testing.T) {
	suite.Run(t, new(MainTestSuite))
}

type MainTestSuite struct {
	suite.Suite
	router           *gin.Engine
	shutdownFunc     func()
	authCookie       string
	authCustomerGUID string
}

func (suite *MainTestSuite) SetupSuite() {
	//start mongo
	exec.Command("/bin/sh", "-c", mongoStopCommand).Run()
	out, err := exec.Command("/bin/sh", "-c", mongoDockerCommand).Output()
	if err != nil {
		suite.FailNow("failed to start mongo", err.Error(), string(out))
	}

	//initialize service
	suite.shutdownFunc = initialize()
	//Create routes
	suite.router = setupRouter()
	//wait for service to be ready
	checkReadiness := func() error {
		w := suite.doRequest(http.MethodGet, "/readiness", nil)
		if w.Code != http.StatusOK {
			return fmt.Errorf("failed to get readiness")
		}
		return nil
	}
	err = retry(10, time.Microsecond*10, checkReadiness)
	if err != nil {
		suite.FailNow("service is not ready readiness", err.Error())
	}

	//addGlobal documents to mong db
	defaultCustomerConfig := decode[interface{}](suite, defaultCustomerConfigJson)
	if _, err := mongo.GetWriteCollection(consts.CustomerConfigCollection).InsertOne(context.Background(), defaultCustomerConfig); err != nil {
		suite.FailNow("failed to insert defaultCustomerConfigJson", err.Error())
	}
}

func (suite *MainTestSuite) SetupTest() {
	//login with default user
	suite.login(defaultUserGUID)
}

func (suite *MainTestSuite) login(customerGUID string) {
	loginDetails := struct {
		CustomerGUID string `json:"customerGUID"`
	}{
		CustomerGUID: customerGUID,
	}
	w := suite.doRequest(http.MethodPost, "/login", loginDetails)
	if w.Code != http.StatusOK {
		suite.FailNow("failed to login")
	}
	suite.authCookie = w.Header().Get("Set-Cookie")
	suite.authCustomerGUID = customerGUID
}

func (suite *MainTestSuite) loginAsAdmin(customerGUID string) {
	loginDetails := struct {
		CustomerGUID string                 `json:"customerGUID"`
		Attributes   map[string]interface{} `json:"attributes"`
	}{
		CustomerGUID: customerGUID,
		Attributes: map[string]interface{}{
			"admin": true,
		},
	}
	w := suite.doRequest(http.MethodPost, "/login", loginDetails)
	if w.Code != http.StatusOK {
		suite.FailNow("failed to login")
	}
	suite.authCookie = w.Header().Get("Set-Cookie")
	suite.authCustomerGUID = customerGUID
}

func (suite *MainTestSuite) TearDownTest() {
	//drop all collections except customer config
	collections, err := mongo.ListCollectionNames(context.Background())
	if err != nil {
		suite.FailNow("failed to list collections", err.Error())
	}
	for _, collection := range collections {
		if collection != consts.CustomerConfigCollection {
			mongo.GetWriteCollection(collection).Drop(context.Background())
			mongo.IndexCollection(collection)
		}
	}
}
func (suite *MainTestSuite) TearDownSuite() {
	suite.shutdownFunc()
	exec.Command("/bin/sh", "-c", mongoStopCommand).Run()
}

func (suite *MainTestSuite) doRequest(method, path string, body interface{}) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	var req *http.Request
	var reqErr error
	if body != nil {
		bodyBytes, err := json.Marshal(body)
		if err != nil {
			suite.FailNow("failed to marshal body", err.Error())
		}
		bodyReader := bytes.NewReader(bodyBytes)
		req, reqErr = http.NewRequest(method, path, bodyReader)
	} else {
		req, reqErr = http.NewRequest(method, path, nil)
	}
	if reqErr != nil {
		suite.FailNow("failed to create request", reqErr.Error())
	}

	if suite.authCookie != "" {
		req.Header.Set("Cookie", suite.authCookie)
	}
	suite.router.ServeHTTP(w, req)

	return w
}

func loadJson[T types.DocContent](jsonBytes []byte) (docs []T, names []string) {
	if err := json.Unmarshal(jsonBytes, &docs); err != nil {
		panic(err)
	}

	sort.Slice(docs, func(i, j int) bool {
		return docs[i].GetName() < docs[j].GetName()
	})

	names = make([]string, len(docs))
	for i, doc := range docs {
		names[i] = doc.GetName()
	}
	return docs, names
}

func retry(attempt int, delay time.Duration, f func() error) error {
	var err error
	for i := 0; i < attempt; i++ {
		err = f()
		if err == nil {
			return nil
		}
		time.Sleep(delay)
	}
	return err
}

var ignoreTime = cmp.FilterValues(func(x, y time.Time) bool { return true }, cmp.Ignore())
