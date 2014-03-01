package main

import (
	"errors"
	"os"
	"reflect"
	"testing"
)

func TestCanParseServicesFromCloudfoundry(t *testing.T) {
	servicesEnvVar := `VCAP_SERVICES={"elephantsql-n/a":[{"name":"production-db2","label":"elephantsql-n/a","tags":[],"plan":"free","credentials":{"uri":"postgres://foobar"}}]}`
	CommandRunner = func(name string, args ...string) string {
		assert(t, name == "/usr/local/bin/cf", "Bad path to cf")
		assert(t, reflect.DeepEqual(args, []string{"files", "foo", "logs/env.log"}), "Bad env variable lookup")
		return servicesEnvVar
	}

	services := GetServices("foo")
	assert(t, services.ElephantSql[0].Name == "production-db2", services.ElephantSql[0].Name)
	assert(t, services.ElephantSql[0].Credentials["uri"] == "postgres://foobar", services.ElephantSql[0].Credentials["uri"])
}

func TestCanExecElephantSqlServices(t *testing.T) {
	elephantSql := CfDbService{}
	elephantSql.Credentials = map[string]string{}
	elephantSql.Credentials["uri"] = "postgres://localhost"

	CommandExecer = func(argv0 string, argv []string, envv []string) error {
		assert(t, argv0 == "/usr/local/bin/psql", "Bad path to psql")
		assert(t, argv[0] == "psql", argv[0])
		assert(t, argv[1] == "postgres://localhost", argv[1])
		return errors.New("foo")
	}

	err := elephantSql.Exec()
	assert(t, err.Error() == "foo", "error should be foo")
}

func TestCanFindServiceByName(t *testing.T) {
	services := CfServices{}
	elephantToFind := CfDbService{
		"babar",
		map[string]string{
			"uri": "postgres://localhost",
		}}
	elephantToNotFind := CfDbService{}
	services.ElephantSql = append(services.ElephantSql, elephantToNotFind, elephantToFind)
	originalGetServices := GetServices
	GetServices = func(appName string) CfServices {
		return services
	}

	foundService := findService(services, "babar")
	GetServices = originalGetServices
	assert(t, foundService.Name == "babar", "Did not find babar")
}

func TestFindsFirstDbByDefault(t *testing.T) {
	firstService := CfDbService{"first service", nil}
	secondService := CfDbService{"second service", nil}
	services := CfServices{
		[]CfDbService{firstService, secondService},
	}
	foundService := findService(services, "")
	assert(t, foundService.Name == "first service", foundService.Name)
}

func TestMainCanTakeServiceNameAsArg(t *testing.T) {
	GetVcapServicesEnv = func(appName string) string {
		return `{"elephantsql-n/a":[{"name":"production-db2","label":"elephantsql-n/a","tags":[],"plan":"free","credentials":{"uri":"postgres://foobar"}}, {"name":"babar","label":"elephantsql-n/a","tags":[],"plan":"free","credentials":{"uri":"postgres://babar"}}]}`
	}

	CommandExecer = func(argv0 string, argv []string, envv []string) error {
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
