package main

import (
	"errors"
	"os"
	"reflect"
	"testing"
)

func TestCanGetPostgresConnectionString(t *testing.T) {
	CommandRunner = func(name string, args ...string) string {
		assert(t, name == "/usr/local/bin/cf", "Bad path to cf")
		assert(t, reflect.DeepEqual(args, []string{"files", "foo", "logs/env.log"}), "Bad env variable lookup")
		return `VCAP_SERVICES={"elephantsql-n/a":[{"name":"production-db2","label":"elephantsql-n/a","tags":[],"plan":"free","credentials":{"uri":"postgres://foobar"}}]}`
	}

	connectionString := GetPostgresConnectionString("foo", "")
	assert(t, connectionString == "postgres://foobar", "should equal postgres://foobar")
}

func TestCanGetPostgresConnectionStringForAnyElephantSqlDb(t *testing.T) {
	CommandRunner = func(name string, args ...string) string {
		return `VCAP_SERVICES={"elephantsql-n/a":[{"name":"production-db2","label":"elephantsql-n/a","tags":[],"plan":"free","credentials":{"uri":"postgres://foobar"}},{"name":"my-other-db","label":"elephantsql-n/a","tags":[],"plan":"free","credentials":{"uri":"postgres://a-different-db"}}]}`
	}

	conn := GetPostgresConnectionString("foo", "my-other-db")
	assert(t, conn == "postgres://a-different-db", "Did not find my-other-db in services", conn)
}

func TestExecPostgres(t *testing.T) {
	CommandExecer = func(argv0 string, argv []string, envv []string) error {
		assert(t, argv0 == "/usr/local/bin/psql", "Bad path to psql")
		return errors.New("foo")
	}

	err := ExecPostgres("postgres://localhost")
	assert(t, err.Error() == "foo", "error should be foo")
}

func TestMain(t *testing.T) {
	GetPostgresConnectionString = func(appName string, serviceName string) string {
		assert(t, appName == "my-foo-app", "Bad app name")
		return "postgres://a-random-postgres"
	}

	ExecPostgres = func(conn string) error {
		assert(t, conn == "postgres://a-random-postgres", "Not using GetPostgresConnectionString")
		return errors.New("A random error")
	}

	originalArgs := os.Args
	os.Args = append(os.Args, "my-foo-app")
	defer func() {
		assert(t, recover() != nil, "Should have panic'ed")
		os.Args = originalArgs
	}()
	main()
}

func TestMainCanTakeServiceNameAsArg(t *testing.T) {
	GetPostgresConnectionString = func(appName string, serviceName string) string {
		assert(t, appName == "my-foo-app", "Bad app name")
		assert(t, serviceName == "my-db-service", "Did not parse my-db-service arg")
		return "postgres://a-random-postgres"
	}

	ExecPostgres = func(conn string) error {
		return nil
	}

	os.Args = append(os.Args, "my-foo-app", "my-db-service")
	main()
}

func assert(t *testing.T, b bool, message ...string) {
	if b != true {
		t.Fatal(message)
	}
}
