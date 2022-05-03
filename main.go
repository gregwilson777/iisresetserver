package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
)

const VERSION = "0.3.0_beta"

// default password - this should be overridden with config
var password string

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

func ping(w http.ResponseWriter, r *http.Request) {
	var resp QAResponse
	resp.CmdOutput = "PONG"
	json.NewEncoder(w).Encode(resp)
	w.Header().Set("Content-Type", "application/json")
}

func dbbackup(w http.ResponseWriter, r *http.Request) {
	runfunc(w, r, "dbbackup", "dbbackup.ps1")
}

func doreset(w http.ResponseWriter, r *http.Request) {
	runfunc(w, r, "doreset", "iisreset.exe")
}

func GetIP(r *http.Request) string {
	forwarded := r.Header.Get("X-FORWARDED-FOR")
	if forwarded != "" {
		return forwarded
	}
	return r.RemoteAddr
}

func runfunc(w http.ResponseWriter, r *http.Request, m string, f string) {
	var req QARequest
	var resp QAResponse

	reqBody, err0 := ioutil.ReadAll(r.Body)
	if err0 != nil {
		resp.Error = err0.Error()
	}

	err1 := json.Unmarshal(reqBody, &req)
	if err1 != nil {
		resp.Error = err1.Error()
	}

	log.Printf("method=\"runfunc:%s\" clientip=\"%s\" action=\"%s\" function=\"%s\" mode=\"%s\" arg1=\"%s\" arg2=\"%s\" \n", m, GetIP(r), req.Action, f, req.Mode, req.RequestArg1, req.RequestArg2)

	if req.Password == password {
		log.Println("Password matched")
		cmd := exec.Command(f, req.RequestArg1, req.RequestArg2)
		output, err := cmd.Output()

		resp.CmdOutput = string(output)
		if err != nil {
			resp.Error = err.Error()
			w.WriteHeader(500)
		}
		log.Println("Command output:" + string(output))
	} else {
		resp.Error = "Invalid password"
		w.WriteHeader(403)
	}
	json.NewEncoder(w).Encode(resp)
	w.Header().Set("Content-Type", "application/json")
}

func handleRequests(password string, port string) {
	http.HandleFunc("/api/v1/iisreset", doreset)
	http.HandleFunc("/api/v1/dbbackup", dbbackup)
	http.HandleFunc("/api/v1/ping", ping)
	log.Printf("Server Started, version %s", VERSION)
	log.Fatal(http.ListenAndServe(":"+port, nil))
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

func main() {
	// variables declaration
	var configFile string
	var port string

	// flags declaration using flag package
	flag.StringVar(&configFile, "c", "config.txt", "Specify config file")
	flag.StringVar(&port, "p", "9991", "Specify port. Default is 9991")

	// read the config file - the trimmed contents are the password for this service
	var err error
	password, err = readConfigFile(configFile)
	if err != nil {
		fmt.Println("Error reading config file: " + configFile)
		panic(err)
	}
	handleRequests(password, port)
}
