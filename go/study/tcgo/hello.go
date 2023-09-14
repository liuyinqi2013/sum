package tcgo

/*
#cgo CFLAGS: -I./
#cgo LDFLAGS: -lpthread -lz

#include <zlib.h>
#include <stdio.h>
#include <stdlib.h>
#include <pthread.h>
#include <string.h>

typedef void* (*ThreadFun)(void*);
extern void go_hello_callback(char *s);

static void hello_cgo(char *s)
{
	go_hello_callback(s);
	printf("hello cgo %s.\n", s);
}

static void create_thread(unsigned int num)
{
	int i = 0;
	pthread_t* arr = malloc(num * sizeof(pthread_t));
	for(i = 0; i < num; i++) {
		pthread_create(&arr[i], NULL, (ThreadFun)hello_cgo, "xxx");
	}

	for(i = 0; i < num; i++) {
		pthread_join(arr[i], NULL);
	}
	free(arr);
}

*/
import "C"
import (
	"errors"
	"fmt"
	"unsafe"
)

func Hello() {
	cs := C.CString("hello from stdio")
	C.hello_cgo(cs)
	C.free(unsafe.Pointer(cs))
}

func Random() int {
	return int(C.random())
}

func Seed(i int) {
	C.srandom(C.uint(i))
}

func Thread() {
	C.create_thread(10)
}

func Print(s string) {
	cs := C.CString(s)
	C.fputs(cs, (*C.FILE)(C.stdout))
	C.free(unsafe.Pointer(cs))
}

//export go_hello_callback
func go_hello_callback(s *C.char) {
	fmt.Printf("go hello:%v\n", C.GoString(s))
}

func ZlibCompress(src []byte) (out []byte, err error) {
	srcBuf := C.CBytes(src)
	srcLen := C.size_t(len(src))
	dstLen := 2 * srcLen
	defer C.free(srcBuf)

	dstBuf := C.malloc(srcLen)
	defer C.free(dstBuf)

	fmt.Printf("dstLen:%v. srcLen:%v\n", dstLen, srcLen)
	ret := C.compress((*C.uchar)(dstBuf), &dstLen, (*C.uchar)(srcBuf), srcLen)
	if ret != 0 {
		err = errors.New("zlib compress failed")
		fmt.Printf("zlib compress failed. err:%v, ret:%v\n", err, ret)
		return
	}
	fmt.Printf("dstLen:%v. srcLen:%v\n", dstLen, srcLen)
	out = C.GoBytes(dstBuf, C.int(dstLen))
	return
}

func ZlibUnCompress(src []byte) (out []byte, err error) {
	srcBuf := C.CBytes(src)
	srcLen := C.size_t(len(src))
	dstLen := 4 * srcLen
	defer C.free(srcBuf)

	dstBuf := C.malloc(srcLen)
	defer C.free(dstBuf)

	fmt.Printf("dstLen:%v. srcLen:%v\n", dstLen, srcLen)
	ret := C.uncompress((*C.uchar)(dstBuf), &dstLen, (*C.uchar)(srcBuf), srcLen)
	if ret != 0 {
		err = errors.New("zlib compress failed")
		fmt.Printf("zlib uncompress failed. err:%v, ret:%v\n", err, ret)
		return
	}
	fmt.Printf("dstLen:%v. srcLen:%v\n", dstLen, srcLen)
	out = C.GoBytes(dstBuf, C.int(dstLen))
	return
}
