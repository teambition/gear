package logging

import (
	"os"
	"log"
	"fmt"
	"time"
	"strconv"
	"math/rand"
)

/**
* 
* @author willian
* @created 2017-03-08 21:27
* @email 18702515157@163.com  
**/

//create a dir that every child of the dir will have same permission mode.
func mkdirlog(dir string) (e error) {
	_, er := os.Stat(dir)
	b := er == nil || os.IsExist(er)
	if !b {
		if err := os.MkdirAll(dir, os.ModePerm); err != nil {
			if os.IsPermission(err) {
				log.Fatalln("create dir error:", err.Error())
			}
		}
	}
	return nil
}


// fileSize return the file size
func fileSize(file string) int64 {
	f, e := os.Stat(file)
	if e != nil {
		fmt.Println(e.Error())
		return 0
	}
	return f.Size()
}

func (f *Logger) changeFile(){

	now := time.Now().Format("2006-01-02")
	f.mu.Lock()
	defer f.mu.Unlock()
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	rand_number := r.Intn(10)
	f.filename = now + strconv.Itoa(rand_number)
	f.logfile, _ = os.OpenFile(f.dir+"/"+f.filename+".log", os.O_RDWR|os.O_APPEND|os.O_CREATE, os.ModePerm)
	f.Out = f.logfile
}