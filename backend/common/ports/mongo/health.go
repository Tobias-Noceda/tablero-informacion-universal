package mongo

import "context"

// Ping asks the primary to answer, as every write needs it.
func (db *MongoDB) Ping(ctx context.Context) error {
	return db.client.Ping(ctx, nil)
}
