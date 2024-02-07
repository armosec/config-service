package mongo

import (
	"config-service/utils"
	"context"
	"fmt"
	"sync"

	"github.com/hashicorp/go-multierror"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/writeconcern"
	"go.uber.org/zap"
)

var mongoDB, mongoDBprimary *mongo.Database

func MustConnect(config utils.MongoConfig) {
	if err := Connect(config); err != nil {
		zap.L().Fatal("failed to connect to mongo", zap.Error(err))
	}
}

func EnsureConnected() error {
	wg := sync.WaitGroup{}
	var dbPingError error
	var primaryDBPingError error

	zap.L().Info("checking mongo connectivity")
	if mongoDB != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			dbPingError = mongoDB.Client().Ping(context.TODO(), nil)
		}()
	}
	if mongoDBprimary != nil && mongoDBprimary != mongoDB {
		wg.Add(1)
		go func() {
			defer wg.Done()
			primaryDBPingError = mongoDBprimary.Client().Ping(context.TODO(), nil)
		}()
	}
	wg.Wait()
	var err error
	if dbPingError != nil {
		err = multierror.Append(err)
	}
	if primaryDBPingError != nil {
		err = multierror.Append(err)
	}
	if err != nil {
		zap.L().Error("mongo connection failed", zap.Error(err))
	} else {
		zap.L().Info("mongo connection verified")
	}
	return err
}

func Connect(config utils.MongoConfig) error {
	defaultOpts := options.Client().
		SetRetryWrites(true)

	dbOptionsWriteConcern := options.Database().
		SetWriteConcern(writeconcern.New(writeconcern.WMajority()))

	dbClientOpts := defaultOpts
	url := generateMongoUrl(config.Host, config.Port, config.User, config.Password)
	dbClient, err := mongo.Connect(context.TODO(), dbClientOpts.ApplyURI(url), dbClientOpts.SetMinPoolSize(100), dbClientOpts.SetMaxPoolSize(400))
	if err != nil {
		return err
	}

	//check if replicaSet is defined
	primaryUrl := getPrimaryUrl(config)
	if primaryUrl != "" {
		zap.L().Info("connecting to replica set " + config.ReplicaSet)
		if mongoDB = dbClient.Database(config.DB); mongoDB == nil {
			return fmt.Errorf("failed to connect. database: %s /n url: %s", config.DB, url)
		}
		primeOpts := defaultOpts
		if primeClient, err := mongo.Connect(context.TODO(), primeOpts.ApplyURI(primaryUrl), dbClientOpts.SetMinPoolSize(100), dbClientOpts.SetMaxPoolSize(400)); err != nil {
			return err
		} else if mongoDBprimary = primeClient.Database(config.DB, dbOptionsWriteConcern); mongoDBprimary == nil {
			return fmt.Errorf("failed to connect to primary DB. database: %s /n url: %s", config.DB, primaryUrl)
		}
	} else {
		zap.L().Info("connecting to single node " + config.Host)
		mongoDB = dbClient.Database(config.DB, dbOptionsWriteConcern)
		if mongoDB == nil {
			return fmt.Errorf("failed to connect to DB. database: %s /n url: %s", config.DB, url)
		}
		mongoDBprimary = mongoDB
	}

	if err := createIndexes(); err != nil {
		return err
	}

	return EnsureConnected()
}

func Disconnect() {
	if mongoDB != nil {
		mongoDB.Client().Disconnect(context.TODO())
	}
	if mongoDBprimary != nil && mongoDBprimary != mongoDB {
		mongoDBprimary.Client().Disconnect(context.TODO())
	}
}

func GetReadCollection(collectionName string) *mongo.Collection {
	return mongoDB.Collection(collectionName)
}

func GetWriteCollection(collectionName string) *mongo.Collection {
	return mongoDBprimary.Collection(collectionName)
}

func ListCollectionNames(c context.Context) ([]string, error) {
	return mongoDB.ListCollectionNames(c, bson.D{}, options.ListCollections().SetAuthorizedCollections(true).SetNameOnly(true))
}

func generateMongoUrl(host, port, user, password string) (mongoUrl string) {
	if host == "" {
		zap.L().Fatal("mongo host is not defined")
	}
	hostNPort := host
	if port != "" {
		hostNPort = fmt.Sprintf("%s:%s", host, port)
	}
	var userNpass string
	if user != "" && password != "" {
		userNpass = fmt.Sprintf("%s:%s", user, password)
	}

	if userNpass != "" {
		mongoUrl = fmt.Sprintf("mongodb://%s@%s", userNpass, hostNPort)
	} else {
		mongoUrl = fmt.Sprintf("mongodb://%s", hostNPort)
	}
	return mongoUrl
}

func getPrimaryUrl(config utils.MongoConfig) string {
	if config.ReplicaSet != "" {
		replicaSetUrl := fmt.Sprintf("%s/?replicaSet=%s", generateMongoUrl(config.Host, config.Port, config.User, config.Password), config.ReplicaSet)
		client, err := mongo.Connect(context.TODO(), options.Client().ApplyURI(replicaSetUrl))
		if err == nil {
			rsDB := client.Database("admin")
			if rsDB != nil {
				var result bson.M
				if err := rsDB.RunCommand(context.TODO(), bson.D{{Key: "replSetGetStatus", Value: 1}}).Decode(result); err != nil {
					if url := parseUrlFromReplicaSetStatusCommand(config, result); url != "" {
						return url
					}
				} else {
					zap.L().Warn("failed to run replSetGetStatus command", zap.Error(err))
				}
			}
		}
		zap.L().Warn("failed to get primary mongo url from admin DB fallback to generated url", zap.Error(err))
		//fallback to default url with replicaSet name if no primary found
		return replicaSetUrl
	}
	//no replicaSet defined
	return ""
}

func parseUrlFromReplicaSetStatusCommand(config utils.MongoConfig, result primitive.M) string {
	if iMem, ok := result["members"]; ok {
		if members, ok := iMem.([]interface{}); ok {
			for _, iMember := range members {
				if member, ok := iMember.(map[string]interface{}); ok {
					if member["stateStr"] == "PRIMARY" {
						if host, ok := member["name"].(string); ok {
							zap.L().Info("primary mongo host", zap.String("host", host))
							return generateMongoUrl(host, config.Port, config.User, config.Password)
						}
					}
				}
			}
		}
	} else {
		zap.L().Warn("cannot find members in replSetGetStatus result")
	}
	zap.L().Warn("cannot find primary in replSetGetStatus result")
	return ""
}
