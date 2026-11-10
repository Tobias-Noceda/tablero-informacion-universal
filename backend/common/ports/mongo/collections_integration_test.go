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

	// Both holding documents is left for a person to merge.
	if _, err := database.Collection(from).InsertOne(ctx, bson.M{"_id": "b"}); err != nil {
		t.Fatal(err)
	}
	if err := db.renameCollections(rename); err == nil {
		t.Error("two non-empty collections were merged silently")
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
}
