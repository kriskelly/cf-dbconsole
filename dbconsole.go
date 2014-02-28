package main

import "os/exec"
import "fmt"
import "encoding/json"
import "os"
import "regexp"
import "syscall"

var ExecPostgres = func(connectionString string) error {
	psqlArgs := []string{"psql", connectionString}
	env := os.Environ()
	psqlPath, pathErr := exec.LookPath("psql")
	if pathErr != nil {
		panic(pathErr)
	}
	psqlErr := CommandExecer(psqlPath, psqlArgs, env)
	return psqlErr
}

var CommandExecer = func(argv0 string, argv []string, envv []string) error {
	return syscall.Exec(argv0, argv, envv)
}

var CommandRunner = func(name string, args ...string) string {
	envCmd := exec.Command(name, args...)
	out, err := envCmd.Output()
	if err != nil {
		panic(err)
	}
	return string(out)
}

var GetPostgresConnectionString = func(appName string, serviceName string) string {
	out := CommandRunner("/usr/local/bin/cf", "files", appName, "logs/env.log")
	r, err := regexp.Compile("VCAP_SERVICES=(.*)")
	if err != nil {
		panic(err)
	}

	match := r.FindStringSubmatch(out)
	matchBytes := []byte(match[1])
	var servicesJson CfServices
	jsonErr := json.Unmarshal(matchBytes, &servicesJson)
	if jsonErr != nil {
		panic(jsonErr)
	}

	return findElephantSqlUri(servicesJson, serviceName)
}

type CfServices struct {
	ElephantSql []CfDbService `json:"elephantsql-n/a"`
}

type CfDbService struct {
	Name        string            `json:"name"`
	Credentials map[string]string `json:"credentials"`
}

func findElephantSqlUri(servicesJson CfServices, serviceName string) string {
	var selectedDb CfDbService
	elephantSql := servicesJson.ElephantSql
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

	credentials := selectedDb.Credentials
	return credentials["uri"]
}

func main() {
	appName := os.Args[1]
	var serviceName string
	if len(os.Args) > 2 {
		serviceName = os.Args[2]
	} else {
		serviceName = ""
	}
	uri := GetPostgresConnectionString(appName, serviceName)
	fmt.Println("Connecting to the following PostgreSQL url: ", uri)
	err := ExecPostgres(uri)
	if err != nil {
		panic(err)
	}
}
