package tnet

import (
	"fmt"
	"io"
	"net"
)

func Client(addr string) error {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		fmt.Printf("Dial failed err:%v\n", err.Error())
		return err
	}

	conn.Close()
	n, err := io.WriteString(conn, "hello")
	if err != nil {
		fmt.Printf("Write string err:%v,n:%v\n", err.Error(), n)
	}

	n, err = io.WriteString(conn, "hello")
	if err != nil {
		fmt.Printf("Write string err:%v,n:%v\n", err.Error(), n)
	}

	n, err = io.WriteString(conn, "hello")
	if err != nil {
		fmt.Printf("Write string err:%v,n:%v\n", err.Error(), n)
	}

	return nil
}
