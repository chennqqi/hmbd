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
	"time"

	"github.com/gin-gonic/gin"
	"github.com/malice-plugins/go-plugin-utils/utils"
)

type Web struct {
	fileto time.Duration
	zipto  time.Duration
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

	}
	src, err := upf.Open()
	if err != nil {
		c.String(http.StatusBadRequest, fmt.Sprintf("get form err: %s", err.Error()))
		return
	}
	defer src.Close()
	tmpDir, err := ioutil.TempDir("/dev/shm", "file")
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

	r, _ := hmScanDir(f.Name(), to)
	//TODO: call hm scan dir
	c.JSON(200, r)
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
	f, err := ioutil.TempFile("/dev/shm/zip", "zip_")
	defer os.Remove(f.Name())
	if err != nil {
		c.String(http.StatusBadRequest, fmt.Sprintf("get form err: %s", err.Error()))
		return
	}
	io.Copy(f, src)
	f.Close()

	tmpDir, err := ioutil.TempDir("/dev/shm", "scan_")
	if err != nil {
		c.String(http.StatusInternalServerError,
			fmt.Sprintf("save zip file err: %s", err.Error()))
		return
	}

	if err = utils.Unzip(f.Name(), tmpDir); err != nil {
		c.String(http.StatusInternalServerError,
			fmt.Sprintf("unzip zip file err: %s", err.Error()))
		return
	}
	//TODO:
	r, err := hmScanDir(tmpDir, to)
	c.JSON(200, r)
}

func hmScanDir(dir string, to time.Duration) (string, error) {
	ctx, cancel := context.WithTimeout(context.TODO(), to)
	defer cancel()
	return utils.RunCommand(ctx, "hmb", "call", dir)
}
