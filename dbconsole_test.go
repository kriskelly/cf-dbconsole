package main

import (
	"errors"
	"os"
	"reflect"
	"testing"
)

func TestCanParseServicesFromCloudfoundry(t *testing.T) {
	servicesEnvVar := `VCAP_SERVICES={"elephantsql-n/a":[{"name":"production-db2","label":"elephantsql-n/a","tags":[],"plan":"free","credentials":{"uri":"postgres://foobar"}}]}`
	commandRunner = func(name string, args ...string) string {
		assert(t, name == "/usr/local/bin/cf", "Bad path to cf")
		assert(t, reflect.DeepEqual(args, []string{"files", "foo", "logs/env.log"}), "Bad env variable lookup")
		return servicesEnvVar
	}
	finder := serviceFinder{}
	finder.findAll("foo")
	services := finder.services
	assert(t, services.ElephantSql[0].Name == "production-db2", services.ElephantSql[0].Name)
	assert(t, services.ElephantSql[0].Credentials["uri"] == "postgres://foobar", services.ElephantSql[0].Credentials["uri"])
}

func TestCanExecElephantSqlServices(t *testing.T) {
	elephantSql := cfDbService{}
	elephantSql.Credentials = map[string]string{}
	elephantSql.Credentials["uri"] = "postgres://localhost"

	commandExecer = func(argv0 string, argv []string, envv []string) error {
		assert(t, argv0 == "/usr/local/bin/psql", "Bad path to psql")
		assert(t, argv[0] == "psql", argv[0])
		assert(t, argv[1] == "postgres://localhost", argv[1])
		return errors.New("foo")
	}

	err := elephantSql.exec()
	assert(t, err.Error() == "foo", "error should be foo")
}

func TestCanFindServiceByName(t *testing.T) {
	services := cfServices{}
	elephantToFind := cfDbService{
		"babar",
		map[string]string{
			"uri": "postgres://localhost",
		}}
	elephantToNotFind := cfDbService{}
	services.ElephantSql = append(services.ElephantSql, elephantToNotFind, elephantToFind)
	finder := serviceFinder{}
	finder.services = services
	foundService := finder.find("babar")
	assert(t, foundService.Name == "babar", "Did not find babar")
}

func TestFindsFirstDbByDefault(t *testing.T) {
	firstService := cfDbService{"first service", nil}
	secondService := cfDbService{"second service", nil}
	services := cfServices{
		[]cfDbService{firstService, secondService},
	}
	finder := serviceFinder{services: services}
	foundService := finder.find("")
	assert(t, foundService.Name == "first service", foundService.Name)
}

func TestMainCanTakeServiceNameAsArg(t *testing.T) {
	getVcapServicesEnv = func(appName string) string {
		return `{"elephantsql-n/a":[{"name":"production-db2","label":"elephantsql-n/a","tags":[],"plan":"free","credentials":{"uri":"postgres://foobar"}}, {"name":"babar","label":"elephantsql-n/a","tags":[],"plan":"free","credentials":{"uri":"postgres://babar"}}]}`
	}

	commandExecer = func(argv0 string, argv []string, envv []string) error {
		assert(t, argv0 == "/usr/local/bin/psql", "Bad psql path", argv0)
		assert(t, argv[0] == "psql", "Bad argv[0]", argv[0])
		assert(t, argv[1] == "postgres://babar", "Bad argv[1]", argv[1])
		return nil
	}

	os.Args = []string{"console", "my-foo-app", "babar"}
	main()
}

func assert(t *testing.T, b bool, message ...string) {
	if b != true {
		t.Fatal(message)
	}
}
