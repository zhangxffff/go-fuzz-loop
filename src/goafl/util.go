package goafl

import "os"

type Semaphore struct {
}

func (*Semaphore) open(name string, flag int, mode os.FileMode, value int) error {
}

func (*Semaphore) getValue() (int, error) {
}

func (*Semaphore) post() error {
}

func (*Semaphore) wait() error {
}

func (*Semaphore) unlink() error {
}

func (*Semaphore) close() error {
}

type SharedMemory struct {
}

func (*SharedMemory) open(name string, flag int, mode os.FileMode) error {
}

func (*SharedMemory) unlink() error {
}

func (*SharedMemory) close() error {
}
