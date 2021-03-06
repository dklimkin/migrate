package mysql

import (
	"database/sql"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/gemnasium/migrate/file"
	"github.com/gemnasium/migrate/migrate/direction"
	pipep "github.com/gemnasium/migrate/pipe"
)

// TestMigrate runs some additional tests on Migrate().
// Basic testing is already done in migrate/migrate_test.go
func TestMigrate(t *testing.T) {
	host := os.Getenv("MYSQL_PORT_3306_TCP_ADDR")
	port := os.Getenv("MYSQL_PORT_3306_TCP_PORT")
	driverURL := "mysql://root@tcp(" + host + ":" + port + ")/migratetest"

	// prepare clean database
	connection, err := sql.Open("mysql", strings.SplitN(driverURL, "mysql://", 2)[1])
	if err != nil {
		t.Fatal(err)
	}

	dropTestTables(t, connection)

	migrate(t, driverURL)

	dropTestTables(t, connection)

	// Make an old-style 32-bit int version column that we'll have to upgrade.
	_, err = connection.Exec("CREATE TABLE IF NOT EXISTS " + tableName + " (version int not null primary key);")
	if err != nil {
		t.Fatal(err)
	}

	migrate(t, driverURL)
}

func migrate(t *testing.T, driverURL string) {
	d := &Driver{}
	if err := d.Initialize(driverURL); err != nil {
		t.Fatal(err)
	}

	files := []file.File{
		{
			Path:      "/foobar",
			FileName:  "20060102150405_foobar.up.sql",
			Version:   20060102150405,
			Name:      "foobar",
			Direction: direction.Up,
			Content: []byte(`
        CREATE TABLE yolo (
          id int(11) not null primary key auto_increment
        );

				CREATE TABLE yolo1 (
				  id int(11) not null primary key auto_increment
				);
      `),
		},
		{
			Path:      "/foobar",
			FileName:  "20060102150405_foobar.down.sql",
			Version:   20060102150405,
			Name:      "foobar",
			Direction: direction.Down,
			Content: []byte(`
        DROP TABLE yolo;
      `),
		},
		{
			Path:      "/foobar",
			FileName:  "20070000000000_foobar.up.sql",
			Version:   20070000000000,
			Name:      "foobar",
			Direction: direction.Up,
			Content: []byte(`

      	// a comment
				CREATE TABLE error (
          id THIS WILL CAUSE AN ERROR
        );
      `),
		},
	}

	pipe := pipep.New()
	go d.Migrate(files[0], pipe)
	errs := pipep.ReadErrors(pipe)
	if len(errs) > 0 {
		t.Fatal(errs)
	}

	version, err := d.Version()
	if err != nil {
		t.Fatal(err)
	}

	if version != 20060102150405 {
		t.Errorf("Expected version to be: %d, got: %d", 20060102150405, version)
	}

	// Check versions applied in DB
	expectedVersions := file.Versions{20060102150405}
	versions, err := d.Versions()
	if err != nil {
		t.Errorf("Could not fetch versions: %s", err)
	}

	pipe = pipep.New()
	go d.Migrate(files[1], pipe)
	errs = pipep.ReadErrors(pipe)
	if len(errs) > 0 {
		t.Fatal(errs)
	}

	pipe = pipep.New()
	go d.Migrate(files[2], pipe)
	errs = pipep.ReadErrors(pipe)
	if len(errs) == 0 {
		t.Error("Expected test case to fail")
	}

	// Check versions applied in DB
	expectedVersions = file.Versions{}
	versions, err = d.Versions()
	if err != nil {
		t.Errorf("Could not fetch versions: %s", err)
	}

	if !reflect.DeepEqual(versions, expectedVersions) {
		t.Errorf("Expected versions to be: %v, got: %v", expectedVersions, versions)
	}

	if err := d.Close(); err != nil {
		t.Fatal(err)
	}
}

func dropTestTables(t *testing.T, db *sql.DB) {
	if _, err := db.Exec(`DROP TABLE IF EXISTS yolo, yolo1, ` + tableName); err != nil {
		t.Fatal(err)
	}
}
