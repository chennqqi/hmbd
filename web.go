package main

import (
	"archive/zip"
	"context"
	"encoding/json"
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

	"github.com/lunny/nodb"
	"github.com/lunny/nodb/config"
)

const (
	PERSIST_LISTKEY_NAME = "__HMBD_PERSIST"
)

type task struct {
	Dir      string `json:"dir"`
	Callback string `json:"callback"`
}

type Web struct {
	fileto   time.Duration
	zipto    time.Duration
	callback string

	db         *nodb.DB
	inst       *nodb.Nodb
	scanQuitCh chan struct{}
	server     *http.Server
}

func NewWeb(dir string) (*Web, error) {
	var web Web
	cfg := new(config.Config)
	cfg.DataDir = dir

	err := os.MkdirAll(cfg.DataDir, 0755)
	if !os.IsExist(err) && err != nil {
		fmt.Printf("mkdir leveldb dir failed, error: \n", err)
		return nil, err
	}

	dbs, err := nodb.Open(cfg)
	if err != nil {
		fmt.Printf("nodb: error opening db: %v", err)
		return nil, err
	}

	err = os.MkdirAll(cfg.DataDir, 0755)
	if !os.IsExist(err) && err != nil {
		fmt.Printf("mkdir leveldb dir failed, error: \n", err)
		return nil, err
	}
	db, _ := dbs.Select(0)

	web.scanQuitCh = make(chan struct{})
	web.db = db
	web.inst = dbs
	return &web, nil
}

func (s *Web) version(c *gin.Context) {
	txt, _ := ioutil.ReadFile("/malware/VERSION")
	c.Data(200, "", txt)
}

func (s *Web) queued(c *gin.Context) {
	db := s.db
	l, err := db.LLen([]byte(PERSIST_LISTKEY_NAME))
	if err != nil {
		c.String(400, "%v", err)
		return
	}
	c.String(200, "%d", l)
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
	tmpDir, err := ioutil.TempDir("/dev/shm", "file")
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
		r, _ := hmScanDir(tmpDir, to)
		c.Header("Content-type", "application/json")
		r1 := strings.Replace(r, tmpDir, "", -1)
		c.String(200, r1)
	} else {
		db := s.db

		var t task
		t.Dir = tmpDir
		t.Callback = callback
		txt, _ := json.Marshal(t)
		queued, err := db.LPush([]byte(PERSIST_LISTKEY_NAME), txt)
		if err != nil {
			c.JSON(http.StatusBadRequest, CR{
				1, fmt.Sprintf("new temp file err: %s", err.Error()),
			})
			return
		}
		c.JSON(200, CR{
			0, fmt.Sprintf("queued %d", queued),
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
	db := s.db
	ticker := time.NewTicker(500 * time.Second)
	defer ticker.Stop()

__FOR_LOOP:
	for {
		select {
		case <-ticker.C:
			txt, err := db.RPop([]byte(PERSIST_LISTKEY_NAME))
			if err != nil {
				continue
			}
			var t task
			err = json.Unmarshal(txt, &t)
			if err != nil {
				fmt.Println("json.Unmarshal Error:", err)
				continue
			}
			defer os.RemoveAll(t.Dir)
			r, err := hmScanDir(t.Dir, 0)
			if err != nil {
				fmt.Println("hmScanDir ERROR:", err)
				continue
			}
			s.doCallback(t.Callback, string(r))

		case <-ctx.Done():
			break __FOR_LOOP
		}
	}
	close(s.scanQuitCh)
}

func (s *Web) Shutdown(ctx context.Context) error {
	err := s.server.Shutdown(ctx)
	<-s.scanQuitCh
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

	callback, _ := c.GetQuery("callback")

	//TODO:
	r, err := hmScanDir(tmpDir, to)
	c.Header("Content-type", "application/json")
	r1 := strings.Replace(r, tmpDir, "", -1)
	s.doCallback(callback, r1)
	c.String(200, r1)
}

func (s *Web) doCallback(callback string, r string) {
	if callback != "" {
		go func(r, cb string) {
			body := strings.NewReader(r)
			resp, err := http.Post(callback, "application/json", body)
			if err != nil {
				fmt.Printf("do callback(%v) error: %v\n", cb, err)
				return
			}
			defer resp.Body.Close()
		}(r, callback)
	}
}

func hmScanDir(dir string, to time.Duration) (string, error) {
	fmt.Println("start scan ", dir)
	//	time.Sleep(time.Second*20)
	ctx, cancel := context.WithTimeout(context.TODO(), to)
	defer cancel()
	return mutils.RunCommand(ctx, "hmb", "call", dir)
}
