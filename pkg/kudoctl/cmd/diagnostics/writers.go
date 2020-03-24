package diagnostics

// TODO: use afero
import (
	"io"
	"os"
	"strings"

	"github.com/kudobuilder/kudo/pkg/kudoctl/env"
	"github.com/kudobuilder/kudo/pkg/version"
)

type writerFactory func(obj interface{}) (io.Writer, error)

var nameExtensions = map[infoType]string{
	ResourceInfoType: ".yaml",
	LogInfoType:      ".log.gz",
	DescribeInfoType: ".desc",
}

// TODO: configure target directory
var diagDir = "diag"

// fileWriter - provide a writer based on known metadata objects, i.e. generate a file name and return a file
func fileWriter(meta interface{}) (io.Writer, error) {
	dir, name := diagDir, ""
	switch info := meta.(type) {
	case resourceInfo:
		dir += "/" + info.Namespace + "/" + strings.ToLower(info.Kind)
		name = info.Name + nameExtensions[info.T]
	case version.Info:
		name = "version.yaml" // TODO: or txt?
	case env.Settings:
		name = "settings.yaml"
	case *multiError:
		name = "errors.txt"
	}
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		err = os.MkdirAll(dir, 0700)
		if err != nil {
			return nil, err
		}
	}
	return os.Create(dir + "/" + name)
}
