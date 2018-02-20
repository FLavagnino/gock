package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"

	"math/rand"

	"github.com/gin-gonic/gin"
)

type Uri = string
type Method = string
type controllers = map[Uri]map[Method][]*data
type params map[string]string

type data struct {
	params
	responseHeaders map[string]string
	delayInMs       time.Duration
	deltaDelay      int
	response        []byte
}

type fileData struct {
	Uri
	Method
	Params          map[string]string
	ResponseHeaders map[string]string `json:"response_headers"`
	MaxDelayInMs    time.Duration     `json:"max_delay_in_ms"`
	MinDelayInMs    time.Duration     `json:"min_delay_in_ms"`
	Response        interface{}
}

const jsonContentType = "application/json; charset=utf-8"

func main() {
	var filePath, port string
	flag.StringVar(&filePath, "f", "/home/facu/mockk.json", "Data file path")
	flag.StringVar(&port, "p", "9292", "Server port")
	flag.Parse()

	fds, err := readDataFile(filePath)
	if err != nil {
		panic(err)
	}

	cs, err := toControllers(fds)
	if err != nil {
		panic(err)
	}

	gin.SetMode(gin.ReleaseMode)
	e := gin.New()

	for uri, vs := range cs {
		for method, ds := range vs {
			fmt.Printf("%s: %s\n", method, uri)

			switch method {
			case "GET":
				e.GET(uri, func(c *gin.Context) { handleReq(c, ds) })
			case "POST":
				e.POST(uri, func(c *gin.Context) { handleReq(c, ds) })
			}
		}
	}

	fmt.Printf("Listen on port %s\n", port)
	e.Run(fmt.Sprintf(":%s", port))
}

func handleReq(c *gin.Context, ds []*data) {
	for _, d := range ds {
		if d.params.equalsTo(c.Request.URL.Query()) {
			fmt.Printf("%s %s: %s\n", time.Now().Format("2006-01-02T15:04:05.999"), c.Request.Method, c.Request.RequestURI)

			var delta time.Duration
			if d.deltaDelay > 0 {
				delta = time.Duration(rand.Intn(d.deltaDelay))
			}
			time.Sleep((d.delayInMs + delta) * time.Millisecond)

			for k, v := range d.responseHeaders {
				c.Header(k, v)
			}

			c.Data(http.StatusOK, jsonContentType, d.response)
			return
		}
	}
	c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"Error": "uri not mapped"})
}

func readDataFile(filePath string) (fds []fileData, err error) {
	if filePath == "" {
		return nil, errors.New("missing data file")
	}

	fmt.Printf("Reading data file [%s]\n", filePath)
	raw, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	if err = json.Unmarshal(raw, &fds); err != nil {
		return nil, err
	}

	for i, v := range fds {
		if v.Method == "" {
			fds[i].Method = "GET"
		}
	}

	return fds, nil
}

func toControllers(fds []fileData) (cs controllers, err error) {
	cs = make(controllers)
	for _, fd := range fds {
		d, err := toData(fd)
		if err != nil {
			return nil, err
		}

		if x, ok := cs[fd.Uri]; ok {
			if y, ok := x[fd.Method]; ok {
				cs[fd.Uri][fd.Method] = append(y, d)
			} else {
				x[fd.Method] = []*data{d}
			}
		} else {
			cs[fd.Uri] = make(map[Method][]*data)
			cs[fd.Uri][fd.Method] = []*data{d}
		}
	}
	return
}

func toData(fd fileData) (*data, error) {
	bs, err := json.Marshal(fd.Response)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("cannot marshal response %v", err))
	}

	var deltaDelay int
	if d := fd.MaxDelayInMs - fd.MinDelayInMs; d > 0 {
		deltaDelay = int(d)
	}

	return &data{
		params:          fd.Params,
		responseHeaders: fd.ResponseHeaders,
		delayInMs:       fd.MinDelayInMs,
		deltaDelay:      deltaDelay,
		response:        bs,
	}, nil
}

func (ps params) equalsTo(reqParams url.Values) bool {
	if len(ps) != len(reqParams) {
		return false
	}
	for h, v1 := range ps {
		v2 := reqParams.Get(h)
		if v1 != v2 {
			return false
		}
	}
	return true
}
