package main

import "os/exec"
import "fmt"
import "encoding/json"
import "os"
import "regexp"
import "syscall"

func execPostgres(connectionString string) error {
	psqlArgs := []string{"psql", connectionString}
	env := os.Environ()
	psqlPath, pathErr := exec.LookPath("psql")
	if pathErr != nil {
		panic(pathErr)
	}
	psqlErr := commandExecer(psqlPath, psqlArgs, env)
	return psqlErr
}

var commandExecer = func(argv0 string, argv []string, envv []string) error {
	return syscall.Exec(argv0, argv, envv)
}

var commandRunner = func(name string, args ...string) string {
	envCmd := exec.Command(name, args...)
	out, err := envCmd.Output()
	if err != nil {
		panic(err)
	}
	return string(out)
}

type cfServices struct {
	ElephantSql []cfDbService `json:"elephantsql-n/a"`
}

type cfDbService struct {
	Name        string            `json:"name"`
	Credentials map[string]string `json:"credentials"`
}

type commandDoer interface {
	exec() error
	run() string
}

type serviceFinder struct {
	commandDoer commandDoer
	services    cfServices
}

func (sf *serviceFinder) findAll(appName string) {
	matchBytes := []byte(getVcapServicesEnv(appName))
	var servicesJson cfServices
	jsonErr := json.Unmarshal(matchBytes, &servicesJson)
	if jsonErr != nil {
		panic(jsonErr)
	}
	sf.services = servicesJson
}

func (sf serviceFinder) find(serviceName string) cfDbService {
	var selectedDb cfDbService
	elephantSql := sf.services.ElephantSql
	// Grab the first database if no service name is given
	if serviceName == "" {
		selectedDb = elephantSql[0]
	} else {
		for _, dbService := range elephantSql {
			if dbService.Name == serviceName {
				selectedDb = dbService
			}
		}
	}
	return selectedDb
}

func (s cfDbService) exec() error {
	credentials := s.Credentials
	uri := credentials["uri"]
	fmt.Println("Connecting to the following PostgreSQL url: ", uri)
	return execPostgres(uri)
}

func getVcapServicesEnv(appName string) string {
	out := commandRunner("/usr/local/bin/cf", "files", appName, "logs/env.log")
	r, err := regexp.Compile("VCAP_SERVICES=(.*)")
	if err != nil {
		panic(err)
	}

	match := r.FindStringSubmatch(out)
	return match[1]
}

func main() {
	appName := os.Args[1]
	var serviceName string
	if len(os.Args) > 2 {
		serviceName = os.Args[2]
	} else {
		serviceName = ""
	}

	finder := serviceFinder{}
	finder.findAll(appName)
	serviceToUse := finder.find(serviceName)
	err := serviceToUse.exec()
	if err != nil {
		panic(err)
	}
}
