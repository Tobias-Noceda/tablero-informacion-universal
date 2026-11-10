package mongo

import (
	"context"
	"errors"
	"fmt"
	"log"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// collectionRename moves a collection to the name the code now uses.
type collectionRename struct {
	from, to string
}

// legacyCollections are collections a migration renames: what used to be
// called custom well-knowns are now post-it templates.
var legacyCollections = []collectionRename{
	{from: "custom_well_knowns", to: "postit_templates"},
}

// RenameLegacyCollections moves every legacy collection to its current name.
// It runs before the indexes are ensured, which would otherwise create the
// new collection empty. Safe to repeat: a collection already moved is skipped.
func (db *MongoDB) RenameLegacyCollections() error {
	return db.renameCollections(legacyCollections)
}

// renameCollections applies each rename in this database:
//   - only the old one exists: moved (see moveCollection);
//   - both exist and the old one is empty: the old one is dropped;
//   - both exist and only the new one is empty (e.g. its indexes were created
//     first): the old one replaces it;
//   - both hold documents and the new one already has every old document (a
//     copy cut short before dropping the old one): the old one is dropped;
//   - otherwise both hold different documents: refused, someone has to merge
//     them.
func (db *MongoDB) renameCollections(renames []collectionRename) error {
	ctx, cancel := timeout()
	defer cancel()

	database := db.postitTemplates.Database()

	// Unfiltered on purpose: Atlas' shared tiers only accept an exact name or
	// a regex as a listCollections filter, and there are a handful of them.
	names, err := database.ListCollectionNames(ctx, bson.D{})
	if err != nil {
		return fmt.Errorf("list collections: %w", err)
	}
	exists := map[string]bool{}
	for _, name := range names {
		exists[name] = true
	}

	for _, rename := range renames {
		if !exists[rename.from] {
			continue
		}

		replaceTarget := false
		if exists[rename.to] {
			from, to := database.Collection(rename.from), database.Collection(rename.to)
			fromCount, err := from.CountDocuments(ctx, bson.D{})
			if err != nil {
				return err
			}
			toCount, err := to.CountDocuments(ctx, bson.D{})
			if err != nil {
				return err
			}

			switch {
			case fromCount == 0:
				if err := from.Drop(ctx); err != nil {
					return fmt.Errorf("drop empty %s: %w", rename.from, err)
				}
				continue
			case toCount == 0:
				replaceTarget = true
			default:
				copied, err := holdsAll(ctx, to, from)
				if err != nil {
					return err
				}
				if !copied {
					return fmt.Errorf("collections %s and %s both hold different documents: merge them by hand", rename.from, rename.to)
				}
				if err := from.Drop(ctx); err != nil {
					return fmt.Errorf("drop already copied %s: %w", rename.from, err)
				}
				continue
			}
		}

		if err := db.moveCollection(ctx, database, rename, replaceTarget); err != nil {
			return err
		}
	}

	return nil
}

// moveCollection renames from to to, atomically and keeping its indexes.
// Where the server refuses renameCollection (some managed tiers do), it copies
// the documents with $out, which replaces to, and drops from; the indexes are
// recreated by EnsureIndexes, which runs next.
func (db *MongoDB) moveCollection(ctx context.Context, database *mongo.Database, rename collectionRename, replaceTarget bool) error {
	command := bson.D{
		{Key: "renameCollection", Value: database.Name() + "." + rename.from},
		{Key: "to", Value: database.Name() + "." + rename.to},
		{Key: "dropTarget", Value: replaceTarget},
	}
	err := db.client.Database("admin").RunCommand(ctx, command).Err()
	if err == nil {
		return nil
	}

	var refused mongo.CommandError
	if !errors.As(err, &refused) {
		return fmt.Errorf("rename %s to %s: %w", rename.from, rename.to, err)
	}
	log.Printf("renameCollection %s to %s refused (%v): copying instead", rename.from, rename.to, err)

	return copyCollection(ctx, database, rename)
}

// copyCollection copies every document of from into to (replacing it), checks
// the copy is complete, then drops from. Cut short after the copy, the next
// run finds to holding everything and drops from (see renameCollections).
func copyCollection(ctx context.Context, database *mongo.Database, rename collectionRename) error {
	from, to := database.Collection(rename.from), database.Collection(rename.to)

	cursor, err := from.Aggregate(ctx, mongo.Pipeline{{{Key: "$out", Value: rename.to}}})
	if err != nil {
		return fmt.Errorf("copy %s to %s: %w", rename.from, rename.to, err)
	}
	if err := cursor.Close(ctx); err != nil {
		return err
	}

	copied, err := holdsAll(ctx, to, from)
	if err != nil {
		return err
	}
	if !copied {
		return fmt.Errorf("copy %s to %s is incomplete: %s was kept", rename.from, rename.to, rename.from)
	}
	if err := from.Drop(ctx); err != nil {
		return fmt.Errorf("drop %s after copying it: %w", rename.from, err)
	}
	return nil
}

// holdsAll reports whether target has a document with every _id in source.
func holdsAll(ctx context.Context, target, source *mongo.Collection) (bool, error) {
	cursor, err := source.Find(ctx, bson.D{}, options.Find().SetProjection(bson.D{{Key: "_id", Value: 1}}))
	if err != nil {
		return false, err
	}
	var docs []struct {
		ID any `bson:"_id"`
	}
	if err := cursor.All(ctx, &docs); err != nil {
		return false, err
	}
	if len(docs) == 0 {
		return true, nil
	}

	ids := make([]any, len(docs))
	for i, doc := range docs {
		ids[i] = doc.ID
	}
	found, err := target.CountDocuments(ctx, bson.D{{Key: "_id", Value: bson.D{{Key: "$in", Value: ids}}}})
	if err != nil {
		return false, err
	}
	return found == int64(len(ids)), nil
}
