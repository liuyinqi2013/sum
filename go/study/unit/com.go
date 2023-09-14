package unit

import (
	"bytes"
	"compress/zlib"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/nsqio/go-nsq"
	"gl.zego.im/goserver/study/tcgo"
)

func zlibCompress(src []byte) ([]byte, error) {
	var in bytes.Buffer
	w := zlib.NewWriter(&in)

	_, err := w.Write(src)
	if err != nil {
		w.Close()
		return nil, err
	}
	w.Close()

	return in.Bytes(), nil
}

func zlibUnCompress(src []byte) ([]byte, error) {
	b := bytes.NewReader(src)
	r, err := zlib.NewReader(b)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	var out bytes.Buffer
	_, err = io.Copy(&out, r)
	if err != nil {
		return nil, err
	}

	return out.Bytes(), nil
}

func verifyMapGet() {
	type panda struct {
		name string
		age  int
	}
	m := map[string]map[string]*panda{}
	p := m["aa"]
	log.Printf("[I] ---p:%v", p)
	for k, v := range p {
		log.Printf("[I] ---k:%v, v:%v", k, v)
	}

	if _, ok := p["panda"]; !ok {
		log.Printf("[I] not find panda")
	}
}

func verifyMapDelete() {
	m := map[int]int{
		0: 0,
		1: 1,
		2: 2,
		3: 3,
		4: 4,
		5: 5,
	}
	for k, v := range m {
		if k%2 == 0 {
			delete(m, k)
		} else {
			log.Printf("[I] k:%v, v:%v", k, v)
		}
	}
	for k, v := range m {
		log.Printf("[I] k:%v, v:%v", k, v)
	}
}

func verifyAppendSlice() {
	a := []string{"1", "2", "3"}
	b := []string{"4", "5", "6"}
	total := []string{}
	total = append(total, a...)
	total = append(total, b...)
	for _, v := range total {
		log.Printf("[I] AppendSlice: %v", v)
	}

	log.Printf("[I] other: %v", total)
	log.Printf("[I] a[0]: %v", a[0])
}

func verifyZlib() {
	data, err := zlibCompress([]byte("go hello zlib"))
	if err != nil {
		log.Printf("go zlib compress failed")
		return
	}
	log.Printf("go zlib compress data:%v\n", data)

	data2, err := zlibUnCompress(data)
	if err != nil {
		log.Printf("go zlib uncompress failed")
		return
	}
	log.Printf("go zlib uncompress data:%v\n", string(data2))

	data1, err := tcgo.ZlibUnCompress(data)
	if err != nil {
		log.Printf("zlib uncompress failed")
		return
	}
	log.Printf("zlib uncompress data:%v\n", string(data1))
}

func verifyThread() {
	tcgo.Thread()
}

func recoverPanic() {
	defer func() {
		if err := recover(); err != nil {
			log.Printf("[E] panic info is: %v", err)
		}
	}()

	go func() {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("[E] panic info is: %v", err)
			}
		}()
		panic("abcdefg")
	}()
}

func kmpSearch(s string, sub string) int {
	n := len(sub)
	m := len(s)
	if n == 0 {
		return 0
	} else if m == 0 || n > m {
		return -1
	}

	var i, j int
	next := kmpNext(sub)
	for i < m {
		switch {
		case n == j:
			return i - j
		case s[i] == sub[j]:
			i++
			j++
		case j > 0:
			j = next[j-1]
			if s[i] == sub[j] {
				j++
			} else {
				j = 0
			}
			i++
		default:
			i++
		}
	}
	if n == j {
		return i - j
	}
	return -1
}

func kmpNext(s string) []int {
	next := make([]int, len(s))
	if len(s) == 0 {
		return next
	}
	for i := 1; i < len(s); i++ {
		if s[i] == s[next[i-1]] {
			next[i] = next[i-1] + 1
		}
	}
	return next
}

