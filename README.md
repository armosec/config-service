# config-service


1. [Overview](#overview)
2. [Document types](#document-types)
3. [Handlers package](#handlers-package)
4. [DB package](#handlers-package)
5. [Adding a new document type handler](#adding-a-new-document-type-handler)
    1. [Todo List](#todo-list)
    2. [Using the generic handlers](#using-the-generic-handlers)
    3. [Router options](#router-options)
    4. [Customized behavior](#customized-behavior)
6. [Log & trace](#log--trace)
7. [Testing](#testing)
8. [Running](#running)
9. [Configuration](#configuration)




## Overview

 The `config-service` is a CRUD service of configuration data for kubescape.

It uses [gin web framework](https://github.com/gin-gonic/gin) for `http` handling and  [mongo](https://github.com/mongodb/mongo-go-driver) as the service database.

The config service provides a ```db``` package for common database `CRUD` operations and a ```handlers``` package for common `http` handling.


![Packages](docs/overview.drawio.svg)

## Document types
The service serves documents of [DocContent](types/types.go) type.

All served document types need to be part of the [DocContent](types/types.go) types constraint and implement the [DocContent](types/types.go) interface.
```go
type DocContent interface {
	*MyType | *CustomerConfig | *Cluster | *PostureExceptionPolicy ...
    InitNew()
	GetReadOnlyFields() []string
```
Document types also need [bson](https://www.mongodb.com/docs/drivers/go/current/usage-examples/struct-tagging/) tags for the fields that are stored in the database.
```go
type PortalCluster struct {
	PortalBase       `json:",inline" bson:"inline"`
	SubscriptionDate string `json:"subscription_date" bson:"subscription_date"`
	LastLoginDate    string `json:"last_login_date" bson:"last_login_date"`
}
```


![Document Types](docs/types.drawio.svg)

Functions in the `db` and `handlers` packages are using [DocContent](types/types.go) type parameter.
```go
clusters := []*types.Cluster{}
frameworks := []*types.Framework{}
//method returns array of specified type 
clusters, err = db.GetAllForCustomer[*types.Cluster](c)
frameworks, err = db.GetAllForCustomer[*types.Framework](c)
```


## Handlers package
*Note: as described in the [Using the generic handlers](#using-the-generic-handlers) section, most endpoints will use the generic handlers by configuring routes options and therefor will not need to use the `handlers` package functions directly.*

The `handlers` package provides:
1. [gin handlers](handlers/handlers.go) for handing requests with common `CRUD` operations.
 The name convention of a request handler is `Handle<method><operation>` e.g. `HandleGetAll`.
2. [Handlers helpers](handlers/handlers.go) for handling different parts of the request lifecycle.These functions are the building blocks of the request handlers and can also be reused when implementing customized handlers. The naming convention for the handlers helpers is `<method><operation>Handler` e.g. `PostDocHandler` or  `GetByNameParamHandler`.
3. Common [middleware](handlers/middleware.go) functions. The middleware name convention is  `<method><operation>Middleware` e.g. `PostValidationMiddleware`.
4. Handlers for common [responses](handlers/response.go).
5. Predefined [mutators-validators](handlers/validate.go) to customized Put and Post validation and/or initialize or set required data.
6. [Routes configuration](handlers/routes.go) to easily use all the above as described in [Using the generic handlers](#using-the-generic-handlers) section.



The functions in the `handlers` package use data stored in the gin context by other middlewares.
For instance `CustomerGUID` is set by the authentication middleware, `RequestLogger` is set by the logger middleware, `dbCollection` is set by the db middleware and so on.
For full list of context keys see [const.go](utils/consts/const.go).

## DB package

The db package provides:
1. Common database [CRUD functions](db/utils.go)
2. [Query filter builder](db/filter.go)
3. [Projection builder](db/projection.go)
4. [Update command generator](db/update.go)
5. [Cache](db/cached_doc.go) for rarely updated and frequently read documents.

*Note: Most endpoints will not need to use the `db` package directly.
Most handlers will be able to implement even customized behavior using just the `handlers` package functions.*

## Adding a new document type handler
- ### Todo List
1. Add the type to [DocContent](types/types.go) types constraint and implement [DocContent](types/types.go) methods.
2. Add `bson` tags to the new type fields.
3. Add the strings of the new type path and DB collection to [const.go](utils/consts/const.go).
4. Add a folder under the `routes` folder for the new type and a file with ```func AddRoutes(g *gin.Engine) ``` function for setting up the `http` handlers for the new type.
5. call `myType.AddRoutes` function from [main.go](main.go) after the authentication middleware.
6. Add e2e [tests](#testing) the new type endpoint.

### Using the generic handlers
Endpoint handlers can configure the desired handling behavior by setting [routes options](handlers/routes.go) and calling the `handlers.AddRoutes` function.


```go
package "myType"
import (
	"config-service/handlers"
	"config-service/types"
	"config-service/utils/consts"

	"github.com/gin-gonic/gin"
)
//Add routes for serving myType 
function AddRoutes(g *gin.Engine) {
    //create a new router options builder for myType with default options 
    routerOptions := handlers.NewRouterOptionsBuilder[*types.MyType].
    WithPath("/myType").
    WithDBCollection(consts.MyTypeCollection).
    //add get by name e.g. GET /myType?typeName="x"
    WithNameQuery("typeName").
    //disable get names list e.g. GET /myType?list
    WithGetNamesList(false)
    //use handlers AddRoutes with the options to build the http handlers 
    handlers.AddRoutes(g, routerOptions.Get()...)
}
```
#### [Router options](handlers/routes.go)
| Method/Action | Description | Option setting example | Default |
| ------------- | ----------- | -------- | -------- | 
|GET all  | get all user's documents | routerOptions.WithServeGet(true) | On |
|GET list of names  | get list of documents names if "list" query param is set (e.g. GET /myType?list) |  routerOptions.WithGetNamesList(true) | On
|GET all with global  | get all user's and global (without an owner) documents | routerOptions.WithIncludeGlobalDocs(true) | Off |
|GET by name  | get a document by name using query param (e.g. GET /myType?typeName="x") |  routerOptions.WithNameQuery("typeName") | Off
|GET by query  | get a document by query params according to given [query config](handlers/scopequery.go) (e.g. GET /myType?scope.cluster="nginx") |  routerOptions.WithQueryConfig(&queryConfig) | Off |
|POST with guid in path or body | create a new document, the post operation can be configured with additional customized or predefined [validators](handlers/validate.go) like unique name, unique short name attribute   |  routerOptions.WithServePost(true).WithValidatePostUniqueName(true).WithPostValidator(myValidator) | On with unique name validator
|PUT  | update a document or a list of documents, the put operation can be configured with additional customized or predefined [mutators/validators](handlers/validate.go) like GUID existence in body or path  |  routerOptions.WithServePut(true).WithValidatePutGUID(true).WithPutValidator(myValidator) | On with guid existence validator
|DELETE with guid in path | delete a document   |  routerOptions.WithServeDelete(true) | On
|DELETE by name  | delete a document or a list of documents by name   |  routerOptions.WithDeleteByName(true) | Off

### Customized behavior
Endpoints that need to implement customized behavior for some routes can still use `handlers.AddRoutes ` for the rest of the routes, see [customer configuration endpoint](routes/v1/customer_config/routes.go) for example.

If an endpoint does not use any of the common handlers it needs to use other helper functions from the `handlers` package and/or function from the `db`, see [customer endpoint](routes/v1/customer/routes.go) for example.


## Log & trace 
Each in-coming request is logged by the `RequestSummary` middleware, the log format is: 
```json
{"level":"info","ts":"2022-12-20T19:46:08.809161523+02:00","msg":"/v1_vulnerability_exception_policy","status":200,"method":"DELETE","path":"/v1_vulnerability_exception_policy","query":"policyName=1660467597.8207463&policyName=1660475024.9930612","ip":"","user-agent":"","latency":0.001204906,"time":"2022-12-20T17:46:08Z","customerGUID":"test-customer-guid","trace_id":"14793d67ea475427a8881f8aebee9c18","span_id":"e5d98f9f362690b4"}
```


In addition other middleware also set in the gin.Context of each request a new logger with the request data and an OpenTelematry tracer. 
Handlers should use [log](utils/log/logtrace..go) functions for specific logging 

Code

```go
import "config-service/utils/log"
 func myHandler(c *gin.Context) {
    ...
    log.LogNTrace("hello world", c)
```
Log
```json
{"level":"info","ts":"2022-12-20T20:11:17.180274896+02:00","msg":"hello world","method":"GET","query":"posturePolicies.controlName=Allowed hostPath&posturePolicies.controlName=Applications credentials in configuration files","path":"/v1_posture_exception_policy","trace_id":"afa45fe66bd47fb7592a76c0fc4c3715","span_id":"98e47f28bc3503a6"}
```

For tracing times of heavy time consumers functions use:
```go 
func deleteAll(c gin.Context) {
    defer log.LogNTraceEnterExit("deleteAll", c)()
    ....
```
log on entry
```json
{"level":"info","ts":"2022-12-20T20:14:05.518747309+02:00","msg":"deleteAll","method":"DELETE","query":"","path":"/v1_myType","trace_id":"71e0cf6b3d355a0733e42c514c9a7772","span_id":"ff51efe3cdf366fd"}
```
log on exit
```json
{"level":"info","ts":"2022-12-20T20:30:05.518747309+02:00","msg":"deleteAll completed","method":"DELETE","query":"","path":"/v1_myType","trace_id":"71e0cf6b3d355a0733e42c514c9a7772","span_id":"ff51efe3cdf366fd"}
```

## Testing
The service main test defines a [testify suite](suite_test.go) that runs a mongo container and the config service for end to end testing.

Endpoints use the common handlers can also reuse the [common tests functions](testers_test.go) to test the endpoint behavior.

#### Coverage
At the top of the [suite](suite_test.go) file there is a comment with command line needed to run the tests and generate a coverage report, please make sure your code is covered by the tests before submitting a PR.

For details see the existing [endpoint tests](service_test.go) 


## Running 
### Running  the tests
```bash
# run the tests
go test ./...
```
### Running the tests with coverage

```bash
#run the tests and generate a coverage report 
#TODO add new packages to the coverpkg list if needed
go test -timeout 30s  -coverpkg=./handlers,./db,./types,./routes/prob,./routes/login,./routes/v1/cluster,./routes/v1/posture_exception,./routes/v1/vulnerability_exception,./routes/v1/customer,./routes/v1/customer_config,./routes/v1/repository,./routes/v1/registry_cron_job,./routes/v1/admin -coverprofile coverage.out  
...
...
PASS
coverage: 75.3% of statements in ./handlers, ./db, ./types, ./routes/prob, ./routes/login, ./routes/v1/cluster, ./routes/v1/posture_exception, ./routes/v1/vulnerability_exception, ./routes/v1/customer, ./routes/v1/customer_config, ./routes/v1/repository,./routes/v1/admin
ok      config-service  7.170s


#convert the coverage report to html and open it in the browser
go tool cover -html=coverage.out -o coverage.html \
&& open coverage.html
```
### Running the service locally
To run the service locally you need first to run a mongo instance.
```bash
docker run --name=mongo -d -p 27017:27017 -e "MONGO_INITDB_ROOT_USERNAME=admin" -e "MONGO_INITDB_ROOT_PASSWORD=admin" mongo 
```
For debug purposes you can also run a mongo-express instance to view the data in the mongo instance.
```bash
docker network create mongo
docker run --name=mongo -d -p 27017:27017 --network mongo -e "MONGO_INITDB_ROOT_USERNAME=admin" -e "MONGO_INITDB_ROOT_PASSWORD=admin" mongo 
docker run --name=mongo-express -d -p 8081:8081 --network mongo -e "ME_MONGO_INITDB_ROOT_USERNAME=admin" -e "ME_MONGO_INITDB_ROOT_PASSWORD=admin" -e "ME_CONFIG_MONGODB_URL=mongodb://admin:admin@mongo:27017/" mongo-express
```
Then you can run the service.
```bash
go run .
{"level":"info","ts":"2022-12-21T15:59:17.579524706+02:00","msg":"connecting to single node localhost"}
{"level":"info","ts":"2022-12-21T15:59:17.579589138+02:00","msg":"checking mongo connectivity"}
{"level":"info","ts":"2022-12-21T15:59:17.594646374+02:00","msg":"mongo connection verified"}
{"level":"info","ts":"2022-12-21T15:59:17.594796442+02:00","msg":"Starting server on port 8080"}
```

Sure, here's a sample `README.md` section detailing each part of your `config.json` file:

## Configuration

The application's settings are usually stored in a `config.json` file, located in the root directory. However, you can specify a different location for the configuration file using the `CONFIG_PATH` environment variable.

Here's a brief description of each setting:

```json
{
    "port": 8080,
    "mongo": {
        "host": "localhost",
        "port": 27017,
        "db": "dbname",
        "user": "username",
        "password": "password",
        "replicaSet": ""
    },
    "logger": {
        "level": "debug"
    },
    "telemetry": {
        "jaegerAgentHost": "localhost",
        "jaegerAgentPort": "32033"
    }
}
```

### Settings Description

- `port` : The port number on which the service runs.

- `mongo` : MongoDB connection settings:
    - `host` : The hostname or IP address of the MongoDB server.
    - `port` : The port number on which the MongoDB server is listening.
    - `db` : The name of the database to use in MongoDB.
    - `user` : The username for MongoDB authentication. Leave it as an empty string if authentication is not enabled.
    - `password` : The password for MongoDB authentication. Leave it as an empty string if authentication is not enabled.
    - `replicaSet` : The name of the MongoDB replica set. Leave it as an empty string if a replica set is not used.

- `logger` : Logger settings:
    - `level` : The level of logs to be emitted by the service. Can be set to "debug", "info", "warn", "error", etc.

- `telemetry` : Telemetry settings:
    - `jaegerAgentHost` : The hostname or IP address of the Jaeger agent for tracing.
    - `jaegerAgentPort` : The port number on which the Jaeger agent is listening.


### Configuring with `config.json`

By default, the service reads its settings from `config.json` in the root directory.

### Configuring with `CONFIG_PATH`

You can specify a different location for the configuration file using the `CONFIG_PATH` environment variable. This is useful if you want to keep configuration files in a separate directory, or if you have different configuration files for different environments.

To use `CONFIG_PATH`, set it to the path of your configuration file. For example:

```bash
export CONFIG_PATH=/path/to/your/config.json
```

_Note_: Sensitive data like usernames, passwords or any kind of secrets should not be stored directly in the `config.json` file for security reasons. Consider using environment variables or secure secret management systems for such data.

### Overriding settings using Environment Variables

Some settings can be overridden using environment variables. 
This is useful if you want to change a specific setting without modifying the `config.json` file, or if you want to provide sensitive data like passwords.

Currently, the service supports the following environment variables:

- `MONGODB_USER` : Overrides the MongoDB username.
- `MONGODB_PASSWORD` : Overrides the MongoDB password.

For example:

```bash
export MONGODB_USER=myuser
export MONGODB_PASSWORD=mypassword
```
