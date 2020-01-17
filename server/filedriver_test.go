package server

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"
)

func Test_Lstat(t *testing.T) {
	filename := "D:\\work-go\\chillon\\upload\\Scan_001.fls\\Main"
	//filename := "D:\\work-go\\chillon\\upload\\Scan_001.fls\\Scan_001.fls"
	info, err := os.Lstat(filename)
	if err != nil{
		fmt.Println("os.Stat err =", err)
		return
	}

	fmt.Println("name =",info.Name())
	fmt.Println("size =",info.Size())
	fmt.Println("mode =",info.Mode())
	fmt.Println("modtime =",info.ModTime())
	fmt.Println("isDir =",info.IsDir())
	fmt.Println("sys =",info.Sys())

	//jsonStr, err := json.Marshal(info)
	jsonStr, err := json.MarshalIndent(info, "", "\t")
	fmt.Println(string(jsonStr))
}