func kmpTest() {
	fmt.Println(kmpSearch("abchabc", "aa"))                                       // -1
	fmt.Println(kmpSearch("abchabc", "aba"))                                      // -1
	fmt.Println(kmpSearch("abchabcabaadc", "aba"))                                // 7
	fmt.Println(kmpSearch("abaabababcdabd", "abcabcabc"))                         // -1
	fmt.Println(kmpSearch("abaabababcabcab abcabcabcdabd", "abcabcabc"))          // 16
	fmt.Println(kmpSearch("abcabcabcabaabababcabcab abcabcabcdabd", "abcabcabc")) // 0
	fmt.Println(kmpSearch("abcabcabc", "abcabcabc"))                              // 0
	fmt.Println(kmpSearch("aaabcdefg", "abcdefg"))                                // 2
	fmt.Println(kmpSearch("aaababdbdacdefg", "abdbda"))                           // 4
}

func emptySlice() {
	var a []int
	a = append(a, 1) //  合法
	for k, v := range a {
		fmt.Printf("k:%v, v:%v\n", k, v)
	}

	var b []string = nil
	b = append(b, "abc") //  合法
	for k, v := range b {
		fmt.Printf("k:%v, v:%v\n", k, v)
	}

	m := make([]int, 0, 100)
	fmt.Printf("len(m):%v, cap(m):%v\n", len(m), cap(m))
	n := m[:50]
	fmt.Printf("len(n):%v, cap(n):%v\n", len(n), cap(n))
}

type tidy interface {
	show()
}

type panda struct {
	name int
}

func (p *panda) show() {
	fmt.Printf("i am %v\n", p.name)
}

func shows(v interface{}) {
	s, ok := v.([]interface{})
	if !ok {
		fmt.Printf("not tidy slice\n")
	}
	for _, t := range s {
		m := t.(tidy)
		m.show()
	}
}

func verifyTidy() {
	s := make([]*panda, 0, 10)
	for i := 0; i < 10; i++ {
		s = append(s, &panda{
			name: i + 1,
		})
	}
	shows(s)
}

type Persion struct {
	Id   int    `json:"uid"`
	Name string `json:"name"`
}

type Student struct {
	Persion
	Class  int    `json:"class"`
	Number string `json:"number"`
}

func verifyJson() {
	str := `{"uid":1, "name":"laoda", "class":"xxx", "number":100}`

	var student Student
	json.Unmarshal([]byte(str), &student)

	fmt.Printf("student:%+v\n", student.Name)
}

type myMessageHandler struct{}

func (h *myMessageHandler) HandleMessage(m *nsq.Message) error {
	if len(m.Body) == 0 {
		return nil
	}
	log.Printf("handler message:%v", string(m.Body))
	return nil
}

func nsqConsumer() {
	config := nsq.NewConfig()
	consumer, err := nsq.NewConsumer("wow", "channel", config)
	if err != nil {
		log.Fatal(err)
	}
	consumer.AddHandler(&myMessageHandler{})
	err = consumer.ConnectToNSQLookupd("localhost:4161")
	if err != nil {
		log.Fatal(err)
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan
	consumer.Stop()
}

func nsqProducer() {
	config := nsq.NewConfig()
	producer, err := nsq.NewProducer("127.0.0.1:4150", config)
	if err != nil {
		log.Fatal(err)
	}

	messageBody := []byte("hello xxxxx")
	topicName := "wow"

	err = producer.Publish(topicName, messageBody)
	if err != nil {
		log.Fatal(err)
	}
	producer.Stop()
}

func Excute() {
	kmpTest()
	//nsqProducer()
	// nsqConsumer()

	/*
		verifyJson()
		emptySlice()
		verifyTidy()
		recoverPanic()
		verifyThread()
		verifyZlib()
		verifyMapGet()
		verifyAppendSlice()
		verifyMapDelete()
		tdb.SQLite3Test()
	*/
}
