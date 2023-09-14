package tnet

import (
	"fmt"
	"io"
	"net"
)

func Server(addr string) error {
	laddr, _ := net.ResolveTCPAddr("tcp", addr)
	listen, err := net.ListenTCP("tcp", laddr)
	if err != nil {
		fmt.Printf("listenTCP  error:%v\n", err.Error())
		return err
	}

	for {
		conn, err := listen.Accept()
		if err != nil {
			fmt.Printf("AcceptTCP  error:%v\n", err.Error())
			return err
		}

		go func() {
			for {
				buf := make([]byte, 5)
				n, err := io.ReadFull(conn, buf)
				if err != nil {
					fmt.Printf("ReadFull  error:%v, n:%v\n", err.Error(), n)
					return
				}
			}
		}()
	}
}
