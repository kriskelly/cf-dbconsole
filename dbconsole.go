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
	RedisCloud  []redisService    `json:"rediscloud-n/a"`
}

type postgresService struct {
	Name        string            `json:"name"`
	Credentials map[string]string `json:"credentials"`
}

type mysqlService struct {
	Name        string            `json:"name"`
	Credentials map[string]string `json:"credentials"`
}

type redisService struct {
	Name        string            `json:"name"`
	Credentials map[string]string `json:"credentials"`
}

type commandRunner interface {
	exec(argv0 string, argv []string, envv []string) error
	run(name string, args ...string) string
}

type service interface {
	exec(runner commandRunner) error
	name() string
}

type serviceFinder struct {
	commandRunner commandRunner
	services      cfServices
}

type cliCommandRunner struct{}

func (c cliCommandRunner) exec(argv0 string, argv []string, envv []string) error {
	return syscall.Exec(argv0, argv, envv)
}

func (c cliCommandRunner) run(name string, args ...string) string {
	envCmd := exec.Command(name, args...)
	out, err := envCmd.Output()
	if err != nil {
		panic(err)
	}
	return string(out)
}

func (sf *serviceFinder) findAll(appName string) {
	matchBytes := []byte(getVcapServicesEnv(sf.commandRunner, appName))
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
	for _, r := range sf.services.RedisCloud {
		allServices = append(allServices, r)
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
	return serviceToUse.exec(sf.commandRunner)
}

func (m mysqlService) exec(runner commandRunner) error {
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
	return runner.exec(mysqlPath, mysqlArgs, env)
}

func (m mysqlService) name() string {
	return m.Name
}

func (s postgresService) exec(runner commandRunner) error {
	credentials := s.Credentials
	uri := credentials["uri"]
	fmt.Println("Connecting to the following PostgreSQL url: ", uri)
	psqlArgs := []string{"psql", uri}
	env := os.Environ()
	psqlPath, pathErr := exec.LookPath("psql")
	if pathErr != nil {
		panic(pathErr)
	}
	psqlErr := runner.exec(psqlPath, psqlArgs, env)
	return psqlErr
}

func (p postgresService) name() string {
	return p.Name
}

func (r redisService) exec(runner commandRunner) error {
	path, err := exec.LookPath("redis-cli")
	if err != nil {
		panic(err)
	}
	env := os.Environ()
	args := []string{
		path,
		"-p",
		r.Credentials["port"],
		"-h",
		r.Credentials["hostname"],
		"-a",
		r.Credentials["password"],
	}
	return runner.exec(path, args, env)
}

func (r redisService) name() string {
	return r.Name
}

func getVcapServicesEnv(runner commandRunner, appName string) string {
	out := runner.run("/usr/local/bin/cf", "files", appName, "logs/env.log")
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

	finder := serviceFinder{commandRunner: cliCommandRunner{}}
	err := finder.findAndExec(appName, serviceName)
	if err != nil {
		panic(err)
	}
}
