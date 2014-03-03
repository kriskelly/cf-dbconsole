package main

import (
	"errors"
	"testing"
)

type myCommandDoer struct {
	t                 *testing.T
	execError         error
	expectedExecArgv0 string
	expectedExecArgv  []string
	expectedRunName   string
	expectedRunArgs   []string
	runOutput         string
}

func (m myCommandDoer) exec(argv0 string, argv []string, envv []string) error {
	assert(m.t, argv0 == m.expectedExecArgv0, argv0, "should have been", m.expectedExecArgv0)
	for i, expectedArg := range m.expectedExecArgv {
		assert(m.t, expectedArg == argv[i], argv[i], "should have been", expectedArg)
	}
	return m.execError
}

func (m myCommandDoer) run(name string, args ...string) string {
	assert(m.t, name == m.expectedRunName, "Bad path to cf")
	for i, arg := range args {
		assert(m.t, arg == m.expectedRunArgs[i], "Should have been ", m.expectedRunArgs[i])
	}
	return m.runOutput
}

func TestCanParseServicesFromCloudfoundry(t *testing.T) {
	servicesEnvVar := `VCAP_SERVICES={"elephantsql-n/a":[{"name":"production-db2","label":"elephantsql-n/a","tags":[],"plan":"free","credentials":{"uri":"postgres://foobar"}}]}`
	finder := serviceFinder{
		commandDoer: myCommandDoer{
			t:               t,
			runOutput:       servicesEnvVar,
			expectedRunArgs: []string{"files", "foo", "logs/env.log"},
			expectedRunName: "/usr/local/bin/cf",
		}}
	finder.findAll("foo")
	services := finder.services
	assert(t, services.ElephantSql[0].Name == "production-db2", services.ElephantSql[0].Name)
	assert(t, services.ElephantSql[0].Credentials["uri"] == "postgres://foobar", services.ElephantSql[0].Credentials["uri"])
}

func TestCanExecElephantSqlServices(t *testing.T) {
	elephantSql := postgresService{}
	elephantSql.Credentials = map[string]string{}
	elephantSql.Credentials["uri"] = "postgres://localhost"

	doer := myCommandDoer{
		t:                 t,
		execError:         errors.New("foo"),
		expectedExecArgv0: "/usr/local/bin/psql",
		expectedExecArgv:  []string{"psql", "postgres://localhost"},
	}
	err := elephantSql.exec(doer)
	assert(t, err.Error() == "foo", "error should be foo")
}

func TestCanFindServiceByName(t *testing.T) {
	services := cfServices{}
	elephantToFind := postgresService{
		"babar",
		map[string]string{
			"uri": "postgres://localhost",
		}}
	elephantToNotFind := postgresService{}
	services.ElephantSql = append(services.ElephantSql, elephantToNotFind, elephantToFind)
	finder := serviceFinder{}
	finder.services = services
	foundService := finder.find("babar").(postgresService)
	assert(t, foundService.Name == "babar", "Did not find babar")
}

func TestFindsFirstDbByDefault(t *testing.T) {
	firstService := postgresService{"first service", nil}
	secondService := postgresService{"second service", nil}
	services := cfServices{
		ElephantSql: []postgresService{firstService, secondService},
	}
	finder := serviceFinder{services: services}
	foundService := finder.find("").(postgresService)
	assert(t, foundService.Name == "first service", foundService.Name)
}

func TestCanFindClearDbMysqlService(t *testing.T) {
	servicesEnvVar := `VCAP_SERVICES={"cleardb-n/a":[{"name":"my-cleardb","credentials":{"name":"my-dbname","hostname":"my-hostname","port":"3306","username":"my-user","password":"mypass"}}]}`
	doer := myCommandDoer{
		t:               t,
		runOutput:       servicesEnvVar,
		expectedRunArgs: []string{"files", "foo", "logs/env.log"},
		expectedRunName: "/usr/local/bin/cf",
	}
	finder := serviceFinder{commandDoer: doer}
	finder.findAll("foo")
	actualService := finder.services.ClearDb[0]
	foundService := finder.find("my-cleardb").(mysqlService)
	assert(t, actualService.Name == foundService.Name, "should have found my-cleardb")
}

func TestCanExecMysqlService(t *testing.T) {
	service := mysqlService{
		Name: "my-cleardb",
		Credentials: map[string]string{
			"name":     "db-name",
			"hostname": "mysql-hostname",
			"port":     "3306",
			"username": "mysql-username",
			"password": "garbage-password",
		},
	}

	commandDoer := myCommandDoer{
		expectedExecArgv0: "/usr/local/bin/mysql",
		expectedExecArgv: []string{
			"mysql",
			"-h",
			"mysql-hostname",
			"-u",
			"mysql-username",
			"-P",
			"3306",
			"-pgarbage-password",
			"-D",
			"db-name",
		},
	}

	service.exec(commandDoer)
}

func TestMainCanTakeServiceNameAsArg(t *testing.T) {
	doer := myCommandDoer{
		t:                 t,
		runOutput:         `VCAP_SERVICES={"elephantsql-n/a":[{"name":"production-db2","label":"elephantsql-n/a","tags":[],"plan":"free","credentials":{"uri":"postgres://foobar"}}, {"name":"babar","label":"elephantsql-n/a","tags":[],"plan":"free","credentials":{"uri":"postgres://babar"}}]}`,
		expectedRunArgs:   []string{"files", "my-foo-app", "logs/env.log"},
		expectedRunName:   "/usr/local/bin/cf",
		execError:         nil,
		expectedExecArgv0: "/usr/local/bin/psql",
		expectedExecArgv:  []string{"psql", "postgres://babar"},
	}

	finder := serviceFinder{commandDoer: doer}
	finder.findAndExec("my-foo-app", "babar")
}

func assert(t *testing.T, b bool, message ...string) {
	if b != true {
		t.Fatal(message)
	}
}
