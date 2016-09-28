package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"strings"

	"github.com/asyou-me/lib.v1/utils"
)

const jsonStream = `
    {"Message": "Hello", "Array": [1, 2, 3], "Null": null, "Number": 1.234}
`

var jsonStreamB = []byte(`
    {"Message": "Hello", "Array": [1, 2, 3], "Null": null, "Number": 1.234}
`)

var jsonStreamC = strings.NewReader(jsonStream)

func name1() {
	jsonStreamC.Reset(jsonStream)
	var dec = json.NewDecoder(jsonStreamC)
	for {
		_, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal(err)
		}
		//fmt.Printf("%T: %v", t, t)
		if dec.More() {
			//fmt.Printf(" (more)")
		}
		//fmt.Printf("\n")
	}
}

type Test struct {
	Message string  `json:"Message"`
	Array   []int64 `json:"Array"`
	Number  float64
}

func name2() {
	v := map[string]interface{}{}
	json.Unmarshal(jsonStreamB, v)
	//fmt.Println(v)
}

func name3() {
	v := Test{}
	json.Unmarshal(jsonStreamB, v)
	//fmt.Println(v)
}

func main() {
	T1 := utils.Time_Time(func() {
		for i := 0; i < 100000; i++ {
			name1()
		}
	})
	fmt.Println("T1:", T1)
	T2 := utils.Time_Time(func() {
		for i := 0; i < 100000; i++ {
			name2()
		}
	})
	fmt.Println("T2:", T2)
	T3 := utils.Time_Time(func() {
		for i := 0; i < 100000; i++ {
			name2()
		}
	})
	fmt.Println("T3:", T3)
}
