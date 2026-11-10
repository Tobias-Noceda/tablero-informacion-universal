package mongo

import (
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"
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
//   - only the old one exists: renamed (atomic, indexes kept);
//   - both exist and the old one is empty: the old one is dropped;
//   - both exist and only the new one is empty (e.g. its indexes were created
//     first): the old one replaces it;
//   - both hold documents: refused, someone has to merge them.
func (db *MongoDB) renameCollections(renames []collectionRename) error {
	ctx, cancel := timeout()
	defer cancel()

	database := db.postitTemplates.Database()
	for _, rename := range renames {
		names, err := database.ListCollectionNames(ctx, bson.M{"name": bson.M{"$in": []string{rename.from, rename.to}}})
		if err != nil {
			return fmt.Errorf("list collections: %w", err)
		}
		exists := map[string]bool{}
		for _, name := range names {
			exists[name] = true
		}
		if !exists[rename.from] {
			continue
		}

		dropTarget := false
		if exists[rename.to] {
			fromCount, err := database.Collection(rename.from).CountDocuments(ctx, bson.M{})
			if err != nil {
				return err
			}
			if fromCount == 0 {
				if err := database.Collection(rename.from).Drop(ctx); err != nil {
					return fmt.Errorf("drop empty %s: %w", rename.from, err)
				}
				continue
			}

			toCount, err := database.Collection(rename.to).CountDocuments(ctx, bson.M{})
			if err != nil {
				return err
			}
			if toCount > 0 {
				return fmt.Errorf("collections %s and %s both hold documents: merge them by hand", rename.from, rename.to)
			}
			dropTarget = true
		}

		command := bson.D{
			{Key: "renameCollection", Value: database.Name() + "." + rename.from},
			{Key: "to", Value: database.Name() + "." + rename.to},
			{Key: "dropTarget", Value: dropTarget},
		}
		if err := db.client.Database("admin").RunCommand(ctx, command).Err(); err != nil {
			return fmt.Errorf("rename %s to %s: %w", rename.from, rename.to, err)
		}
	}

	return nil
}
