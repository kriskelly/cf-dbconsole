package main

import "os/exec"
import "fmt"
import "encoding/json"
import "os"
import "regexp"
import "syscall"

func main() {
	appName := os.Args[1]
	envCmd := exec.Command("/usr/local/bin/cf", "files", appName, "logs/env.log")
	out, err := envCmd.Output()
	if err != nil {
		panic(err)
	}
	r, err := regexp.Compile("VCAP_SERVICES=(.*)")
	if err != nil {
		panic(err)
	}

	match := r.FindStringSubmatch(string(out))
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
	fmt.Println("Connecting to the following PostgreSQL url: ", uri)

	psqlArgs := []string{"psql", uri}
	env := os.Environ()
	psqlPath, pathErr := exec.LookPath("psql")
	if pathErr != nil {
		panic(pathErr)
	}
	psqlErr := syscall.Exec(psqlPath, psqlArgs, env)
	if psqlErr != nil {
		panic(psqlErr)
	}
}
