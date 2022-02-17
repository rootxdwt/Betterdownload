package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	neturl "net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	TempFileNameStrLength = 12   //length of the random name of the temporary file used in the download progress
	FilePermission        = 0777 //the numeric chmod parameter for the generated file
)

const (
	B  = 1
	KB = 1024 * B
	MB = 1024 * KB
	GB = 1024 * MB
	TB = 1024 * GB
)

var tempFileName = randStr(TempFileNameStrLength)

func main() {

	var url string
	fmt.Println("Input File Link")
	fmt.Scanf("%s", &url)

	req, e := http.NewRequest("", url, nil)

	if e != nil {
		errorHandler(e)
	}

	client := &http.Client{}
	resp, e := client.Do(req)
	if e != nil {
		errorHandler(e)
	}
	resp.Body.Close()

	contentLength, _ := strconv.Atoi(resp.Header.Get("Content-Length"))

	fmt.Printf("Found File with length %d\n", contentLength)
	fmt.Println("Writing To:" + tempFileName)
	if e := ioutil.WriteFile(tempFileName, nil, FilePermission); e != nil {
		errorHandler(e)
	}

	chunkSlice := calcChunk(contentLength)
	var wg sync.WaitGroup

	howManyChunks := len(chunkSlice)
	switch {
	case howManyChunks <= 5:
		for i := 0; i < len(chunkSlice); i++ {
			fmt.Printf("Starting Worker %d\n", i)
			wg.Add(1)
			go getAndWriteChunk(url, []int{chunkSlice[0] * i, chunkSlice[0]*i + chunkSlice[i]}, tempFileName, contentLength, &wg)
		}
		wg.Wait()

	case howManyChunks > 5:
		var i int
		for v := 0; v < howManyChunks/5; v++ {
			for i = i; i < 5*v; i++ {
				fmt.Printf("Starting Worker %d\n", i)
				wg.Add(1)
				go getAndWriteChunk(url, []int{chunkSlice[0] * i, chunkSlice[0]*i + chunkSlice[i]}, tempFileName, contentLength, &wg)
			}
			wg.Wait()
		}
	}

	filename, _ := neturl.Parse(url)
	e = os.Rename(tempFileName, strings.Replace(filename.Path, "/", "-", -1))
	if e != nil {
		errorHandler(e)
	}
	fmt.Printf("Done")
}

func errorHandler(e error) {
	os.Remove(tempFileName)
	fmt.Printf("Aborting because an error occured: %s \n", e.Error())
	os.Exit(1)
}

func randStr(length int) string {
	var returnVal string
	chars := "ABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890abcdefghijklmnopqrstuvwxyz"
	rand.Seed(time.Now().UnixNano())
	for i := 0; i < length; i++ {
		ri := rand.Intn(62)
		returnVal = returnVal + chars[ri:ri+1]
	}
	return returnVal
}

func calcChunk(BinarySize int) []int {
	var returnVal []int
	switch {
	case BinarySize < 5*MB:
		returnVal = []int{BinarySize}
	case BinarySize < 100*MB:
		returnVal = devideTo(BinarySize, 5)
	case BinarySize < 5*GB:
		returnVal = devideTo(BinarySize, 20)
	case BinarySize < 15*GB:
		returnVal = devideTo(BinarySize, 40)
	case BinarySize < 45*GB:
		returnVal = devideTo(BinarySize, 80)
	}
	return returnVal
}

func devideTo(size int, n int) []int {
	var returnVal []int
	for i := 0; i < n; i++ {
		returnVal = append(returnVal, (size-(size%n))/n)
	}
	returnVal = append(returnVal, (size % n))
	return returnVal
}

func getAndWriteChunk(url string, position []int, fileLocation string, fullFileSize int, wg *sync.WaitGroup) {
	defer wg.Done()

	fmt.Printf("Started Downloading For byte range %d to %d\n", position[0], position[1])
	client := &http.Client{}
	req, e := http.NewRequest("GET", url, nil)

	req.Header.Add("Range", "bytes="+strconv.Itoa(position[0])+"-"+strconv.Itoa(position[1]))
	if e != nil {
		errorHandler(e)
	}
	resp, e := client.Do(req)
	if e != nil {
		errorHandler(e)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 206 {
		errorHandler(errors.New("HTTP Range Header unsupported"))
	}
	if e != nil {
		errorHandler(e)
	}
	filedata, e := os.OpenFile(fileLocation, os.O_RDWR, 0777)
	if e != nil {
		errorHandler(e)
	}
	fb, e := ioutil.ReadAll(resp.Body)
	if e != nil {
		errorHandler(e)
	}
	_, e = filedata.WriteAt(fb, int64(position[0]))
	if e != nil {
		errorHandler(e)
	}

	if e != nil {
		errorHandler(e)
	}

	e = filedata.Close()
	if e != nil {
		errorHandler(e)
	}
	fmt.Printf("Finished\n")
}
