package whatsmeow_service

import (
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/evolution-foundation/evolution-go/pkg/config"
	"go.mau.fi/whatsmeow/store/sqlstore"
)

func TestDeviceStoreContainerReusesSharedPostgresContainer(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create sqlmock database: %v", err)
	}
	defer db.Close()

	sharedContainer := sqlstore.NewWithDB(db, "postgres", nil)
	service := &whatsmeowService{
		authDB:        db,
		config:        &config.Config{PostgresAuthDB: "postgres://shared"},
		postgresStore: sharedContainer,
	}
	service.postgresStoreOnce.Do(func() {})

	first, err := service.deviceStoreContainer()
	if err != nil {
		t.Fatalf("first device store container: %v", err)
	}
	second, err := service.deviceStoreContainer()
	if err != nil {
		t.Fatalf("second device store container: %v", err)
	}

	if first != sharedContainer || second != sharedContainer {
		t.Fatalf("expected repeated client store creation to reuse the shared container")
	}
	if got := db.Stats().MaxOpenConnections; got != 0 {
		t.Fatalf("expected test database pool settings to remain unchanged, got max open %d", got)
	}
}
