package main

import (
	"flag"
	"goafl"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	confPath := flag.String("conf", "", "Configuration of go-fuzz-lop")
	flag.Parse()
	conf := goafl.ParseConf(*confPath)
	fuzzer := goafl.ExecAFL(conf)
	defer fuzzer.Proc.Wait()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGKILL, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGINT)
	go func() {
		<-sigChan
		fuzzer.Proc.Process.Kill()
	}()
}
