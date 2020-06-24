package diagnostics

import (
	"fmt"

	"github.com/spf13/afero"
)

const (
	testLog        = "2020/05/25 08:00:55 Ein Fichtenbaum steht einsam im Norden auf kahler Höh"
	testLogGZipped = "\x1f\x8b\b\x00\x00\x00\x00\x00\x00\xff2202\xd070\xd572U0\xb0\xb020" +
		"\xb025Up\xcd\xccSp\xcbL\xce(I\xcdKJ,\xcdU(.I\xcd(QH\xcd\xcc+N\xccU\xc8\xccU\xf0" +
		"\xcb/JI\xcdSH,MS\xc8N\xcc\xc8I-R\xf08\xbc-\x03\x10\x00\x00\xff\xff'\b\x1b\xe7J\x00\x00\x00"
)

var errFakeTestError = fmt.Errorf("fake test error")

// failingFs is a wrapper of afero.Fs to simulate a specific file creation failure for printer
type failingFs struct {
	afero.Fs
	failOn string
}

func (s *failingFs) Create(name string) (afero.File, error) {
	if name == s.failOn {
		return nil, errFakeTestError
	}
	return s.Fs.Create(name)
}
