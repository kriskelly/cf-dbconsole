package main

import "os/exec"
import "fmt"
import "encoding/json"
import "os"
import "regexp"
import "syscall"

var execPostgres = func(connectionString string) error {
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

var getVcapServicesEnv = func(appName string) string {
	out := commandRunner("/usr/local/bin/cf", "files", appName, "logs/env.log")
	r, err := regexp.Compile("VCAP_SERVICES=(.*)")
	if err != nil {
		panic(err)
	}

	match := r.FindStringSubmatch(out)
	return match[1]
}

type cfServices struct {
	ElephantSql []cfDbService `json:"elephantsql-n/a"`
}

type cfDbService struct {
	Name        string            `json:"name"`
	Credentials map[string]string `json:"credentials"`
}

func (s cfDbService) Exec() error {
	credentials := s.Credentials
	uri := credentials["uri"]
	fmt.Println("Connecting to the following PostgreSQL url: ", uri)
	return execPostgres(uri)
}

var getServices = func(appName string) cfServices {
	matchBytes := []byte(getVcapServicesEnv(appName))
	var servicesJson cfServices
	jsonErr := json.Unmarshal(matchBytes, &servicesJson)
	if jsonErr != nil {
		panic(jsonErr)
	}
	return servicesJson
}

func findService(services cfServices, serviceName string) cfDbService {
	var selectedDb cfDbService
	elephantSql := services.ElephantSql
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

func main() {
	appName := os.Args[1]
	var serviceName string
	if len(os.Args) > 2 {
		serviceName = os.Args[2]
	} else {
		serviceName = ""
	}

	services := getServices(appName)
	serviceToUse := findService(services, serviceName)
	err := serviceToUse.Exec()
	if err != nil {
		panic(err)
	}
}
