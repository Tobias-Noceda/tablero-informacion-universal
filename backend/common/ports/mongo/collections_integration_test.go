//go:build integration

package mongo

import (
	"strings"
	"testing"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/v2/bson"
)

func TestMongo_RenameCollections(t *testing.T) {
	db := integrationDB(t)
	ctx, cancel := timeout()
	defer cancel()

	database := db.postitTemplates.Database()
	suffix := strings.ReplaceAll(uuid.NewString(), "-", "")
	from, to := "it_old_"+suffix, "it_new_"+suffix
	rename := []collectionRename{{from: from, to: to}}
	t.Cleanup(func() {
		ctx, cancel := timeout()
		defer cancel()
		_ = database.Collection(from).Drop(ctx)
		_ = database.Collection(to).Drop(ctx)
	})

	count := func(name string) int64 {
		t.Helper()
		n, err := database.Collection(name).CountDocuments(ctx, bson.M{})
		if err != nil {
			t.Fatal(err)
		}
		return n
	}
	exists := func(name string) bool {
		t.Helper()
		names, err := database.ListCollectionNames(ctx, bson.M{"name": name})
		if err != nil {
			t.Fatal(err)
		}
		return len(names) == 1
	}

	// Nothing to move: a no-op.
	if err := db.renameCollections(rename); err != nil {
		t.Fatalf("no legacy collection: %v", err)
	}

	// The old one alone is renamed, documents included.
	if _, err := database.Collection(from).InsertOne(ctx, bson.M{"_id": "a"}); err != nil {
		t.Fatal(err)
	}
	if err := db.renameCollections(rename); err != nil {
		t.Fatalf("rename: %v", err)
	}
	if exists(from) || count(to) != 1 {
		t.Errorf("after rename: old exists=%v, new holds %d, want gone and 1", exists(from), count(to))
	}

	// Repeating it changes nothing.
	if err := db.renameCollections(rename); err != nil || count(to) != 1 {
		t.Errorf("second run: err=%v, new holds %d", err, count(to))
	}

	// An empty old one next to the new one is dropped.
	if err := database.CreateCollection(ctx, from); err != nil {
		t.Fatal(err)
	}
	if err := db.renameCollections(rename); err != nil || exists(from) || count(to) != 1 {
		t.Errorf("empty old: err=%v, old exists=%v, new holds %d", err, exists(from), count(to))
	}

	// Both holding different documents is left for a person to merge.
	if _, err := database.Collection(from).InsertOne(ctx, bson.M{"_id": "b"}); err != nil {
		t.Fatal(err)
	}
	if err := db.renameCollections(rename); err == nil {
		t.Error("two collections with different documents were merged silently")
	}

	// An empty new one (only its indexes) is replaced by the old one.
	if _, err := database.Collection(to).DeleteMany(ctx, bson.M{}); err != nil {
		t.Fatal(err)
	}
	if err := db.renameCollections(rename); err != nil {
		t.Fatalf("empty new: %v", err)
	}
	if exists(from) || count(to) != 1 {
		t.Errorf("empty new: old exists=%v, new holds %d, want gone and 1", exists(from), count(to))
	}

	// A copy cut short before dropping the old one: the new one already holds
	// every old document, so the old one goes.
	if _, err := database.Collection(from).InsertOne(ctx, bson.M{"_id": "b"}); err != nil {
		t.Fatal(err)
	}
	if err := db.renameCollections(rename); err != nil {
		t.Fatalf("copy cut short: %v", err)
	}
	if exists(from) || count(to) != 1 {
		t.Errorf("copy cut short: old exists=%v, new holds %d, want gone and 1", exists(from), count(to))
	}
}

// The fallback used where renameCollection is refused (Atlas shared tiers).
// A local Mongo accepts the rename, so it is called directly.
func TestMongo_CopyCollection(t *testing.T) {
	db := integrationDB(t)
	ctx, cancel := timeout()
	defer cancel()

	database := db.postitTemplates.Database()
	suffix := strings.ReplaceAll(uuid.NewString(), "-", "")
	rename := collectionRename{from: "it_copy_old_" + suffix, to: "it_copy_new_" + suffix}
	t.Cleanup(func() {
		ctx, cancel := timeout()
		defer cancel()
		_ = database.Collection(rename.from).Drop(ctx)
		_ = database.Collection(rename.to).Drop(ctx)
	})

	docs := []any{bson.M{"_id": "a", "name": "A"}, bson.M{"_id": "b", "name": "B"}}
	if _, err := database.Collection(rename.from).InsertMany(ctx, docs); err != nil {
		t.Fatal(err)
	}
	// An existing empty target (e.g. created by its indexes) is replaced.
	if err := database.CreateCollection(ctx, rename.to); err != nil {
		t.Fatal(err)
	}

	if err := copyCollection(ctx, database, rename); err != nil {
		t.Fatalf("copy: %v", err)
	}

	names, err := database.ListCollectionNames(ctx, bson.M{"name": rename.from})
	if err != nil {
		t.Fatal(err)
	}
	if len(names) != 0 {
		t.Error("the old collection was kept after a complete copy")
	}
	var got struct {
		Name string `bson:"name"`
	}
	if err := database.Collection(rename.to).FindOne(ctx, bson.M{"_id": "b"}).Decode(&got); err != nil || got.Name != "B" {
		t.Errorf("copied b = %+v (%v), want its fields", got, err)
	}
	if n, err := database.Collection(rename.to).CountDocuments(ctx, bson.M{}); err != nil || n != 2 {
		t.Errorf("new collection holds %d (%v), want 2", n, err)
	}
}
