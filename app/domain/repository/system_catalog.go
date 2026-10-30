package repository

import (
	"time"
	"um/app/core/utils"
	"um/app/domain/model"
	"um/db"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// ISystemCatalog stores one entry per System code (ADR-0012).
type ISystemCatalog interface {
	List() ([]model.SystemCatalogEntry, error)
	Get(systemCode string) (*model.SystemCatalogEntry, error)
	Save(entry model.SystemCatalogEntry) (*model.SystemCatalogEntry, error)
	Delete(systemCode string) error
	// InsertMissing adds entries whose code is not in the catalog yet and
	// leaves existing ones untouched; it returns how many it added.
	InsertMissing(entries []model.SystemCatalogEntry) (int, error)
}

type systemCatalogEntity struct {
	catalog *mongo.Collection
	systems *mongo.Collection
}

func NewSystemCatalogEntity(resource *db.Resource) ISystemCatalog {
	return &systemCatalogEntity{
		catalog: resource.UmDb.Collection("system_catalog"),
		systems: resource.UmDb.Collection("systems"),
	}
}

func (e *systemCatalogEntity) List() ([]model.SystemCatalogEntry, error) {
	ctx, cancel := utils.InitContext()
	defer cancel()
	cursor, err := e.catalog.Find(ctx, bson.M{}, options.Find().SetSort(bson.M{"_id": 1}))
	if err != nil {
		return nil, err
	}
	items := []model.SystemCatalogEntry{}
	if err := cursor.All(ctx, &items); err != nil {
		return nil, err
	}
	return items, nil
}

func (e *systemCatalogEntity) Get(systemCode string) (*model.SystemCatalogEntry, error) {
	ctx, cancel := utils.InitContext()
	defer cancel()
	item := model.SystemCatalogEntry{}
	if err := e.catalog.FindOne(ctx, bson.M{"_id": systemCode}).Decode(&item); err != nil {
		return nil, err
	}
	return &item, nil
}

// Save writes the entry and renames every Client's System of that code, so
// the name a Client sees follows the catalog.
func (e *systemCatalogEntity) Save(entry model.SystemCatalogEntry) (*model.SystemCatalogEntry, error) {
	ctx, cancel := utils.InitContext()
	defer cancel()
	entry.UpdatedDate = time.Now()
	_, err := e.catalog.ReplaceOne(ctx, bson.M{"_id": entry.SystemCode}, entry, options.Replace().SetUpsert(true))
	if err != nil {
		return nil, err
	}
	_, err = e.systems.UpdateMany(ctx, bson.M{"systemCode": entry.SystemCode}, bson.M{"$set": bson.M{"systemName": entry.SystemName}})
	if err != nil {
		return nil, err
	}
	return &entry, nil
}

func (e *systemCatalogEntity) Delete(systemCode string) error {
	ctx, cancel := utils.InitContext()
	defer cancel()
	res, err := e.catalog.DeleteOne(ctx, bson.M{"_id": systemCode})
	if err != nil {
		return err
	}
	if res.DeletedCount == 0 {
		return mongo.ErrNoDocuments
	}
	return nil
}

func (e *systemCatalogEntity) InsertMissing(entries []model.SystemCatalogEntry) (int, error) {
	ctx, cancel := utils.InitContext()
	defer cancel()
	added := 0
	for _, entry := range entries {
		entry.UpdatedDate = time.Now()
		res, err := e.catalog.UpdateOne(ctx, bson.M{"_id": entry.SystemCode}, bson.M{"$setOnInsert": entry}, options.Update().SetUpsert(true))
		if err != nil {
			return added, err
		}
		if res.UpsertedCount > 0 {
			added++
		}
	}
	return added, nil
}
