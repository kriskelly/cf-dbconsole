package main

import "os/exec"
import "fmt"
import "encoding/json"
import "os"
import "regexp"
import "syscall"

func main() {
	appName := os.Args[1]
	uri := GetPostgresConnectionString(appName)
	fmt.Println("Connecting to the following PostgreSQL url: ", uri)
	err := ExecPostgres(uri)
	if err != nil {
		panic(err)
	}
}

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

var GetPostgresConnectionString = func(appName string) string {
	out := CommandRunner("/usr/local/bin/cf", "files", appName, "logs/env.log")
	r, err := regexp.Compile("VCAP_SERVICES=(.*)")
	if err != nil {
		panic(err)
	}

	match := r.FindStringSubmatch(out)
	matchBytes := []byte(match[1])
	var servicesJson map[string]interface{}
	jsonErr := json.Unmarshal(matchBytes, &servicesJson)
	if jsonErr != nil {
		panic(jsonErr)
	}

	// TODO: Support access to different databases if there are multiple
	elephantSql := servicesJson["elephantsql-n/a"].([]interface{})
	firstDb := elephantSql[0].(map[string]interface{})
	credentials := firstDb["credentials"].(map[string]interface{})
	uri := credentials["uri"].(string)
	return uri
}
