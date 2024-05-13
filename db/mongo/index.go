package mongo

import (
	"config-service/utils/consts"
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"
)

// collectionIndexes is a map of collection name to index models for collections that need custom indexes
// if a collection is not in this map, it will use the default index
var collectionIndexes = map[string][]mongo.IndexModel{
	consts.CustomersCollection: {
		{
			Keys: bson.D{
				{Key: "guid", Value: 1},
			},
		},
	},
	consts.UsersNotificationsCacheCollection: {
		{
			Keys: bson.D{
				{Key: "guid", Value: 1},
			},
		},
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
		{
			Keys: bson.D{
				{Key: "dataType", Value: 1},
			},
		},
		{
			Keys: bson.D{
				{Key: "expiryTime", Value: 1},
			},
			Options: options.Index().SetExpireAfterSeconds(0),
		},
	},
	consts.AttackChainsCollection: {
		{
			Keys: bson.D{
				{Key: "guid", Value: 1},
			},
		},
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
		{
			Keys: bson.D{
				{Key: "clusterName", Value: 1},
			},
		},
		{
			Keys: bson.D{
				{Key: "lastPostureScanTriggered", Value: 1},
			},
		},
		{
			Keys: bson.D{
				{Key: "lastTimeEngineCompleted", Value: 1},
			},
		},
	},
	consts.UsersNotificationsVulnerabilitiesCollection: {
		{
			Keys: bson.D{
				{Key: "cveID", Value: 1},
			},
		},
		{
			Keys: bson.D{
				{Key: "cluster", Value: 1},
			},
		},
		{
			Keys: bson.D{
				{Key: "namespace", Value: 1},
			},
		},
		{
			Keys: bson.D{
				{Key: "notificationType", Value: 1},
			},
		},
		{
			Keys: bson.D{
				{Key: "customers", Value: 1},
			},
		},
	},
	consts.CollaborationConfigCollection: {
		{
			Keys: bson.D{
				{Key: "guid", Value: 1},
			},
		},
		{
			Keys: bson.D{
				{Key: "name", Value: 1},
			},
		},
		{
			Keys: bson.D{
				{Key: "provider", Value: 1},
			},
		},
		{
			Keys: bson.D{
				{Key: "customers", Value: 1},
			},
		},
	},
	consts.IntegrationReferenceCollection: {
		{
			Keys: bson.D{
				{Key: "guid", Value: 1},
			},
		},
		{
			Keys: bson.D{
				{Key: "provider", Value: 1},
			},
		},
		{
			Keys: bson.D{
				{Key: "type", Value: 1},
			},
		},
		{
			Keys: bson.D{
				{Key: "customers", Value: 1},
			},
		},
		{
			Keys: bson.D{
				{Key: "owner.name", Value: 1},
			},
		},
		{
			Keys: bson.D{
				{Key: "owner.kind", Value: 1},
			},
		},
		{
			Keys: bson.D{
				{Key: "owner.namespace", Value: 1},
			},
		},
		{
			Keys: bson.D{
				{Key: "owner.cluster", Value: 1},
			},
		},
		{
			Keys: bson.D{
				{Key: "owner.repoHash", Value: 1},
			},
		},
		{
			Keys: bson.D{
				{Key: "owner.resourceID", Value: 1},
			},
		},
		{
			Keys: bson.D{
				{Key: "owner.resourceHash", Value: 1},
			},
		},
		{
			Keys: bson.D{
				{Key: "relatedObjects.layerHash", Value: 1},
			},
		},
		{
			Keys: bson.D{
				{Key: "relatedObjects.controlID", Value: 1},
			},
		},
		{
			Keys: bson.D{
				{Key: "relatedObjects.baseScore", Value: 1},
			},
		},
		{
			Keys: bson.D{
				{Key: "relatedObjects.cveName", Value: 1},
			},
		},
		{
			Keys: bson.D{
				{Key: "relatedObjects.cveID", Value: 1},
			},
		},
		{
			Keys: bson.D{
				{Key: "relatedObjects.severity", Value: 1},
			},
		},
		{
			Keys: bson.D{
				{Key: "relatedObjects.severityScore", Value: 1},
			},
		},
		{
			Keys: bson.D{
				{Key: "relatedObjects.component", Value: 1},
			},
		},
		{
			Keys: bson.D{
				{Key: "relatedObjects.componentVersion", Value: 1},
			},
		},
		{
			Keys: bson.D{
				{Key: "relatedObjects.imageRepository", Value: 1},
			},
		},
	},
	consts.RuntimeIncidentCollection: {
		{
			Keys: bson.D{
				{Key: "guid", Value: 1},
			},
		},
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
		{
			Keys: bson.D{
				{Key: "relatedAlerts.ruleID", Value: 1},
			},
		},
		{
			Keys: bson.D{
				{Key: "relatedAlerts.timestamp", Value: 1},
			},
		},
		{
			Keys: bson.D{
				{Key: "guid", Value: 1},
				// {Key: "customers", Value: 1},
				{Key: "relatedAlerts.timestamp", Value: 1},
			},
		},
		{
			Keys: bson.D{
				{Key: "customers", Value: 1},
				{Key: "incidentSeverity", Value: 1},
			},
		},
		{
			Keys: bson.D{
				{Key: "isDismissed", Value: 1},
			},
		},
		{
			Keys: bson.D{
				{Key: "creationTimestamp", Value: -1},
			},
		},
		{
			Keys: bson.D{
				{Key: "customers", Value: 1},
				{Key: "workloadKind", Value: 1},
				{Key: "workloadName", Value: 1},
			},
		},
		{
			Keys: bson.D{
				{Key: "customers", Value: 1},
				{Key: "isDismissed", Value: 1},
			},
		},
	},
}

// defaultIndex is the default index for all collections unless overridden in collectionIndexes
var defaultIndex = []mongo.IndexModel{
	{
		Keys: bson.D{
			{Key: "guid", Value: 1},
		},
	},
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
		if err := IndexCollection(collection); err != nil {
			return err
		}
	}
	return nil
}

func IndexCollection(collectionName string) error {
	if indexModels, ok := collectionIndexes[collectionName]; ok {
		if indexModels == nil {
			return nil
		}
		// if collection has custom indexes, create them
		zap.L().Info("creating custom indexes", zap.String("collection", collectionName), zap.Any("indexes", indexModels))
		res, err := GetReadCollection(collectionName).Indexes().CreateMany(context.Background(), indexModels)
		if err != nil {
			zap.L().Error("failed to create custom indexes", zap.Error(err), zap.Any("result", res), zap.String("collection", collectionName))
			return err
		}
		zap.L().Info("created custom indexes", zap.String("collection", collectionName), zap.Any("result", res))
	} else {
		// otherwise, create the default indexes
		zap.L().Info("creating default indexes", zap.String("collection", collectionName), zap.Any("indexes", defaultIndex))
		res, err := GetReadCollection(collectionName).Indexes().CreateMany(context.Background(), defaultIndex)
		if err != nil {
			zap.L().Error("failed to create default indexes", zap.Error(err), zap.Any("result", res), zap.String("collection", collectionName))
			return err
		}
		zap.L().Info("created default indexes", zap.String("collection", collectionName), zap.Any("result", res))
	}
	return nil
}
