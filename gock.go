package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"time"

	"os"

	"net/http"

	"errors"

	"github.com/gin-gonic/gin"
)

type fileData struct {
	Uri      string
	Method   string
	Headers  map[string]string
	Delay    time.Duration
	Response interface{}
}

var ds = make([]fileData, 1)

func main() {
	readDataFile(os.Args[2])

	gin.SetMode(gin.ReleaseMode)
	e := gin.New()
	e.GET("*uri", func(c *gin.Context) { handleReq(c, "GET") })
	e.POST("*uri", func(c *gin.Context) { handleReq(c, "POST") })

	port := os.Args[1]
	fmt.Printf("Listen on port %s\n", port)
	for _, d := range ds {
		fmt.Printf("%s: %s\n", d.Method, d.Uri)
	}

	e.Run(fmt.Sprintf(":%s", port))
}

func handleReq(c *gin.Context, method string) {
	uri := c.Request.RequestURI
	d, err := getData(method, uri)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"Error": "Unknown uri"})
		return
	}

	fmt.Printf("%s %s: %s\n", time.Now().Format("2006-01-02T15:04:05.999"), d.Method, d.Uri)

	time.Sleep(d.Delay * time.Millisecond)

	for k, v := range d.Headers {
		c.Header(k, v)
	}
	c.JSON(http.StatusOK, &d.Response)
}

func getData(method string, uri string) (fileData, error) {
	for _, d := range ds {
		if d.Uri == uri && d.Method == method {
			return d, nil
		}
	}
	return fileData{}, errors.New("URI not found")
}

func readDataFile(filePath string) {
	fmt.Printf("Reading file [%s]\n", filePath)
	raw, err := ioutil.ReadFile(filePath)
	if err != nil {
		panic(err)
	}

	if err = json.Unmarshal(raw, &ds); err != nil {
		panic(err)
	}

	for i, v := range ds {
		if v.Method == "" {
			ds[i].Method = "GET"
		}
	}
}
