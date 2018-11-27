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
	"sync"
	"time"

	"github.com/chennqqi/goutils/persistlist"
	utime "github.com/chennqqi/goutils/time"
	"github.com/chennqqi/goutils/utils"

	"github.com/gin-gonic/gin"
	mutils "github.com/malice-plugins/go-plugin-utils/utils"
)

const (
	PERSIST_LISTKEY_NAME = "__HMBD_PERSIST"
)

type task struct {
	Dir      string         `json:"dir"`
	Callback string         `json:"callback"`
	To       utime.Duration `json:"to"`
}

type Web struct {
	fileto   utime.Duration
	zipto    utime.Duration
	callback string

	tmpDir string
	batch  int

	wg     sync.WaitGroup
	server *http.Server
	cancel context.CancelFunc

	list persistlist.PersistList
}

func NewWeb(dataDir, indexDir string, batch int) (*Web, error) {
	var web Web

	list, err := persistlist.NewNodbList(indexDir, PERSIST_LISTKEY_NAME)
	if err != nil {
		return nil, err
	}

	err = os.MkdirAll(dataDir, 0755)
	if !os.IsExist(err) && err != nil {
		fmt.Printf("mkdir tmp dir error: \n", err)
		return nil, err
	}

	web.tmpDir = dataDir
	web.batch = 1
	if batch > 1 {
		web.batch = batch
	}
	web.list = list
	return &web, nil
}

func (s *Web) version(c *gin.Context) {
	txt, _ := ioutil.ReadFile("/malware/VERSION")
	c.Data(200, "", txt)
}

func (s *Web) queued(c *gin.Context) {
	list := s.list
	l, err := list.Len()
	if err != nil {
		c.String(400, "%v", err)
		return
	}
	c.String(200, "%d", l)
}

func (s *Web) scanFile(c *gin.Context) {
	var err error
	to := s.fileto

	timeout, ok := c.GetQuery("timeout")
	if ok {
		tto, err := time.ParseDuration(timeout)
		if err == nil {
			to = utime.Duration(tto)
		}
	}

	upf, err := c.FormFile("filename")
	if err != nil {
		c.JSON(http.StatusBadRequest, CR{
			1, err.Error(),
		})
		return
	}
	src, err := upf.Open()
	if err != nil {
		c.JSON(http.StatusBadRequest, CR{
			1, err.Error(),
		})
		return
	}
	defer src.Close()
	tmpDir, err := ioutil.TempDir(s.tmpDir, "file")
	if err != nil {
		c.JSON(http.StatusBadRequest, CR{
			1, fmt.Sprintf("new temp dir err: %s", err.Error()),
		})
		return
	}
	f, err := ioutil.TempFile(tmpDir, "scan_")
	if err != nil {
		c.JSON(http.StatusBadRequest, CR{
			1, fmt.Sprintf("new temp file err: %s", err.Error()),
		})
		return
	}
	io.Copy(f, src)
	f.Close()

	callback, _ := c.GetQuery("callback")
	if callback == "" {
		callback = s.callback
	}

	if callback == "" {
		defer os.RemoveAll(tmpDir)
		r, _ := hmScanDir(tmpDir, time.Duration(to))
		c.Header("Content-type", "application/json")
		r1 := strings.Replace(r, tmpDir, "", -1)
		c.String(200, r1)
	} else {
		var t task
		t.Dir = tmpDir
		t.Callback = callback
		t.To = to

		list := s.list
		pending, err := list.Push(t)

		if err != nil {
			c.JSON(500, CR{
				1, fmt.Sprintf("add task err: %s", err.Error()),
			})
			return
		}
		c.JSON(200, CR{
			0, fmt.Sprintf("pending %d", pending),
		})
	}
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

func (s *Web) scanRoute(ctx context.Context) {
	defer s.wg.Done()

	ticker := time.NewTicker(time.Second / 2)
	defer ticker.Stop()

	list := s.list

__FOR_LOOP:
	for {
		select {
		case <-ticker.C:
			for {
				var t task
				err := list.Pop(&t)
				if err == persistlist.ErrNil {
					break
				} else {
					fmt.Println("Pop Error:", err)
					break
				}

				r, err := hmScanDir(t.Dir, time.Duration(t.To))
				if err != nil {
					fmt.Println("hmScanDir ERROR:", err)
					err = os.RemoveAll(t.Dir)
					if err != nil {
						fmt.Println("RemoveAll ERROR:", err)
					}
					continue
				}
				err = os.RemoveAll(t.Dir)
				if err != nil {
					fmt.Println("RemoveAll ERROR:", err)
				}
				s.doCallback(t.Callback, string(r))
			}

		case <-ctx.Done():
			break __FOR_LOOP
		}
	}
}

func (s *Web) Shutdown(ctx context.Context) error {
	list := s.list
	err := s.server.Shutdown(ctx)
	s.cancel()
	list.Close()

	s.wg.Wait()
	return err
}

func (s *Web) Run(port int, ctx context.Context) error {
	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()
	r.POST("/zip", s.scanZip)
	r.POST("/file", s.scanFile)
	r.GET("/version", s.version)
	r.GET("/queued", s.queued)

	httpServer := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: r,
	}
	scanctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel

	for i := 0; i < s.batch; i++ {
		s.wg.Add(1)
		go s.scanRoute(scanctx)
	}

	s.server = httpServer
	return httpServer.ListenAndServe()
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
		tto, err := time.ParseDuration(timeout)
		if err == nil {
			to = utime.Duration(tto)
		}
	}

	src, err := upf.Open()
	if err != nil {
		c.String(http.StatusBadRequest, fmt.Sprintf("get form err: %s", err.Error()))
		return
	}
	defer src.Close()
	f, err := ioutil.TempFile(s.tmpDir, "zip_")
	if err != nil {
		c.String(http.StatusBadRequest, fmt.Sprintf("new tmp file err: %s", err.Error()))
		return
	}
	defer os.Remove(f.Name())
	io.Copy(f, src)
	f.Close()

	tmpDir, err := ioutil.TempDir(s.tmpDir, "scan_")
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

	callback, _ := c.GetQuery("callback")

	//TODO:
	r, err := hmScanDir(tmpDir, time.Duration(to))
	c.Header("Content-type", "application/json")
	r1 := strings.Replace(r, tmpDir, "", -1)
	s.doCallback(callback, r1)
	c.String(200, r1)
}

func (s *Web) doCallback(callback string, r string) {
	go func(r, cb string) {
		body := strings.NewReader(r)
		resp, err := http.Post(cb, "application/json", body)
		if err != nil {
			fmt.Printf("do callback(%v) error: %v\n", cb, err)
			return
		}
		io.Copy(ioutil.Discard, resp.Body)
		resp.Body.Close()
	}(r, callback)
}

func hmScanDir(dir string, to time.Duration) (string, error) {
	fmt.Println("start scan ", dir)
	//	time.Sleep(time.Second*20)
	ctx, cancel := context.WithTimeout(context.TODO(), to)
	defer cancel()
	return mutils.RunCommand(ctx, "hmb", "call", dir)
}
