package main

import (
	"archive/zip"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/chennqqi/goutils/utils"
	"github.com/gin-gonic/gin"
	mutils "github.com/malice-plugins/go-plugin-utils/utils"
)

type Web struct {
	fileto   time.Duration
	zipto    time.Duration
	callback string
}

func (s *Web) version(c *gin.Context) {
	txt, _ := ioutil.ReadFile("/opt/hmb/VERSION")
	c.Data(200, "", txt)
}

func (s *Web) scanFile(c *gin.Context) {
	var err error
	to := s.zipto
	timeout, ok := c.GetQuery("timeout")
	if ok {
		to, err = time.ParseDuration(timeout)
		if err != nil {
			to = s.fileto
		}
	}

	upf, err := c.FormFile("filename")
	if err != nil {
		c.String(http.StatusBadRequest, fmt.Sprintf("get form err: %s", err.Error()))
		return
	}
	src, err := upf.Open()
	if err != nil {
		c.String(http.StatusBadRequest, fmt.Sprintf("open form err: %s", err.Error()))
		return
	}
	defer src.Close()
	tmpDir, err := ioutil.TempDir("/dev/shm", "file")
	defer os.Remove(tmpDir)
	if err != nil {
	}
	f, err := ioutil.TempFile(tmpDir, "scan_")
	if err != nil {
		c.String(http.StatusBadRequest, fmt.Sprintf("get form err: %s", err.Error()))
		return
	}
	defer os.Remove(f.Name())
	io.Copy(f, src)
	f.Close()

	r, _ := hmScanDir(tmpDir, to)
	//TODO: call hm scan dir
	c.Header("Content-type", "application/json")
	r1 := strings.Replace(r, tmpDir, "", -1)
	s.doCallback(c, r1)
	c.String(200, r1)
}

func Unzip(src, dest string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer func() {
		if err := r.Close(); err != nil {
			panic(err)
		}
	}()

	os.MkdirAll(dest, 0755)

	// Closure to address file descriptors issue with all the deferred .Close() methods
	extractAndWriteFile := func(f *zip.File) error {
		rc, err := f.Open()
		if err != nil {
			return err
		}
		defer func() {
			if err := rc.Close(); err != nil {
				panic(err)
			}
		}()

		path := filepath.Join(dest, f.Name)

		if f.FileInfo().IsDir() {
			os.MkdirAll(path, f.Mode())
		} else {
			os.MkdirAll(filepath.Dir(path), f.Mode())
			f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
			if err != nil {
				return err
			}
			defer func() {
				if err := f.Close(); err != nil {
					panic(err)
				}
			}()

			_, err = io.Copy(f, rc)
			if err != nil {
				return err
			}
		}
		return nil
	}

	for _, f := range r.File {
		err := extractAndWriteFile(f)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *Web) Run(port int) {
	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()
	r.POST("/zip", s.scanZip)
	r.POST("/file", s.scanFile)
	r.GET("/version", s.scanFile)
	r.Run(fmt.Sprintf(":%d", port))
}

func (s *Web) scanZip(c *gin.Context) {
	var err error
	upf, err := c.FormFile("zipname")
	if err != nil {
		c.String(http.StatusBadRequest, fmt.Sprintf("get form err: %s", err.Error()))
		return
	}
	to := s.zipto
	timeout, ok := c.GetQuery("timeout")
	if ok {
		to, err = time.ParseDuration(timeout)
		if err != nil {
			to = s.zipto
		}
	}

	src, err := upf.Open()
	if err != nil {
		c.String(http.StatusBadRequest, fmt.Sprintf("get form err: %s", err.Error()))
		return
	}
	defer src.Close()
	f, err := ioutil.TempFile("/dev/shm", "zip_")
	if err != nil {
		c.String(http.StatusBadRequest, fmt.Sprintf("new tmp file err: %s", err.Error()))
		return
	}
	defer os.Remove(f.Name())
	io.Copy(f, src)
	f.Close()

	tmpDir, err := ioutil.TempDir("/dev/shm", "scan_")
	if err != nil {
		c.String(http.StatusInternalServerError,
			fmt.Sprintf("save zip file err: %s", err.Error()))
		return
	}
	defer os.RemoveAll(tmpDir)

	if err = utils.UnzipSafe(f.Name(), tmpDir, 0); err != nil {
	//if err = utils.Unzip(f.Name(), tmpDir); err != nil {
		c.String(http.StatusInternalServerError,
			fmt.Sprintf("unzip zip file err: %s", err.Error()))
		return
	}

	//TODO:
	r, err := hmScanDir(tmpDir, to)
	c.Header("Content-type", "application/json")
	r1 := strings.Replace(r, tmpDir, "", -1)
	s.doCallback(c, r1)
	c.String(200, r1)
}

func (s *Web) doCallback(c *gin.Context, r string) {
	callback := c.Query("callback")
	if callback == "" {
		callback = s.callback
	}
	if callback != "" {
		go func(r string) {
			body := strings.NewReader(r)
			resp, err := http.Post(callback, "application/json", body)
			if err != nil{
				fmt.Printf("do callback(%v) error: %v\n", callback, err)
			}
			if resp.Body != nil{
				defer resp.Body.Close()
			}
		}(r)
	}
}

func hmScanDir(dir string, to time.Duration) (string, error) {
	fmt.Println("start scan ", dir)
	//	time.Sleep(time.Second*20)
	ctx, cancel := context.WithTimeout(context.TODO(), to)
	defer cancel()
	return mutils.RunCommand(ctx, "hmb", "call", dir)
}
