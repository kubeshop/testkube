package transfer

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"

	"github.com/kubeshop/testkube/cmd/testworkflow-toolkit/common"
)

type server struct {
	files       map[string]struct{}
	requests    map[string]string
	storagePath string
	host        string
	port        int
}

type Server interface {
	Count() int
	RequestsCount() int
	Has(dirPath string, files []string) bool
	Request(dirPath string) RequestEntry
	Include(dirPath string, files []string) (Entry, error)
	Listen() (func(), error)
}

type Entry struct {
	Id  string `json:"id"`
	Url string `json:"url"`
}

type RequestEntry struct {
	Id  string `json:"id"`
	Url string `json:"url"`
}

func NewServer(storagePath string, host string, port int) Server {
	return &server{
		files:       make(map[string]struct{}),
		requests:    make(map[string]string),
		storagePath: storagePath,
		host:        host,
		port:        port,
	}
}

func (t *server) Count() int {
	return len(t.files)
}

func (t *server) RequestsCount() int {
	return len(t.requests)
}

func (t *server) Has(dirPath string, files []string) bool {
	_, ok := t.files[SourceID(dirPath, files)]
	return ok
}

func (t *server) GetUrl(id string) string {
	return fmt.Sprintf("http://%s:%d/download/%s.tar.gz", t.host, t.port, id)
}

func (t *server) GetRequestUrl(id string) string {
	return fmt.Sprintf("http://%s:%d/upload/%s", t.host, t.port, id)
}

func (t *server) Include(dirPath string, files []string) (Entry, error) {
	id := SourceID(dirPath, files)

	// Ensure that is not prepared already
	if _, ok := t.files[id]; ok {
		return Entry{Id: id, Url: t.GetUrl(id)}, nil
	}

	// Access the file on the disk
	fileStream, err := os.Create(filepath.Join(t.storagePath, fmt.Sprintf("%s.tar.gz", id)))
	if err != nil {
		return Entry{}, err
	}
	defer fileStream.Close()

	// Prepare files archive
	err = common.WriteTarball(fileStream, dirPath, files)
	if err != nil {
		return Entry{}, err
	}

	t.files[id] = struct{}{}
	return Entry{Id: id, Url: t.GetUrl(id)}, nil
}

func (t *server) hasRequest(id string) bool {
	_, ok := t.requests[id]
	return ok
}

func (t *server) Request(dirPath string) RequestEntry {
	id := SourceID(dirPath, []string{"request"})
	number := 1
	for t.hasRequest(fmt.Sprintf("%s/%d", id, number)) {
		number++
	}
	id = fmt.Sprintf("%s/%d", id, number)
	t.requests[id] = dirPath
	return RequestEntry{Id: id, Url: t.GetRequestUrl(id)}
}

func (t *server) handler() http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/download/", http.StripPrefix("/download/", http.FileServer(http.Dir(t.storagePath))))
	mux.HandleFunc("/upload/", func(writer http.ResponseWriter, request *http.Request) {
		dirPath := t.requests[request.RequestURI[8:]]
		if request.Method != http.MethodPost || dirPath == "" {
			writer.WriteHeader(http.StatusNotFound)
			return
		}

		err := common.UnpackTarball(dirPath, request.Body)
		defer request.Body.Close()
		if err != nil {
			fmt.Printf("Warning: '%s' error while unpacking tarball to: %s\n", dirPath, err.Error())
			writer.WriteHeader(http.StatusInternalServerError)
		} else {
			writer.WriteHeader(http.StatusNoContent)
		}
	})
	return mux
}

func (t *server) Listen() (func(), error) {
	addr := fmt.Sprintf(":%d", t.port)
	srv := http.Server{Addr: addr, Handler: t.handler()}
	listener, err := (&net.ListenConfig{}).Listen(context.Background(), "tcp", addr)
	if err != nil {
		return nil, err
	}
	stop := func() {
		_ = srv.Shutdown(context.Background())
	}
	go func() {
		_ = srv.Serve(listener)
	}()
	return stop, err
}
