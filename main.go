package main

import (
	"bufio"
	"bytes"
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

const VERSION = "0.8.0"
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

func cloud_init_logs(w http.ResponseWriter, r *http.Request) {
	log.Printf("method=\"cloud_init_logs\" clientip=\"%s\" action=\"version\" function=\"version\" mode=\"\" arg1=\"\" arg2=\"\" version=\"%s\"\r\n", GetIP(r), VERSION)

	content, err := ioutil.ReadFile("C:\\ProgramData\\Amazon\\EC2-Windows\\Launch\\Log\\UserdataExecution.log") // the file is inside the local directory
	if err != nil {
		fmt.Println("Error reading file")
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte(content))
}

func getlisting(d string) string {

	buf := new(bytes.Buffer)

	_, err := os.Stat(d)
	if os.IsNotExist(err) {
		buf.WriteString(d + " folder does not exist. \n")
		return buf.String()
	}

	buf.WriteString("Dir Listing for " + d + ": \n")
	files, err := ioutil.ReadDir(d)
	if err != nil {
		log.Fatal(err)
	}
	for _, f := range files {
		buf.WriteString(fmt.Sprintf("    - %s %s\n", f.Mode().String(), f.Name()))
	}
	return buf.String()
}

func dirs(w http.ResponseWriter, r *http.Request) {
	log.Printf("method=\"dirs\" clientip=\"%s\" action=\"version\" function=\"version\" mode=\"\" arg1=\"\" arg2=\"\" version=\"%s\"\r\n", GetIP(r), VERSION)

	buf := new(bytes.Buffer)
	ls := getlisting("C:\\inetpub\\wwwroot")
	buf.WriteString(ls)
	buf.WriteString("\n")

	ls = getlisting("C:\\Users\\Administrator\\Packages")
	buf.WriteString(ls)
	buf.WriteString("\n")

	ls = getlisting("C:\\CAFS\\Packages")
	buf.WriteString(ls)
	buf.WriteString("\n")

	/*buf := new(bytes.Buffer)

	buf.WriteString("Dir Listing for C:\\inetpub\\wwwroot: \n")
	files, err := ioutil.ReadDir("C:\\inetpub\\wwwroot")
	if err != nil {
		log.Fatal(err)
	}
	for _, f := range files {
		buf.WriteString(fmt.Sprintf("    - %s %s\n", f.Mode().String(), f.Name()))
	}

	buf.WriteString("Dir Listing for C:\\Users\\Administrator\\Packages \n")
	files, err = ioutil.ReadDir("C:\\Users\\Administrator\\Packages")
	if err != nil {
		log.Fatal(err)
	}
	for _, f := range files {
		buf.WriteString(fmt.Sprintf("    - %s %s\n", f.Mode().String(), f.Name()))
	}*/

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "text/plain")
	w.Write(buf.Bytes())
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
	http.HandleFunc("/api/v1/clogs", cloud_init_logs)
	http.HandleFunc("/api/v1/dirs", dirs)
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
