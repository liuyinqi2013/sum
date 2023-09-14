package main

import (
	"encoding/json"
	"fmt"

	"gl.zego.im/goserver/study/tdb"
	"gl.zego.im/goserver/study/unit"
)

type A struct {
	Name string
	List []uint8
}

func main() {
	unit.Excute()
	tdb.ConnectMySQL()

	a := A{
		Name: "panda",
		List: []byte{1, 2, 3},
	}

	data, _ := json.Marshal(a)
	fmt.Println(string(data))
}
