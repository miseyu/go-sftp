package memory

import (
	"io"

	gsftp "github.com/pkg/sftp"

	"github.com/miseyu/go-sftp/pkg/sftp"
)

type SftpHandler interface {
	gsftp.FileReader
	gsftp.FileWriter
	gsftp.FileCmder
	gsftp.FileLister
}

type inMemHandler struct {
	handler gsftp.Handlers
}

// InMemHandler returns a Hanlders object with the test handlers.
func NewInMemHandler() sftp.SftpHandler {
	return &inMemHandler{
		handler: gsftp.InMemHandler(),
	}
}

func (h *inMemHandler) Fileread(r *gsftp.Request) (io.ReaderAt, error) {
	return h.handler.FileGet.Fileread(r)
}

func (h *inMemHandler) Filecmd(r *gsftp.Request) error {
	return h.handler.FileCmd.Filecmd(r)
}

func (h *inMemHandler) Filelist(r *gsftp.Request) (gsftp.ListerAt, error) {
	return h.handler.FileList.Filelist(r)
}

func (h *inMemHandler) Filewrite(r *gsftp.Request) (io.WriterAt, error) {
	return h.handler.FilePut.Filewrite(r)
}
