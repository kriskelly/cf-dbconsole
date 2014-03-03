package main

import "os/exec"
import "fmt"
import "encoding/json"
import "os"
import "regexp"
import "syscall"

type cfServices struct {
	ElephantSql []postgresService `json:"elephantsql-n/a"`
	ClearDb     []mysqlService    `json:"cleardb-n/a"`
}

type postgresService struct {
	Name        string            `json:"name"`
	Credentials map[string]string `json:"credentials"`
}

type mysqlService struct {
	Name        string            `json:"name"`
	Credentials map[string]string `json:"credentials"`
}

type commandDoer interface {
	exec(argv0 string, argv []string, envv []string) error
	run(name string, args ...string) string
}

type service interface {
	exec(doer commandDoer) error
	name() string
}

type serviceFinder struct {
	commandDoer commandDoer
	services    cfServices
}

type cliCommandDoer struct{}

func (c cliCommandDoer) exec(argv0 string, argv []string, envv []string) error {
	return syscall.Exec(argv0, argv, envv)
}

func (c cliCommandDoer) run(name string, args ...string) string {
	envCmd := exec.Command(name, args...)
	out, err := envCmd.Output()
	if err != nil {
		panic(err)
	}
	return string(out)
}

func (sf *serviceFinder) findAll(appName string) {
	matchBytes := []byte(getVcapServicesEnv(sf.commandDoer, appName))
	var servicesJson cfServices
	jsonErr := json.Unmarshal(matchBytes, &servicesJson)
	if jsonErr != nil {
		panic(jsonErr)
	}
	sf.services = servicesJson
}

func (sf serviceFinder) find(serviceName string) service {
	var selectedDb service
	allServices := []service{}
	for _, e := range sf.services.ElephantSql {
		allServices = append(allServices, e)
	}
	for _, m := range sf.services.ClearDb {
		allServices = append(allServices, m)
	}
	// Grab the first database if no service name is given
	if serviceName == "" {
		selectedDb = allServices[0]
	} else {
		for _, dbService := range allServices {
			if dbService.name() == serviceName {
				selectedDb = dbService
			}
		}
	}
	return selectedDb
}

func (sf serviceFinder) findAndExec(appName string, serviceName string) error {
	sf.findAll(appName)
	serviceToUse := sf.find(serviceName)
	return serviceToUse.exec(sf.commandDoer)
}

func (m mysqlService) exec(doer commandDoer) error {
	creds := m.Credentials
	mysqlArgs := []string{
		"mysql",
		"-h",
		creds["hostname"],
		"-u",
		creds["username"],
		"-P",
		creds["port"],
		"-p" + creds["password"],
		"-D",
		creds["name"],
	}
	env := os.Environ()
	mysqlPath, err := exec.LookPath("mysql")
	if err != nil {
		panic(err)
	}
	return doer.exec(mysqlPath, mysqlArgs, env)
}

func (m mysqlService) name() string {
	return m.Name
}

func (s postgresService) exec(doer commandDoer) error {
	credentials := s.Credentials
	uri := credentials["uri"]
	fmt.Println("Connecting to the following PostgreSQL url: ", uri)
	psqlArgs := []string{"psql", uri}
	env := os.Environ()
	psqlPath, pathErr := exec.LookPath("psql")
	if pathErr != nil {
		panic(pathErr)
	}
	psqlErr := doer.exec(psqlPath, psqlArgs, env)
	return psqlErr
}

func (p postgresService) name() string {
	return p.Name
}

func getVcapServicesEnv(doer commandDoer, appName string) string {
	out := doer.run("/usr/local/bin/cf", "files", appName, "logs/env.log")
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

	finder := serviceFinder{commandDoer: cliCommandDoer{}}
	err := finder.findAndExec(appName, serviceName)
	if err != nil {
		panic(err)
	}
}
