package goafl

import (
	"sync"
	"testing"
	"time"
)

func testOneSem(t *testing.T, sem *semaphore, value int) {
	val := sem.getValue()
	t.Logf("Current value: %d\n", val)
	if val != value {
		t.Errorf("Semaphore init value is not %d!", value)
	}
	t.Log("Post semaphore.")
	sem.post()
	val = sem.getValue()
	t.Logf("Current value: %d\n", val)
	if val != value+1 {
		t.Errorf("Semaphore value should be %d but %d\n", value+1, val)
	}
	t.Log("Testing wait")
	sem.wait()
	time.Sleep(100 * time.Millisecond)
	val = sem.getValue()
	t.Logf("Current value: %d\n", val)
	if val != value {
		t.Errorf("Semaphore value should be %d but %d\n", value, val)
	}
	var wg sync.WaitGroup
	wg.Add(1)
	for sem.getValue() > 0 {
		sem.wait()
	}
	t.Logf("Semaphore decrease to 0")
	go func() {
		t.Logf("Start post")
		time.Sleep(100 * time.Millisecond)
		sem.post()
	}()
	t.Logf("Start wait")
	sem.wait()
	t.Logf("Wait finish")
}

func TestSemaphore(t *testing.T) {
	ipc, err := SetupIPC()
	if err != nil {
		t.Fatalf("Setup IPC error: %s\n", err)
	}
	t.Log("Testing global semaphore...")
	testOneSem(t, ipc.gloSem, 1)
	t.Log("Testing operator semaphore...")
	testOneSem(t, ipc.opSem, 1)
	t.Log("Testing data semaphore...")
	testOneSem(t, ipc.dataSem, 0)
}

func TestShm(t *testing.T) {
	ipc, err := SetupIPC()
	if err != nil {
		t.Fatalf("Setup IPC error: %s\n", err)
	}
	t.Log("Testing shared memory")
	ipc.shm.writebyte(0, 10)
	ipc.shm.writebyte(1, 20)
	ipc.shm.writebyte(2, 30)
	if ipc.shm.readbyte(0) != 10 {
		t.Errorf("Offset 0 should be 10 but %d", ipc.shm.readbyte(0))
	}
	if ipc.shm.readbyte(1) != 20 {
		t.Errorf("Offset 1 should be 20 but %d", ipc.shm.readbyte(1))
	}
	if ipc.shm.readbyte(2) != 30 {
		t.Errorf("Offset 2 should be 30 but %d", ipc.shm.readbyte(2))
	}
}
