package goafl

import (
	"bufio"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strings"
)

type Fuzzer struct {
	ipc  *FuzzIPC
	Proc *exec.Cmd
}

// Set up shared memory and semaphore
func (fuzzer *Fuzzer) setShm() error {
	ipc, err := SetupIPC()
	fuzzer.ipc = ipc
	return err
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
func ExecAFL(conf map[string]string) *Fuzzer {
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

	log.Println("Setup shared memory...")
	// setup shared memory
	ipc, err := SetupIPC()
	if err != nil {
		log.Fatalln("Setup shared memory error!")
	} else {
		log.Println("Setup shared memory successfully")
	}

	cmd := exec.Command(conf["afl_path"], args...)
	cmd.Stdout = ioutil.Discard
	err = cmd.Start()
	if err != nil {
		log.Fatalln("AFL start fail!")
		log.Fatalln(err)
	} else {
		log.Println("AFL start successfully!")
	}

	fuzzer := new(Fuzzer)
	fuzzer.Proc = cmd
	fuzzer.ipc = ipc
	return fuzzer
}

// Get bitmap from afl
func (fuzzer *Fuzzer) GetBitmap() []byte {
	bitmap, err := fuzzer.ipc.GetBitMap()
	if err != nil {
		log.Println("Git bitmap error!")
	}
	return bitmap
}

// Get status from afl
func GetStatus() {
}
