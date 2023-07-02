package mongo

import (
	"config-service/utils/consts"
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.uber.org/zap"
)

// collectionIndexes is a map of collection name to index models for collections that need custom indexes
// if a collection is not in this map, it will use the default index
var collectionIndexes = map[string][]mongo.IndexModel{
	consts.CustomersCollection: {
		//no need to index customers collection
	},
}

// defaultIndex is the default index for all collections unless overridden in collectionIndexes
var defaultIndex = []mongo.IndexModel{
	{
		Keys: bson.D{
			{Key: "name", Value: 1},
		},
	},
	{
		Keys: bson.D{
			{Key: "customers", Value: 1},
		},
	},
}

func createIndexes() error {
	zap.L().Info("creating indexes on mongo")
	collections, err := ListCollectionNames(context.Background())
	if err != nil {
		return err
	}
	for _, collection := range collections {
		if indexModels, ok := collectionIndexes[collection]; ok {
			// if collection has custom indexes, create them
			zap.L().Info("creating custom indexes", zap.String("collection", collection), zap.Any("indexes", indexModels))
			res, err := GetReadCollection(collection).Indexes().CreateMany(context.Background(), indexModels)
			if err != nil {
				zap.L().Error("failed to create custom indexes", zap.Error(err), zap.Any("result", res), zap.String("collection", collection))
				return err
			}
			zap.L().Info("created custom indexes", zap.String("collection", collection), zap.Any("result", res))
		} else {
			// otherwise, create the default indexes
			zap.L().Info("creating default indexes", zap.String("collection", collection), zap.Any("indexes", defaultIndex))
			res, err := GetReadCollection(collection).Indexes().CreateMany(context.Background(), defaultIndex)
			if err != nil {
				zap.L().Error("failed to create default indexes", zap.Error(err), zap.Any("result", res), zap.String("collection", collection))
				return err
			}
			zap.L().Info("created default indexes", zap.String("collection", collection), zap.Any("result", res))
		}
	}
	return nil
}
