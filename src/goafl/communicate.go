package goafl

/*
#cgo linux LDFLAGS: -lrt
#include <stdlib.h>
#include <semaphore.h>
#include <sys/mman.h>
#include <unistd.h>


sem_t *open_sem(const char *name, int flag, mode_t mode, unsigned int value) {
	return sem_open(name, flag, mode, value);
}

void *add(void *ptr, int offset) {
	return ptr + offset;
}

int readbyte(void *ptr, int offset) {
	return (int)*((unsigned char *)ptr + offset);
}

void writebyte(void *ptr, int offset, unsigned char value) {
	*((unsigned char *)ptr + offset) = value;
}

int go_ftruncate(int fd, int off) {
	return ftruncate(fd, (off_t)off);
}
*/
import "C"
import (
	"log"
	"os"
	"unsafe"
)

type semaphore struct {
	name string
	sem  *C.sem_t
}

func openSem(name string, flag int, mode os.FileMode, value int) (*semaphore, error) {
	namePtr := C.CString(name)
	defer C.free(unsafe.Pointer(namePtr))
	semPtr, err := C.open_sem(namePtr, C.int(flag), C.mode_t(mode), C.uint(value))
	if err != nil {
		return nil, err
	}
	sem := new(semaphore)
	sem.sem = semPtr
	sem.name = name
	return sem, nil
}

func initSem(psem unsafe.Pointer, value int) (*semaphore, error) {
	_, err := C.sem_init((*C.sem_t)(psem), 1, C.uint(value))
	if err != nil {
		return nil, err
	}
	sem := new(semaphore)
	sem.sem = (*C.sem_t)(psem)
	return sem, nil
}

func (sem *semaphore) getValue() (int, error) {
	var val C.int
	_, err := C.sem_getvalue(sem.sem, &val)
	if err != nil {
		return 0, err
	}
	return int(val), nil
}

func (sem *semaphore) post() error {
	_, err := C.sem_post(sem.sem)
	return err
}

func (sem *semaphore) wait() error {
	_, err := C.sem_wait(sem.sem)
	return err
}

func (sem *semaphore) unlink() error {
	namePtr := C.CString(sem.name)
	defer C.free(unsafe.Pointer(namePtr))
	_, err := C.sem_unlink(namePtr)
	return err
}

func (sem *semaphore) close() error {
	_, err := C.sem_close(sem.sem)
	return err
}

type sharedmemory struct {
	name   string
	flag   int
	mode   os.FileMode
	length int
	fd     C.int
	ptr    unsafe.Pointer
}

func openShm(name string, flag int, mode os.FileMode, length int) (*sharedmemory, error) {
	shm := new(sharedmemory)
	namePtr := C.CString(name)
	defer C.free(unsafe.Pointer(namePtr))
	fd, err := C.shm_open(namePtr, C.int(flag), C.mode_t(0755))
	if err != nil {
		log.Fatalf("shm_open fail: %s!", err)
		return nil, err
	}

	_, err = C.go_ftruncate(fd, C.int(length))
	if err != nil {
		log.Fatalf("ftruncate fail: %s!", err)
		return nil, err
	}

	ptr, err := C.mmap(nil, C.size_t(length),
		C.PROT_READ|C.PROT_WRITE, C.MAP_SHARED, fd, 0)

	if err != nil {
		log.Fatalln("mmap fail!")
		return nil, err
	}

	shm.name = name
	shm.flag = flag
	shm.mode = mode
	shm.length = length
	shm.ptr = ptr
	shm.fd = fd
	return shm, nil
}

func (shm *sharedmemory) read(offset int, length int) []byte {
	if offset > shm.length || offset < 0 || length <= 0 {
		return nil
	}

	if offset+length > shm.length {
		length = shm.length - offset
	}

	ptr := C.add(shm.ptr, C.int(offset))

	return C.GoBytes(ptr, C.int(length))
}

func (shm *sharedmemory) readbyte(offset int) int {
	if offset > shm.length || offset < 0 {
		return 0
	}

	return int(C.readbyte(shm.ptr, C.int(offset)))
}

func (shm *sharedmemory) writebyte(offset int, value byte) {
	if offset > shm.length || offset < 0 {
		return
	}

	C.writebyte(shm.ptr, C.int(offset), C.uchar(value))
}

func (shm *sharedmemory) unlink() error {
	namePtr := C.CString(shm.name)
	defer C.free(unsafe.Pointer(namePtr))
	_, err := C.shm_unlink(namePtr)
	return err
}

// FuzzIPC is used to communicate with afl process
// sharedmemory store the semaphore, operation and data
// There are two operation currently, read bitmap and write bitmap
// the byte after sizeof(sem_t) * 3 store the command
// the bytes after sizeof(sem_t) * 3 / 8 * 8 store data
// readBitMap	0
// writeBitMap	1
type FuzzIPC struct {
	shm        *sharedmemory
	gloSem     *semaphore
	opSem      *semaphore
	dataSem    *semaphore
	opOffset   int
	dataOffset int
	bitmapLen  int
}

const semLen int = int(unsafe.Sizeof(C.sem_t{}))
const shmLen int = (1 << 16) + semLen*3 + 8
const opOffset int = semLen * 3
const dataOffset int = semLen * 3 / 8 * 8

func SetupIPC() (*FuzzIPC, error) {
	var err error
	ipc := new(FuzzIPC)
	ipc.shm, err = openShm("FuzzShm", int(os.O_CREATE|os.O_RDWR), os.FileMode(0600), shmLen)
	if err != nil {
		log.Fatalf("Setup shared memory fail: %s!", err)
		return nil, err
	} else {
		log.Println("Setup shared memory successfully")
	}

	ipc.gloSem, err = initSem(ipc.shm.ptr, 1)
	if err != nil {
		log.Fatalf("Setup global semaphore fail: %s!", err)
		return nil, err
	} else {
		log.Println("Setup global semaphore successfully")
	}

	ipc.opSem, err = initSem(C.add(ipc.shm.ptr, C.int(semLen)), 1)
	if err != nil {
		log.Fatalf("Setup operation semaphore fail: %s!", err)
		return nil, err
	} else {
		log.Println("Setup operation semaphore successfully")
	}

	ipc.dataSem, err = initSem(C.add(ipc.shm.ptr, C.int(semLen*2)), 0)
	if err != nil {
		log.Fatalf("Setup data semaphore fail: %s!", err)
		return nil, err
	} else {
		log.Println("Setup data semaphore successfully")
	}

	ipc.opOffset = opOffset
	ipc.dataOffset = dataOffset
	ipc.bitmapLen = 1 << 16

	log.Println("Setup ipc successfully")
	return ipc, nil
}

func (ipc *FuzzIPC) GetBitMap() ([]byte, error) {
	ipc.gloSem.wait()
	ipc.shm.writebyte(ipc.opOffset, 0)
	ipc.opSem.post()
	ipc.dataSem.wait()
	bitmap := ipc.shm.read(ipc.dataOffset, ipc.bitmapLen)
	return bitmap, nil
}

func (ipc *FuzzIPC) SetBitMap([]byte) error {
	return nil
}
