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

// default password - this should be overridden with config
var password = "nMF-Xff-uVe-jQz"

type ResetRequest struct {
	Password string `json:password`
}

type ResetResponse struct {
	Error     string `json:"error"`
	CmdOutput string `json:"command_output"`
}

func doreset(w http.ResponseWriter, r *http.Request) {
	var req ResetRequest
	var resp ResetResponse

	log.Println("Endpoint Hit: doreset")

	reqBody, err0 := ioutil.ReadAll(r.Body)
	if err0 != nil {
		resp.Error = err0.Error()
	}

	err1 := json.Unmarshal(reqBody, &req)
	if err1 != nil {
		resp.Error = err1.Error()
	}

	if req.Password == password {
		log.Println("Password matched")
		cmd := exec.Command("iisreset.exe")
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
