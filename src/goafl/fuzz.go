package goafl

import (
	"bufio"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strings"
)

// Set up shared memory and semaphore
func setShm() {
}

// Parse configuration from file and return a map
func ParseConf(confPath string) map[string]string {
	conf := make(map[string]string)

	confFile, err := os.Open(confPath)
	if err != nil {
		panic(err)
	}
	defer confFile.Close()

	scanner := bufio.NewScanner(confFile)
	linenum := 1
	for scanner.Scan() {
		line := scanner.Text()
		spl := strings.Split(line, ":")
		if len(spl) != 2 {
			log.Fatalf("Line %d format error.\n", linenum)
		} else {
			conf[strings.Trim(spl[0], " \n")] = strings.Trim(spl[1], " \n")
		}
	}

	if scanner.Err() != nil {
		log.Fatal(scanner.Err())
	}

	for _, s := range []string{"afl_path", "binary_path", "input_path", "output_path"} {
		if _, ok := conf[s]; !ok {
			log.Fatalf("Configuration lack of %s\n", s)
		}
	}
	log.Printf("Read configuration successfully.")

	return conf
}

// Execute afl process with conf and return cmd struct
func ExecAFL(conf map[string]string) *exec.Cmd {
	args := make([]string, 0)
	args = append(args, "-i")
	args = append(args, conf["input_path"])
	args = append(args, "-o")
	args = append(args, conf["output_path"])

	args = append(args, "--")
	args = append(args, conf["binary_path"])
	if _, ok := conf["fuzz_args"]; ok {
		args = append(args, strings.Split(conf["fuzz_args"], " ")...)
	}

	cmd := exec.Command(conf["afl_path"], args...)
	cmd.Stdout = ioutil.Discard
	err := cmd.Start()
	if err != nil {
		log.Fatal(err)
	}
	return cmd
}

// Get bitmao from afl
func GetBitmap() {
}

// Get status from afl
func GetStatus() {
}
