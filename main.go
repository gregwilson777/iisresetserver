package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/kardianos/service"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
)

const VERSION = "0.5.0"
const serviceName = "QA Manager"
const serviceDescription = "Simple Service for performing IISResets and DBBackups remotely"

var (
	debug            bool   = true
	port             string = "9991"
	serviceIsRunning bool
	programIsRunning bool
	writingSync      sync.Mutex
)

type QARequest struct {
	Password    string `json:"password"`
	Action      string `json:"action"`
	Mode        string `json:"mode"`
	RequestArg1 string `json:"reqarg1"`
	RequestArg2 string `json:"reqarg2"`
}

type QAResponse struct {
	Error     string `json:"error"`
	CmdOutput string `json:"command_output"`
}

func version(w http.ResponseWriter, r *http.Request) {
	var resp QAResponse
	log.Printf("method=\"version\" clientip=\"%s\" action=\"version\" function=\"version\" mode=\"\" arg1=\"\" arg2=\"\" version=\"%s\"\r\n", GetIP(r), VERSION)

	resp.CmdOutput = VERSION
	json.NewEncoder(w).Encode(resp)
	w.Header().Set("Content-Type", "application/json")
}

func ping(w http.ResponseWriter, r *http.Request) {
	var resp QAResponse
	log.Printf("method=\"ping\" clientip=\"%s\" action=\"ping\" function=\"ping\" mode=\"\" arg1=\"\" arg2=\"\" \r\n", GetIP(r))

	resp.CmdOutput = "PONG"
	json.NewEncoder(w).Encode(resp)
	w.Header().Set("Content-Type", "application/json")
}

func dbbackup(w http.ResponseWriter, r *http.Request) {
	runfunc(w, r, "dbbackup", "dbbackup.ps1", true)
}

func doreset(w http.ResponseWriter, r *http.Request) {
	runfunc(w, r, "doreset", "iisreset.exe", false)
}

func GetIP(r *http.Request) string {
	forwarded := r.Header.Get("X-FORWARDED-FOR")
	if forwarded != "" {
		return forwarded
	}
	return r.RemoteAddr
}

func runfunc(w http.ResponseWriter, r *http.Request, m string, f string, parseInput bool) {
	var req QARequest
	var resp QAResponse

	if parseInput {
		reqBody, err0 := ioutil.ReadAll(r.Body)
		if err0 != nil {
			resp.Error = err0.Error()
		}

		if debug {
			log.Printf("%s \r\n", string(reqBody))
		}

		err1 := json.Unmarshal(reqBody, &req)
		if err1 != nil {
			resp.Error = err1.Error()
		}
	}
	
	log.Printf("method=\"runfunc:%s\" clientip=\"%s\" action=\"%s\" function=\"%s\" mode=\"%s\" arg1=\"%s\" arg2=\"%s\" \r\n", m, GetIP(r), req.Action, f, req.Mode, req.RequestArg1, req.RequestArg2)

	cmd := exec.Command(f, req.RequestArg1, req.RequestArg2)
	output, err := cmd.Output()

	resp.CmdOutput = string(output)
	if err != nil {
		resp.Error = err.Error()
		w.WriteHeader(500)
	}
	log.Println("Command output:" + string(output))

	json.NewEncoder(w).Encode(resp)
	w.Header().Set("Content-Type", "application/json")
}

func handleRequests(port string) error {
	http.HandleFunc("/api/v1/iisreset", doreset)
	http.HandleFunc("/api/v1/dbbackup", dbbackup)
	http.HandleFunc("/api/v1/ping", ping)
	http.HandleFunc("/api/v1/version", version)
	log.Printf("Server Started, version %s\n\r", VERSION)
	return http.ListenAndServe(":"+port, nil)
}

func readConfigFile(configFile string) (string, error) {
	file, err := os.Open(configFile)
	if err != nil {
		return "", err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var lines []string
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return "", err
	}
	return strings.TrimSpace(lines[0]), nil
}

type program struct{}

func (p program) run() {
	programIsRunning = true
	err := handleRequests(port)
	if err != nil {
		log.Println("Problem starting web server: " + err.Error())
	}
	programIsRunning = false

	/*for serviceIsRunning {
		fmt.Println("Service is running")
		time.Sleep(1 * time.Second)
	}*/
}

func (p program) Start(s service.Service) error {
	log.Println(s.String() + " started")
	writingSync.Lock()
	serviceIsRunning = true
	writingSync.Unlock()
	go p.run()
	return nil
}

func (p program) Stop(s service.Service) error {
	writingSync.Lock()
	serviceIsRunning = false
	writingSync.Unlock()
	count := 0
	for programIsRunning {
		log.Println(s.String() + " stopping...")
		time.Sleep(1 * time.Second)
		count++
		// allow 5 secs for graceful shutdown....
		if count > 5 {
			programIsRunning = false
		}
	}
	log.Println(s.String() + " stopped")
	return nil
}

func main() {
	var logFile string
	serviceConfig := &service.Config{
		Name:        serviceName,
		DisplayName: serviceName,
		Description: serviceDescription,
	}

	flag.StringVar(&logFile, "l", "qa.log", "output log file")
	flag.Parse()

	fmt.Printf("Using logfile: %s \r\n", logFile)

	f, err := os.OpenFile(logFile, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	log.SetOutput(f)

	prg := &program{}
	s, err := service.New(prg, serviceConfig)
	if err != nil {
		log.Println("Cannot create the service: " + err.Error())
	}
	err = s.Run()
	if err != nil {
		log.Println("Cannot start the service: " + err.Error())
	}
}
