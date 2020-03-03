// Code generated for package crd by go-bindata DO NOT EDIT. (@generated)
// sources:
// config/crds/kudo.dev_instances.yaml
// config/crds/kudo.dev_operators.yaml
// config/crds/kudo.dev_operatorversions.yaml
package crd

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func bindataRead(data []byte, name string) ([]byte, error) {
	gz, err := gzip.NewReader(bytes.NewBuffer(data))
	if err != nil {
		return nil, fmt.Errorf("Read %q: %v", name, err)
	}

	var buf bytes.Buffer
	_, err = io.Copy(&buf, gz)
	clErr := gz.Close()

	if err != nil {
		return nil, fmt.Errorf("Read %q: %v", name, err)
	}
	if clErr != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

type asset struct {
	bytes []byte
	info  os.FileInfo
}

type bindataFileInfo struct {
	name    string
	size    int64
	mode    os.FileMode
	modTime time.Time
}

// Name return file name
func (fi bindataFileInfo) Name() string {
	return fi.name
}

// Size return file size
func (fi bindataFileInfo) Size() int64 {
	return fi.size
}

// Mode return file mode
func (fi bindataFileInfo) Mode() os.FileMode {
	return fi.mode
}

// Mode return file modify time
func (fi bindataFileInfo) ModTime() time.Time {
	return fi.modTime
}

// IsDir return file whether a directory
func (fi bindataFileInfo) IsDir() bool {
	return fi.mode&os.ModeDir != 0
}

// Sys return file is sys mode
func (fi bindataFileInfo) Sys() interface{} {
	return nil
}

var _configCrdsKudoDev_instancesYaml = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\xc4\x5a\xeb\x6f\xe3\xb8\x11\xff\xee\xbf\x62\x90\xfb\x90\x0d\x2e\x96\xbb\x69\x3f\x14\x46\x51\x20\x4d\xf6\x0a\xb7\xd7\x24\xd8\x24\x5b\x1c\xf6\xb6\x00\x4d\x8e\x2d\x36\x14\xa9\xe5\xc3\x8e\x7b\x7b\xff\x7b\x31\xa4\x24\x3f\x24\xf9\xb1\xe8\x5e\xf9\x25\x31\x45\xce\xe3\x37\x0f\xce\x50\x1a\x0c\x87\xc3\x01\x2b\xe5\x07\xb4\x4e\x1a\x3d\x06\x56\x4a\x7c\xf5\xa8\xe9\x97\xcb\x5e\xfe\xe8\x32\x69\x46\x8b\xb7\x53\xf4\xec\xed\xe0\x45\x6a\x31\x86\x9b\xe0\xbc\x29\xde\xa3\x33\xc1\x72\xbc\xc5\x99\xd4\xd2\x4b\xa3\x07\x05\x7a\x26\x98\x67\xe3\x01\x00\xd3\xda\x78\x46\xd3\x8e\x7e\x02\x70\xa3\xbd\x35\x4a\xa1\x1d\xce\x51\x67\x2f\x61\x8a\xd3\x20\x95\x40\x1b\x39\xd4\xfc\x17\xbf\xcb\xae\xb2\x3f\x0c\x00\xb8\xc5\xb8\xfd\x49\x16\xe8\x3c\x2b\xca\x31\xe8\xa0\xd4\x00\x40\xb3\x02\xc7\x20\xb5\xf3\x4c\x73\x74\xd9\x4b\x10\x26\x13\xb8\x18\xb8\x12\x39\x31\x9b\x5b\x13\xca\x31\x34\xf3\x69\x4b\x25\x47\xd2\x61\x52\xed\x8e\x53\x4a\x3a\xff\xf7\xad\xe9\x1f\xa5\xf3\xf1\x51\xa9\x82\x65\x6a\x83\x5b\x9c\x75\x52\xcf\x83\x62\x76\x3d\x3f\x00\x70\xdc\x94\x38\x86\x3b\x62\x55\x32\x8e\x82\xe6\xc2\xd4\x56\x38\x55\xec\x9d\x67\x3e\xb8\x31\xfc\xf2\xeb\x00\x60\xc1\x94\x14\x51\xcb\xf4\xd0\x94\xa8\xaf\x1f\x26\x1f\x7e\xff\xc8\x73\x2c\x58\x9a\x04\x10\xe8\xb8\x95\x65\x5c\xd7\x88\x08\xd2\x81\xcf\x11\xd2\x52\x98\x19\x1b\x7f\x36\x82\xc2\xf5\xc3\x24\xab\x08\x94\xd6\x94\x68\xbd\xac\x85\xa0\xb1\x61\xf4\x66\x6e\x87\xd5\x39\xc9\x92\xd6\x80\x20\x33\x63\x62\x59\x19\x0b\x05\xb8\xc4\xdc\xcc\xc0\xe7\xd2\x81\xc5\xd2\xa2\x43\x9d\x0c\xbf\x41\x16\x68\x09\xd3\x60\xa6\xff\x46\xee\x33\x78\x44\x4b\x44\xc0\xe5\x26\x28\x41\xbe\xb1\x40\xeb\xc1\x22\x37\x73\x2d\xff\xd3\x50\x76\xe0\x4d\x64\xa9\x98\xc7\xca\x24\xf5\x90\xda\xa3\xd5\x4c\x11\x8a\x01\x2f\x81\x69\x01\x05\x5b\x81\x45\xe2\x01\x41\x6f\x50\x8b\x4b\x5c\x06\xff\x30\x96\x20\x9a\x99\x31\xe4\xde\x97\x6e\x3c\x1a\xcd\xa5\xaf\xdd\x9c\x9b\xa2\x08\x5a\xfa\xd5\x28\x3a\xab\x9c\x06\x6f\xac\x1b\x09\x5c\xa0\x1a\x39\x39\x1f\x32\xcb\x73\xe9\x91\xfb\x60\x71\xc4\x4a\x39\x8c\x82\xeb\xe8\xe5\x59\x21\xbe\x6b\x6c\x7d\xbe\x21\xa9\x5f\x91\x5b\x38\x6f\xa5\x9e\x37\xd3\xd1\x0b\x7b\x71\x27\x67\x24\xfb\xb2\x6a\x5b\x92\x7f\x0d\x2f\x4d\x11\x2a\xef\xdf\x3d\x3e\x41\xcd\x34\x9a\x60\x1b\xf3\x88\xf6\x7a\x9b\x5b\x03\x4f\x40\x49\x3d\x43\x9b\x0c\x37\xb3\xa6\x88\x14\x51\x8b\xd2\x48\xed\xe3\x0f\xae\x24\xea\x6d\xd0\x5d\x98\x16\xd2\x93\xa5\x3f\x07\x74\x9e\xec\x93\xc1\x4d\x0c\x76\x98\x22\x84\x52\x30\x8f\x22\x83\x89\x86\x1b\x56\xa0\xba\x61\x0e\xbf\x39\xec\x84\xb0\x1b\x12\xa4\x87\x81\xdf\xcc\x51\xdb\x0b\x13\x5a\xcd\x74\x9d\x4c\x3a\x2d\x54\x07\xe1\x63\x89\x7c\x2b\x34\x04\x3a\x69\xc9\x7d\x3d\xf3\x48\x4e\x5f\xaf\xcc\x36\x48\x75\x85\x63\x15\xfe\x96\x79\x63\x3b\xe2\xb2\x25\xc1\xfd\xf6\xda\x28\xae\x9c\x49\x24\xa7\xb1\x38\x43\x8b\x94\x23\xbc\x21\x1f\x4a\x8f\xf8\xee\x9e\x1d\xf2\xb5\xbf\x64\x3b\xf3\x7d\xd2\x42\x6f\x12\xe9\x14\xf8\xfa\x61\x52\x27\x8e\x94\x2f\xb0\x96\xb3\xc5\xb1\xd7\x78\xf5\x98\x49\x54\xe2\x81\xf9\xfc\x20\xd7\xf3\xc9\x2c\xb1\x89\x61\x14\xe1\x28\x25\x72\xdc\xca\x47\x31\x69\x22\x13\x69\xb2\x83\x24\x00\x79\x9b\xc5\x6a\xfd\x65\x0a\x9a\x2a\x36\xd7\x39\xcc\x33\xa9\x81\xa5\xac\x0e\x7f\x7b\xbc\xbf\x1b\xfd\xd5\x24\x59\x3b\x69\x32\xce\xd1\xb9\xe4\x2a\x05\x6a\x7f\x09\x2e\xf0\x1c\x98\xab\xbd\xe8\x91\x9e\x64\x05\xd3\x72\x86\xce\x67\x15\x07\xb4\xee\xe3\xd5\xa7\x2e\xcc\x00\x7e\x30\x16\xf0\x95\x15\xa5\xc2\x4b\x90\x09\xe5\x26\x0b\xd4\x4e\x21\x5d\x02\xa2\xa1\x07\x4b\xe9\x73\xd9\xad\x38\x83\xd2\x88\x4a\xe1\x65\x54\xd4\xb3\x17\x04\x53\x29\x1a\x10\x94\x7c\xc1\x31\x9c\x91\x97\x6d\x88\xf8\x0b\x1d\xb9\xbf\x9e\x75\xd2\x7c\xb3\xcc\xd1\x22\x9c\xd1\x92\xb3\x24\x58\x93\xe8\x69\xae\xf6\x8f\xb5\x80\x3e\x67\x1e\xbc\x95\xf3\x39\x5a\xec\x46\x33\x66\x2f\xca\x0a\x17\x60\x2c\xe9\xae\xcd\x06\x81\x48\x96\x6c\x56\x85\x89\x68\x09\xfc\xf1\xea\x53\x8f\xb4\xdb\x38\x81\xd4\x02\x5f\xe1\x0a\xa4\x4e\xa8\x94\x46\x5c\x64\xf0\x14\x3d\x62\xa5\x3d\x7b\x25\x3e\x3c\x37\x0e\x35\x18\xad\x56\xdd\xd2\x1a\xc8\xd9\x02\xc1\x99\x02\x61\x89\x4a\x0d\x53\x16\x11\xb0\x64\x2b\xd2\xbf\x36\x17\x79\x18\x83\x92\x59\xbf\x7d\x84\x76\x52\x7d\xba\xbf\xbd\x1f\x27\xa9\xc8\x85\xe6\x9a\x44\xa1\xd4\x3c\x93\x74\x50\xd2\x09\x99\xd2\x3d\xf9\x64\x84\x23\x24\xe7\xf0\x06\x78\xce\xf4\x1c\x3b\xc9\x46\x4d\x11\x66\x81\x12\x70\x76\x7e\x6a\xb4\xee\x9e\x75\xf5\xe8\x38\xf3\x76\x13\xc3\xff\xe9\xe4\x38\x4a\xad\x58\x86\x1e\x54\xeb\x6e\xc3\x9f\xf7\xaa\x45\x05\xb1\xd5\xe8\x31\x6a\x26\x0c\x77\xa4\x14\xc7\xd2\xbb\x91\x59\xa0\x5d\x48\x5c\x8e\x96\xc6\xbe\x48\x3d\x1f\x92\x23\x0e\x93\x27\xb8\x51\x2c\x6e\x47\xdf\xc5\x3f\x5f\xa5\x45\x2c\x57\x8f\x53\x25\x2e\xfd\x2d\xf4\x21\x3e\x6e\x74\xb2\x3a\x75\x31\x74\xec\xa9\x74\xfe\x58\x1f\x8e\x3b\x3b\x29\x24\x96\xb9\xe4\x79\x5d\xd9\xae\xb3\x67\x67\x8c\x14\x4c\xa4\x94\xcb\xf4\xea\x9b\xbb\x2d\x01\x19\x2c\xc9\xb3\x1a\x56\x7d\xd5\x90\x69\x41\xff\x3b\xe9\x3c\xcd\x9f\x8c\x5c\x90\x47\x04\xe9\xf3\xe4\xf6\xb7\x71\xe6\x20\x4f\x8e\xc8\xce\x2a\x8e\x46\xc9\x2c\x2b\xd0\xa3\x6d\x15\x30\x4c\x88\xd8\xb9\x32\xf5\xb0\xa7\xc8\xf9\x2a\x9e\x8a\xe9\x77\xaf\xc8\x83\x3f\x54\xc8\x9d\x3f\xc5\xc3\x90\x59\x04\xbf\x34\x94\xfe\xa9\x84\xa3\xfd\x80\x35\x01\xe0\x4c\x53\x79\xdd\x9c\x80\x63\x80\xb7\x17\x2d\x41\xa5\x16\xd2\x22\xf7\x6a\x05\x3e\xb7\x26\xcc\xf3\xaa\x20\x8f\x47\x07\x70\x63\x2d\xba\xd2\x68\x41\x87\x4a\x83\x4a\x9d\xde\x37\x6b\xda\xec\xa1\xc1\xac\xc5\xa5\x60\x25\xc0\xd5\x05\xb4\x78\x39\xf4\xb1\x33\xa9\x1c\x64\x9b\xde\x26\x1e\xdd\xf5\x1c\xfc\x33\x97\x0a\x1b\x25\xe0\xcd\xdb\x8b\x5a\x61\x07\x39\x2b\x4b\xd4\x8e\x4e\x78\xbb\x02\x2f\x0b\x04\x06\xc1\xa1\xad\xce\xad\xb6\x98\x6c\xad\xe1\x25\xb0\xb5\xb4\x6f\xae\x2e\xd6\x38\x26\x9c\x63\x7c\x3b\xea\x8c\x44\xd3\x47\x3b\xe9\x43\xba\xbe\x68\x51\x5e\xe6\xa8\x37\x9c\x0a\x84\x41\xa7\xcf\xcf\x7d\x25\x0a\x60\x36\xcf\x88\x3d\x5a\x69\x84\xe4\x30\x65\xfc\x25\x94\xb1\xec\xea\xad\x60\x28\x28\xac\x14\x75\x63\x87\xaf\xd2\x45\x2c\xab\xbd\x33\xa9\x30\x83\xeb\xc6\x5d\xd5\xaa\x2a\xc9\x4c\x44\xc5\x1a\x53\xb4\x41\x35\x96\xfc\x86\xa3\x8a\x35\x04\x9d\xae\x6b\x26\x29\x7d\x10\x1e\x36\x68\x1d\xfd\x41\x31\xed\x76\x8e\xfa\x16\xcd\x3b\xe3\x71\x0c\x5b\xc6\xac\x8c\x57\x37\x41\x11\xd0\x58\x6d\x11\xc7\x1e\x97\x6b\x63\x1a\x0b\xbc\xc9\x23\xdc\x3c\xbf\x7f\xff\xee\xee\xe9\xc7\x9f\x2a\xe7\xa7\x5e\xf2\x3e\x76\x32\x1b\x77\x1b\x1b\x77\x49\xf0\x66\x72\x73\x41\xd0\x0a\xa3\xdb\xc5\x4b\xac\xd7\x12\x9e\x95\xb4\x97\x9b\x05\xd0\x52\x2a\x45\x61\xc5\x15\x32\x8b\xa2\x55\xd9\xec\xeb\x7c\x48\xcf\xbb\x9e\x0a\x60\x4f\xc2\xa0\xbe\x99\x4a\xfb\xdd\x6d\xc3\x86\xe0\x71\x19\xa6\xbb\x65\x4d\x77\x4b\x07\x9b\xd6\xb8\x6c\xab\x6d\x35\xd3\xca\xfd\x5b\x7d\xeb\x11\x6d\x2b\x9b\xcf\x2d\xce\xa9\xef\x7f\x6c\x09\xd0\x12\xe2\x7a\x67\x31\xd9\xae\x3e\x11\xaa\x0a\xb7\x31\xb3\xab\x05\xb5\x72\xd1\x11\x33\xcd\xa5\x45\xf4\xb9\xb4\xf8\x94\xde\x95\x7b\xb9\xc0\x87\xaf\xb3\x63\x17\xd8\x9d\xfa\x36\x71\x52\xa9\xcb\x59\x49\xa1\x95\x60\x6f\xd0\x8e\x07\xaa\x51\xca\x84\x53\x9b\xe1\xbd\x27\x50\xb7\x3d\x8e\x3b\xf5\xb6\xb4\x38\x7b\x68\xa8\xc1\xe6\x3d\x5f\xec\x7f\xd3\x74\x3c\x62\xa2\x25\x7e\xd6\xf0\x94\xa3\xeb\xea\x25\xe8\x94\x4b\xed\x71\x54\xdd\xa5\x96\xc4\x32\xed\xa2\x44\x8e\xf6\x76\x8f\x2f\x70\x47\xb9\xbf\x83\x66\x9d\x28\xe0\x4b\xcf\xd6\x35\x89\xfd\x63\xd1\x41\x3c\xee\x79\x67\xad\xb1\xf1\xd7\x9f\x86\x71\xfc\x39\x4e\x3f\x60\x4a\x6b\x9b\xb4\xff\xd5\xc7\xbb\x83\xf6\xcf\xfb\xc5\x5a\xf4\x88\xfd\x7d\x92\x61\x58\xff\x1d\x7e\x7f\x3c\xed\xf6\xde\x1e\x26\x47\xc8\xbd\xe8\x95\xfb\x0b\xfc\xc0\x3c\x53\x80\x11\xb7\x86\x74\xfc\xe7\xc6\x14\xa5\x42\xdf\xe5\x1c\x44\xf7\x4b\xbb\x07\xdf\x17\xc3\x00\x8a\x39\xff\x9c\x6e\x1d\xd7\x6f\x0a\x3a\x2b\xf4\x99\xb1\x05\xf3\x63\xa0\xb5\x43\xaa\x20\x3a\x57\xe9\xa0\x14\x9b\x2a\x1c\x83\xb7\xa1\x7b\xc9\xde\xb4\x00\x50\xa0\x73\x6c\xde\x99\x50\x0e\xee\xed\xeb\x29\x0f\x6e\x2c\x73\xe6\xba\x01\x02\x90\x1e\x8b\x9e\x47\x3b\x61\xfe\x40\x54\x8e\x09\x73\x5a\xd7\x43\x70\xbf\xb9\xd2\xd8\x0b\xd1\x51\xfa\xa6\xd1\x0f\xd7\x09\x44\xfa\x13\x79\x3d\xfe\xf7\x09\xfd\x44\x01\xb1\xdc\x2b\xdf\x5e\x03\x77\xa8\xf0\xe8\xb1\x3c\xc2\xca\xc4\x77\x2f\xd1\x63\x4c\x9d\xc6\x11\x06\x4f\xe3\x28\x40\xd2\x38\x64\xfc\x93\x09\x1e\x76\x84\x34\x4e\x77\x87\x83\x24\xe1\x58\x87\x39\x51\xa9\xde\xea\xa0\x6b\x19\xb3\x96\x75\xdf\x56\x1e\x41\x68\x3f\x89\x7d\xd0\x7e\x8b\xe8\x3a\x00\x50\xcf\x5d\xc7\x8e\x2c\xcf\x93\xdb\xf4\x12\x8e\xa8\xa5\x0e\x25\x37\x4a\x38\x08\x5a\x7e\x0e\x08\x93\xdb\xea\xbd\xe2\x25\x48\xcd\x55\x10\xfd\xd6\x78\x7e\x9e\xdc\xba\x0c\xe0\x2f\xc8\x59\x70\x08\x4b\xa4\x86\xe5\xdc\xc3\xfd\xdd\x8f\x3f\x51\x1f\x9b\x56\x54\xdd\x09\x31\xd5\xc0\x94\x8c\xef\x3f\x7b\x48\x26\xe5\x22\xcd\x74\x47\x1c\xa5\x6c\xf0\x92\xda\xa3\xf6\xb1\xbe\xca\x51\x95\x0e\x0a\xf6\x82\xe0\x82\x4d\x9a\xf4\xc9\x39\xb9\x4d\x35\x59\xbc\x86\x01\x61\xe2\xed\xf1\x1c\x3d\xb5\x5c\x33\x15\xdf\xed\x9d\x0e\xf8\x1e\xe7\xd9\x02\xdc\x29\xc9\xb1\x7a\xcb\x30\x45\x40\x1d\xaf\x15\xe2\xf5\xc8\x34\x78\x02\x8d\xa7\x37\x8d\x04\x58\x5a\x3c\x4d\x80\xb6\xfb\x69\x2b\xd0\x92\xab\x38\xfc\x1c\xd2\xbd\x9d\x86\x15\x2b\x54\xbc\x9c\x37\xda\x49\x11\xfb\x7f\x27\xe7\x5a\xce\x24\x67\xda\xc3\x32\xde\x3f\x44\x76\xd2\x9f\xb7\x7b\x54\x6d\x76\xa5\x3f\xbe\x39\xdb\x99\x5a\x7f\xee\x50\x7d\x59\xd1\x4c\xc5\x20\x19\x56\xdf\x38\xac\x9f\x02\xa4\x06\x6d\xa3\x2c\x71\xde\x58\xca\xa9\x69\x66\x1d\x61\x8c\x73\x2c\x3d\x8a\xbb\xdd\x6f\x1e\xce\x52\x6d\x55\x7f\xd2\x10\x7f\x72\x6a\xcd\xd3\x57\x1a\xf0\xf1\xd3\x20\x51\x45\xf1\xa1\x16\x86\x26\xff\x1b\x00\x00\xff\xff\xcf\xcc\x70\x94\x24\x22\x00\x00")

func configCrdsKudoDev_instancesYamlBytes() ([]byte, error) {
	return bindataRead(
		_configCrdsKudoDev_instancesYaml,
		"config/crds/kudo.dev_instances.yaml",
	)
}

func configCrdsKudoDev_instancesYaml() (*asset, error) {
	bytes, err := configCrdsKudoDev_instancesYamlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "config/crds/kudo.dev_instances.yaml", size: 0, mode: os.FileMode(0), modTime: time.Unix(0, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var _configCrdsKudoDev_operatorsYaml = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\xb4\x56\x4d\x8f\x1b\x37\x0c\xbd\xcf\xaf\x20\xd2\x43\x2e\xf5\xb8\x69\x7b\x28\x7c\x0b\xb6\x39\x2c\xda\xa4\x41\x36\xd8\x4b\xd1\x03\x67\x44\xdb\xec\xce\x48\x2a\x49\x19\xdd\xfc\xfa\x42\xd2\xf8\x63\xbc\xf6\x6e\x5b\x20\x73\xd3\x93\xc8\x47\x3d\x7e\x8c\x9a\xc5\x62\xd1\x60\xe4\x7b\x12\xe5\xe0\x57\x80\x91\xe9\x6f\x23\x9f\x57\xda\x3e\xfc\xa4\x2d\x87\xe5\xee\x4d\x47\x86\x6f\x9a\x07\xf6\x6e\x05\x37\x49\x2d\x8c\x9f\x48\x43\x92\x9e\x7e\xa6\x35\x7b\x36\x0e\xbe\x19\xc9\xd0\xa1\xe1\xaa\x01\x40\xef\x83\x61\x86\x35\x2f\x01\xfa\xe0\x4d\xc2\x30\x90\x2c\x36\xe4\xdb\x87\xd4\x51\x97\x78\x70\x24\x85\x61\xcf\xbf\xfb\xae\xfd\xbe\xfd\xb1\x01\xe8\x85\x8a\xf9\x67\x1e\x49\x0d\xc7\xb8\x02\x9f\x86\xa1\x01\xf0\x38\xd2\x0a\x42\x24\x41\x0b\xa2\xed\x43\x72\xa1\x75\xb4\x6b\x34\x52\x9f\xc9\x36\x12\x52\x5c\xc1\x01\xaf\x26\x53\x1c\xf5\x0e\xbf\x4d\xd6\x05\x1a\x58\xed\x97\x19\xfc\x2b\xab\x95\xad\x38\x24\xc1\xe1\x84\xad\xa0\xca\x7e\x93\x06\x94\x23\xde\x00\x68\x1f\x22\xad\xe0\x43\xa6\x8a\xd8\x93\x6b\x00\x76\x38\xb0\x2b\xd7\xa8\xe4\x21\x92\x7f\xfb\xf1\xf6\xfe\x87\xbb\x7e\x4b\x23\x56\x10\xc0\x91\xf6\xc2\xb1\x9c\x3b\xc4\x00\xac\x60\x5b\x82\x7a\x14\xd6\x41\xca\x72\xcf\x08\x6f\x3f\xde\x4e\xe6\x51\x32\x68\xbc\xbf\x62\xfe\x4e\x72\x7a\xc0\xce\x88\x5e\xe7\x48\xea\x19\x70\x39\x8b\x54\x09\xa7\x5c\x90\x03\xad\xd4\x61\x0d\xb6\x65\x05\xa1\x28\xa4\xe4\x6b\x5e\x4f\xdc\x42\x3e\x82\x1e\x42\xf7\x27\xf5\xd6\xc2\x1d\x49\x76\x02\xba\x0d\x69\x70\x39\xf5\x3b\x12\x03\xa1\x3e\x6c\x3c\x7f\x39\x78\x56\xb0\x50\x28\x07\x34\x9a\x14\xdf\x7f\xec\x8d\xc4\xe3\x90\x35\x4c\xf4\x2d\xa0\x77\x30\xe2\x23\x08\x65\x0e\x48\xfe\xc4\x5b\x39\xa2\x2d\xbc\x0f\x42\xc0\x7e\x1d\x56\xb0\x35\x8b\xba\x5a\x2e\x37\x6c\xfb\x2a\xee\xc3\x38\x26\xcf\xf6\xb8\x2c\xb5\xc8\x5d\xca\x09\x5d\x3a\xda\xd1\xb0\x54\xde\x2c\x50\xfa\x2d\x1b\xf5\x96\x84\x96\x18\x79\x51\x02\xf7\xa5\x88\xdb\xd1\x7d\x23\x53\xc9\xeb\xeb\x93\x48\xed\x31\x67\x5d\x4d\xd8\x6f\x0e\x70\x29\xb2\xab\xba\xe7\x5a\xcb\xd9\xc5\xc9\xac\xc6\x7f\x94\x37\x43\x59\x95\x4f\xef\xee\x3e\xc3\x9e\xb4\xa4\x60\xae\x79\x51\xfb\x68\xa6\x47\xe1\xb3\x50\xec\xd7\x24\x35\x71\x6b\x09\x63\xf1\x48\xde\xc5\xc0\xde\xca\xa2\x1f\x98\xfc\x5c\x74\x4d\xdd\xc8\x96\x33\xfd\x57\x22\xb5\x9c\x9f\x16\x6e\x4a\x2f\x43\x47\x90\xa2\x43\x23\xd7\xc2\xad\x87\x1b\x1c\x69\xb8\x41\xa5\xaf\x2e\x7b\x56\x58\x17\x59\xd2\x97\x85\x3f\x1d\x41\xf3\x83\x55\xad\x03\xbc\x9f\x15\x17\x33\xb4\x6f\xc1\xbb\x48\xfd\xac\x35\x1c\x29\x4b\x2e\x5f\x43\xa3\x5c\xf4\xb3\x39\x72\xbd\x1b\xcf\x19\x66\x1b\x57\xae\x52\xea\x28\x75\x24\x9e\x8c\xf4\x42\x33\xbf\x60\xe9\xc2\x7f\xb5\x19\x91\xbd\x21\x7b\x12\x3d\xb7\x61\xa3\xf1\x09\x78\xa6\xda\xfb\x83\xf9\x84\x77\xa4\x79\x2a\x1c\x06\xda\xd1\x7f\xfb\xc4\xd3\x35\xd5\xea\x47\x23\xf2\x70\x69\xe3\x2c\x84\x77\xf9\x5c\x69\x2d\x0f\xa1\x60\x38\x54\x63\x40\xe7\x84\xb4\x4c\x9c\x5c\x87\xd8\x97\x26\xb8\xe8\xb2\xfe\x2f\xdc\xb3\xf1\x3e\x2b\xe4\xd1\xc9\xbf\x88\x39\xff\x30\xea\x34\x48\x4a\x52\xac\x20\x08\x04\xd9\xa0\xe7\x2f\x65\xd4\x16\xf0\x7f\xc4\x70\xb1\xf2\x4f\xb7\x50\x04\x1f\x67\x3b\x49\x9e\xe8\x7c\x85\xe2\x72\x5b\x19\x5a\xd2\x97\x1b\xab\x1c\x9b\xb5\x56\xe8\x34\x0f\xaf\xe7\x7b\xeb\x02\xe7\x19\x74\x7c\x48\x4c\x6f\x96\x03\x54\xa2\x5a\x4c\xaf\x87\xe3\x2e\x40\xe5\x5d\x81\x49\xaa\xf5\xa0\x16\x04\x37\x34\x21\xc7\x2b\x61\xdf\x53\x34\x72\x1f\xce\x5f\x13\xaf\x5e\xcd\x1e\x0b\x65\xd9\x07\xef\xb8\xbe\x7f\xe0\xf7\x3f\x9a\xea\x95\xdc\xfd\x3e\x98\x0c\xfe\x13\x00\x00\xff\xff\x08\x2a\xd5\x2a\x7e\x09\x00\x00")

func configCrdsKudoDev_operatorsYamlBytes() ([]byte, error) {
	return bindataRead(
		_configCrdsKudoDev_operatorsYaml,
		"config/crds/kudo.dev_operators.yaml",
	)
}

func configCrdsKudoDev_operatorsYaml() (*asset, error) {
	bytes, err := configCrdsKudoDev_operatorsYamlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "config/crds/kudo.dev_operators.yaml", size: 0, mode: os.FileMode(0), modTime: time.Unix(0, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var _configCrdsKudoDev_operatorversionsYaml = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\xec\x5b\xdd\x6f\xe3\x36\x12\x7f\xcf\x5f\x31\x48\x1f\xf6\x0a\xc4\xf6\x75\xdb\x87\x83\xdf\x16\xbb\xed\x21\xd7\x36\x09\x36\xd9\x1e\x0e\x6d\x0f\x4b\x4b\x63\x8b\x0d\x45\xaa\x24\x65\xaf\x6f\xd1\xff\xfd\x30\xfc\x90\x64\x5b\x94\x15\x67\xd3\xdb\x03\xa2\x97\xc4\x12\xc5\xf9\xe0\x7c\xfc\x66\x48\x9d\x4d\x26\x93\x33\x56\xf1\x9f\x50\x1b\xae\xe4\x1c\x58\xc5\xf1\x83\x45\x49\xbf\xcc\xf4\xfe\x6f\x66\xca\xd5\x6c\xfd\xd5\x02\x2d\xfb\xea\xec\x9e\xcb\x7c\x0e\xaf\x6b\x63\x55\xf9\x16\x8d\xaa\x75\x86\x6f\x70\xc9\x25\xb7\x5c\xc9\xb3\x12\x2d\xcb\x99\x65\xf3\x33\x00\x26\xa5\xb2\x8c\x6e\x1b\xfa\x09\x90\x29\x69\xb5\x12\x02\xf5\x64\x85\x72\x7a\x5f\x2f\x70\x51\x73\x91\xa3\x76\x14\x22\xfd\xf5\x5f\xa7\x2f\xa7\xdf\x9c\x01\x64\x1a\xdd\xeb\x77\xbc\x44\x63\x59\x59\xcd\x41\xd6\x42\x9c\x01\x48\x56\xe2\x1c\x54\x85\x9a\x59\xa5\xc3\x9b\x66\x7a\x5f\xe7\x6a\x9a\xe3\xfa\xcc\x54\x98\x11\xcd\x95\x56\x75\x35\x87\xe6\xbe\x7f\x33\xb0\xe3\x45\xb9\x0e\x93\x04\xf1\xdd\x13\xc1\x8d\xfd\xbe\xef\xe9\x0f\xdc\x58\x37\xa2\x12\xb5\x66\xe2\x90\x05\xf7\xd0\x70\xb9\xaa\x05\xd3\x07\x8f\xcf\x00\x4c\xa6\x2a\x9c\xc3\x15\xb1\x51\xb1\x0c\xf3\x33\x80\x35\x13\x3c\x77\x92\x7a\xc6\x54\x85\xf2\xd5\xcd\xe5\x4f\x5f\xdf\x66\x05\x96\xcc\xdf\x04\xc8\xd1\x64\x9a\x57\x6e\xdc\x3e\x63\xc0\x0d\xd8\x02\xc1\xbf\x01\x4b\xa5\xdd\xcf\x7d\xf6\xe0\xd5\xcd\xe5\x34\x4c\x57\x69\x7a\x6a\x79\x54\x07\x5d\x1d\x33\x68\xee\xed\x11\x7e\x41\x9c\x05\xa2\x39\x2d\x3c\x7a\xca\x81\x04\xe6\x60\x3c\x0f\x6a\x09\xb6\xe0\x06\x34\x56\x1a\x0d\x4a\x6f\x0a\x9d\x69\x81\x86\x30\x09\x6a\xf1\x1b\x66\x76\x0a\xb7\xe8\xf8\x04\x53\xa8\x5a\xe4\x64\x2d\x6b\xd4\x16\x34\x66\x6a\x25\xf9\x7f\x9a\x99\x0d\x58\xe5\x48\x0a\x66\x31\xac\x47\xbc\xb8\xb4\xa8\x25\x13\xa4\xd3\x1a\x2f\x80\xc9\x1c\x4a\xb6\x05\x8d\x44\x03\x6a\xd9\x99\xcd\x0d\x31\x53\xf8\x51\x69\x04\x2e\x97\x6a\x0e\x85\xb5\x95\x99\xcf\x66\x2b\x6e\xa3\xe1\x67\xaa\x2c\x6b\xc9\xed\x76\xe6\xcc\x97\x2f\x6a\xab\xb4\x99\xe5\xb8\x46\x31\x33\x7c\x35\x61\x3a\x2b\xb8\xc5\xcc\xd6\x1a\x67\xac\xe2\x13\xc7\xb8\x74\x76\x3f\x2d\xf3\x2f\x74\xf0\x12\xf3\xa2\xc3\xa9\xdd\x92\x15\x18\xab\xb9\x5c\x35\xb7\x9d\x41\x26\xf5\x4e\x06\x49\xcb\xcc\xc2\x6b\x9e\xff\x56\xbd\x74\x8b\xb4\xf2\xf6\xdb\xdb\x3b\x88\x44\xdd\x12\xec\xea\xdc\x69\xbb\x7d\xcd\xb4\x8a\x27\x45\x71\xb9\x44\xed\x17\x6e\xa9\x55\xe9\x66\x44\x99\x57\x8a\x4b\xeb\x7e\x64\x82\xa3\xdc\x55\xba\xa9\x17\x25\xb7\xb4\xd2\xbf\xd7\x68\x2c\xad\xcf\x14\x5e\x3b\xf7\x87\x05\x42\x5d\xe5\xcc\x62\x3e\x85\x4b\x09\xaf\x59\x89\xe2\x35\x33\xf8\xe4\x6a\x27\x0d\x9b\x09\xa9\xf4\xb8\xe2\xbb\x51\x6b\x77\xa0\xd7\x56\x73\x3b\xc6\x95\xde\x15\xda\x73\xc9\xdb\x0a\xb3\x1d\x0f\xc9\xd1\x70\x4d\x56\x6c\x99\x45\xb2\xfd\xbd\x17\xa6\x9d\x89\xfb\x9c\xd3\x3b\x68\xd5\xe3\xa0\x49\xc1\xc0\x47\x5d\x89\x19\xb1\x78\xeb\x1e\xee\xbf\xb8\x23\xc3\xeb\xbd\xc1\x8d\x00\x0c\x2c\x96\x15\x79\x5c\x1e\xed\xcf\x16\xcc\x42\xc6\x24\x2c\x70\x6f\x4a\x80\xda\x60\x4e\x6e\x1a\x88\xd3\xbf\x4c\x02\x97\xc6\x32\x99\xa1\x8f\x0d\xd8\x28\x60\x3a\x56\x96\x1c\x2b\x94\x39\xca\xec\x40\x31\x7b\x72\xbc\xe9\x0c\x04\xe6\x02\xba\x8b\x36\x42\xec\xcc\x11\x19\x51\x09\x46\xb8\xc5\xf2\x80\x50\x62\xd9\x1b\x92\x14\x6d\x96\xa8\x51\x66\x8e\xb6\xd7\x60\x9e\xa4\x91\x5e\xec\xb8\xe4\x7d\x31\x39\xc1\xcc\xab\x9b\xcb\x18\x89\xa3\x6c\x81\x19\x7b\x48\x77\x50\xd5\xfe\x5a\x72\x14\xf9\x0d\xb3\xc5\x08\xda\x2f\x2e\x97\x9e\x98\xb7\x0e\x05\x0c\x2a\x8e\x7e\xb5\x9b\x30\xef\x6c\x00\x59\x0e\x6a\xd9\x3b\x23\xc1\x06\x20\x37\xd6\x18\xde\xb8\xf0\xd1\x28\x18\x5d\x9b\x1c\x2c\xe3\x12\x98\x4f\x9e\xf0\x8f\xdb\xeb\xab\xd9\xdf\x55\x62\x4a\x27\x05\xb0\x2c\x43\x63\xbc\xfb\x95\x28\xed\x05\x98\x3a\x2b\x80\x99\xe8\x99\xb7\xf4\x64\x5a\x32\xc9\x97\x68\xec\x34\xd0\x40\x6d\x7e\x7e\xf9\x6b\xbf\xf6\x00\xbe\x53\x1a\xf0\x03\x2b\x2b\x81\x17\xc0\x83\x35\xc5\x10\x1b\xac\xc0\x25\x67\x52\x47\x33\x23\x6c\xb8\x2d\xb8\x4c\x69\x00\x2a\x95\x07\xb1\x37\x4e\x5c\xcb\xee\x11\x54\x10\xb7\x46\x10\xfc\x1e\xe7\x70\x4e\xe1\xa8\xc3\xe6\x47\x02\x37\x7f\x9c\x27\x66\xfd\xcb\xa6\x40\x8d\x70\x4e\x83\xce\x3d\x73\x4d\x26\xa5\x7b\xd1\x5e\x5a\x26\x9d\x83\x5b\xcd\x57\x2b\xd4\x0e\xa8\xf4\x5d\x2e\x41\x50\xe0\xfd\x12\x94\x26\x0d\x48\xd5\x99\xc2\x4d\x4c\xab\x57\x61\xc6\x97\x1c\xf3\x03\xa6\x7f\x7e\xf9\x6b\x92\xe3\x5d\x7d\x01\x97\x39\x7e\x80\x97\xc0\xa5\xd7\x4d\xa5\xf2\x2f\xa7\x70\xe7\xac\x63\x2b\x2d\xfb\x40\x94\xb2\x42\x19\x4c\x69\x56\x49\xb1\x25\x99\x0b\xb6\x46\x30\xaa\x44\xd8\xa0\x10\x93\xe8\xa2\x1b\xb6\x25\x2d\xc4\x85\x23\x7b\x63\x50\x31\x6d\x07\xad\x35\xe2\x97\xbb\xeb\x37\xd7\x73\xcf\x19\x19\xd4\xca\x81\x32\xca\x81\x4b\x4e\x88\x84\xa0\x88\xcf\xab\xce\x1a\xf7\xd2\x72\x7b\x99\xda\x9b\x0f\x45\xcf\x82\xc9\x15\x7a\x79\x11\x96\x35\xe5\xba\xe9\x8b\x53\xfc\x78\x1f\x5a\xb4\x57\x0f\xc8\xd8\x0f\x1c\xff\xa3\x54\x3d\x5a\x38\x57\x0d\x8c\x10\xee\xaa\x63\xe5\x83\xc2\x51\x65\xa2\x25\x5a\x74\xf2\xe5\x2a\x33\x24\x5a\x86\x95\x35\x33\xb5\x46\xbd\xe6\xb8\x99\x6d\x94\xbe\xe7\x72\x35\x21\xd3\x9c\x78\x1b\x30\x33\x57\x5e\xcc\xbe\x70\x7f\x4e\x96\xc5\x15\x06\x63\x05\x72\x83\xff\x0c\xa9\x88\x8e\x99\x9d\x24\x54\x13\x09\xaf\xc6\xad\x94\x5b\xa8\x18\x32\xcc\x41\x84\x6a\x32\xf8\x76\x0a\x6f\xe3\xd4\xa9\xf8\xb4\xe6\x2e\x05\xb3\x5a\x58\x43\x61\x67\xc9\x57\x27\x65\xc2\x88\xa8\xc7\xe7\xe2\x17\xb7\x5e\x82\x6c\xff\x5d\x72\xed\x4d\xc1\xb3\x22\x16\x48\x41\x84\x84\x04\x9c\x90\x79\xee\xd3\x0b\x93\xdb\x27\x77\x47\x32\x8a\x5a\x13\x47\xdb\x49\x28\xd9\x27\x4c\xe6\xf4\xbf\xe1\xc6\xd2\xfd\x93\xac\xa0\xe6\xa3\x42\xd0\xbb\xcb\x37\x7f\x8e\x93\xd6\xfc\xc4\x78\xb3\x1e\x6d\x02\xe7\x71\xc5\x33\x56\x91\xba\x4d\x10\xeb\xf7\x9a\x6b\x87\x41\x8c\x2b\xd5\x37\x94\x68\x77\x5a\x08\x87\x57\x50\x09\x5b\xa8\x75\x83\x30\x98\x46\x82\xb4\x6a\x43\xd5\xd5\x2f\x12\xbe\xf5\x40\x64\x0e\xff\xfe\x7a\xfa\xd5\xf4\x9b\xfe\xbc\x3a\x28\x5c\x60\xad\x67\xa1\x26\xbb\x5e\xdc\xf3\x7c\xdd\xe9\xa0\x1c\x12\xdc\xab\xa4\xba\x8f\x98\xd6\x6c\xbb\x5b\xa4\x06\xac\x3c\x88\xf1\xaf\xdd\x8c\x4d\x00\x88\xa8\xc3\x00\x4a\x55\xaf\x0a\x67\x2e\xba\x74\x5d\x07\xf2\x38\x81\x16\xb6\xaa\x3e\x60\x8f\x4b\x8a\x34\x96\xd0\x4b\xa9\x72\xbe\xdc\xb6\xa6\x47\xd5\x5a\xc8\xee\x7b\xaf\x0d\x41\xf6\x61\xc0\xfe\x18\xb8\x3e\xb8\x74\x83\x50\xfd\x51\x40\x1d\x58\x3f\x9e\x3a\x19\xa6\x7b\x5e\x7b\xe7\x7c\x02\x90\xfe\xe9\x21\xfa\x53\x00\xf4\x27\x81\xe7\x4f\x06\xce\x1f\x01\xcd\x1d\x08\xef\xe7\xf6\x24\x60\xde\x81\xe0\xbd\xb3\x3e\x14\x96\x1f\x02\xf0\xde\x69\x8f\x83\xf2\x41\x6f\x4d\x01\xf2\xcf\x1f\x8e\x0f\x8a\x95\x82\xe2\x9f\x1d\x10\x3f\x2a\x45\x12\x84\x7f\x96\x10\xfc\x48\x52\x3f\x0a\x5d\x1f\x07\x5c\x53\xc5\xec\xff\x03\x6c\x1d\xd4\x5c\x02\xb2\x7e\x66\x80\x75\x40\x84\x24\xf6\xaa\x98\x66\x25\x5a\xd4\x07\x00\x66\x4c\xcf\xf3\x26\xbe\xbd\x0b\x6c\xd7\x4c\x73\xb6\xe0\x82\xdb\x6d\x08\xcc\x7d\xbb\x6b\xbb\xd7\x02\x29\x9a\xfb\xce\xb0\xe5\xae\xbf\x4c\x80\xa1\x6d\x16\x3f\xb4\x5f\x1a\x8a\xbd\x11\xe8\xfc\x8d\x1f\xe9\x37\x55\xc2\x6b\x21\x7f\xfb\x54\xd9\x28\x89\x86\x54\x5a\xad\x79\x9e\xac\x33\x17\x1e\x37\xa6\xb9\x86\xe3\x85\x45\x97\xbb\x31\xec\x37\x3f\xda\x65\x60\x20\x94\x5c\xa1\xee\x0e\xa5\xb5\x28\xd4\x66\xa0\x81\xd7\x0a\xba\xe1\x42\xb8\x4d\x1b\x83\xf9\x69\x32\x70\x53\x09\xb6\x1d\x59\xe9\xbf\x69\x47\x87\xad\x04\xbf\x75\xb0\xd8\xc2\xbb\x4b\x73\x12\x03\x23\xbb\x41\xe7\x57\x01\xfd\x90\xfc\xdd\x1d\x8d\x00\x5d\x23\x27\x21\xcf\xc7\xdd\x8f\x64\x87\x59\xa0\x2b\xe5\xba\x40\xf3\xbd\xdf\xa6\x7e\x7d\xfd\xee\xea\xee\x3d\xcd\x22\xa1\x36\x71\x9b\xce\xfb\x8a\x20\x8b\x49\xb6\x81\x09\x8d\x05\x28\xf9\x8b\xf4\x9b\x4f\x2e\x9c\x57\x82\x67\xcc\xcc\x01\x3e\x7e\x84\xa9\xf3\x45\x33\x75\x54\xe0\x8f\x04\xba\x3c\xda\xdc\x48\x95\x7d\x07\x7a\x7b\x1b\x86\x76\xfa\x33\x01\x53\xef\x78\x4b\x9c\x11\x6c\xaa\x29\xbf\xc0\xc6\xa5\x68\xb9\x99\x10\x8d\xf3\x98\x0b\x70\x55\x31\xda\x02\x75\xc7\x37\xc9\x42\x4c\xbd\x5c\xf2\x61\xff\x5a\x28\x25\xb0\xb7\x66\x09\x68\x79\x84\x98\x77\x7e\x24\xf0\x9c\x52\x4c\xd3\x86\xaa\x04\x93\xde\x4c\x56\x68\x0d\xe0\x07\xcc\x6a\x0a\x59\x9b\x22\xd9\x73\xf6\x78\xb8\x0d\x98\x0e\x52\x9a\x68\x57\x97\xcd\x96\x58\xe8\x22\x77\x82\xd2\x7b\xbf\x73\xfa\x3e\xd5\x17\x5a\x12\x08\x26\x86\x1c\x04\x77\x5c\x39\x48\x8f\x1f\xb8\xb1\xa4\x43\x52\xdf\x86\x1b\x04\x6e\x5f\x18\x78\x9f\x63\x25\xd4\xf6\xfd\x49\x5e\xe5\x1e\x8f\x50\xdb\xb6\xda\xef\xdc\xf9\x80\x4a\xef\x37\xc2\xb9\xc2\xe6\xdc\xd3\x3a\x3f\x81\x9d\x13\x3a\x0a\xa4\xa7\x83\x74\xc1\xf2\xdc\x1d\x5b\x61\xe2\x66\x20\xa5\xec\x66\x3e\xd2\x77\x2b\x20\x03\x83\x3a\x6c\x24\xde\x14\xcc\x38\x99\x69\x1d\xd0\xef\x7f\x2e\xa8\x60\xa3\x80\x60\xfb\xc2\xe9\x70\x22\xab\xdc\x7c\x23\x94\x1e\x08\x97\xac\x22\x86\xdc\x6b\xde\x10\x5c\x45\xeb\x9e\x0e\x56\x48\x89\x8c\x9f\xa2\xb4\x23\x7e\xdc\x59\x35\x16\xab\x20\x7b\x2c\xfa\xbf\x6f\xf0\x4e\x62\xea\x78\x18\x21\x11\xe7\x8f\xe9\xc7\x5f\xe9\x70\xef\xaf\x23\x76\xed\x2f\xc7\xfd\xd0\x2c\x3b\x5a\xb8\x75\xb2\x06\x75\xd3\xab\x1d\x6d\x47\x7d\x34\xbb\xe5\x03\x93\x42\x47\x45\x51\x15\x60\xac\xa2\xb0\xc9\xda\x83\x1e\x29\xed\xc0\xb1\xa5\x4b\xb0\xde\xd9\xd3\x37\x11\xe8\x1b\x74\x5c\xfb\xbe\xdb\x40\x17\x32\x5e\x6e\xa1\x55\x96\xd5\x3d\xdb\xd9\xdd\x6b\xcc\x0a\xfa\xeb\xd8\x3a\x06\xba\x63\x56\x33\x0c\x65\xe6\xfe\x28\xd5\x51\x1a\x7c\x30\xe9\x74\x18\xea\x1f\xd7\x1b\xc9\x1e\x3a\x9d\xb1\x9a\x59\x5c\x6d\x47\x9b\xf1\xb5\xce\xd1\x37\xeb\x1a\x7f\x2e\xd4\xc6\xe3\xa1\x7a\xe1\xf4\x12\xfb\x39\xc3\x6b\x2c\x98\x9c\xf9\xa8\xd3\x62\x27\x77\xce\x2f\x07\x55\x27\x62\x4e\x57\xae\x41\x9d\x1e\xd5\x90\xac\x85\x20\x20\x35\x07\xab\xeb\x7e\x7c\x36\xac\xbe\x61\xc5\x9d\xaa\xb2\x8e\x5a\x12\x92\x8d\x57\xd6\xa9\xc9\xf0\x20\x73\xb5\x49\x82\xd2\x58\x1b\xb5\xe8\xe7\x3e\xe9\x41\xbd\x26\x89\xf6\x7a\xdd\x0e\x1f\x3f\x74\xce\xe2\xb8\xd1\xc0\xd6\x8c\x8b\x80\x85\xbd\xee\x06\x4e\x46\xc1\xc8\x12\xf5\x8e\x99\x7b\x5f\xd9\xad\x84\x5a\x30\x71\x01\x95\x12\xdb\x52\xe9\xaa\xe0\x19\x70\xca\xc9\x65\x3c\x94\x18\xd9\xa9\xea\x85\xe0\x59\x6f\x77\xb2\xe5\xd1\xf1\xfc\xc0\x54\x9e\xde\x7e\x3f\xb9\x98\x39\xf2\xe2\xfe\x49\xb5\x01\x2d\xb9\x83\x6a\x58\x2e\x30\x37\x5e\x0b\xca\x18\x1e\x25\x75\x13\x99\xd0\xca\x75\x7b\x4d\xa9\x60\x50\xfb\x0e\xfa\x5a\xf1\x1c\x36\x9a\xbb\xf3\x88\x99\x3b\x27\x0c\xb5\x9c\x95\x4c\x9b\x82\x09\xe1\xba\xda\x94\x3c\x7c\xdf\xdc\x1d\xc8\xa8\x98\x4e\x3a\x49\x86\xda\x81\x09\xd7\x9d\x35\x61\xeb\x97\xa6\x56\xa1\x2e\x23\x1e\xbf\xe7\x32\x27\x16\x11\x72\xb5\x91\x86\xe7\x18\x4f\x9f\xa6\x4a\xab\xaa\xd2\x8a\x65\x05\x70\x73\xe1\xd9\x71\xf2\x53\x29\xe2\xba\x9f\xae\xd2\x90\xca\xfa\x7e\x74\xa0\x1d\x50\x76\xd2\x9d\xc9\x9b\x7e\x33\xca\xfb\x95\xa1\x0c\xce\xa3\x98\x0b\xcc\x54\x89\xc0\xca\x05\x5f\xd5\xaa\x36\xcd\x01\xdd\x50\xd9\xa4\xf0\x0f\x29\x46\x4f\xe1\x9f\x08\x25\x5f\x15\x16\x34\xae\xb9\xe1\xd6\x3b\x49\x2b\x44\xb7\x15\x1d\xc2\xca\x50\x31\x12\xb9\x91\xc0\x8d\xa9\x13\xa5\xd4\xf1\xcc\x9d\x2b\x39\x90\xb1\x8f\x95\x62\x74\x2d\x99\x65\xe2\x71\x53\x34\x85\xd5\xb1\x69\x06\x93\x4c\xc5\x53\xd5\x0d\x8c\x81\x08\xbb\x71\x96\x57\x18\xce\x7e\xd2\xdd\x45\xc8\x11\xcc\xf7\x08\x56\x28\x29\xb4\xb9\x9a\x77\x30\xab\x32\x37\x51\x8c\x62\x11\x15\xca\xbc\xed\xba\x0e\x41\xcb\xb1\xb8\x8b\x78\x3a\x06\x7e\x46\x03\x9f\x7b\x1c\x04\x1f\x0f\x9b\x2b\x19\x31\x1f\x3c\xd9\x28\x8c\x35\x02\x45\xc0\x28\x20\x56\xa9\x01\xbe\x47\x70\xdc\x1c\x17\x7f\x84\x3d\x8e\x52\xcc\x27\x93\x78\xc3\xa4\xfd\x56\x1f\x75\xc0\x21\x3f\x1e\x5c\xa2\x13\xaa\xfd\x58\x7d\x9d\x58\xf1\x0f\xe8\x6f\x37\x7b\x46\x32\x1e\x68\xc4\xda\xaf\x73\xfc\xd7\x2a\xf8\xd7\xab\x1f\x7f\x68\x19\x02\xa1\xb2\xde\xb2\x70\xaf\xcf\x48\x29\x42\xe4\xa8\x9d\xcb\xd3\x0d\xdd\x71\xfc\x70\x34\x9f\x80\x48\xff\xd1\xe9\x1e\x65\xd5\xd5\x4a\xb3\x9c\x16\xfc\x3b\xad\xca\x41\x84\xf6\x6e\x67\xa8\x13\xcb\x23\x83\x3d\x58\x66\xda\x23\xe0\x7e\xf6\x43\x2b\x72\x1b\xd8\x9f\x06\xd0\x7d\xa2\xe3\x1e\x27\x1e\xf8\x78\x3e\xa5\xbd\x27\xef\xf3\x29\xed\xe7\x53\xda\x9e\xe3\xe7\x53\xda\xcf\xa7\xb4\x47\x08\xf7\x7c\x4a\xfb\xb3\x3f\xa5\xfd\x7c\xc2\xf9\xf9\x84\xf3\x69\x80\x3b\x71\x24\x3a\x41\xa6\xff\xdb\x4a\xcb\x6c\x6d\x46\x7f\x5d\xe9\x46\xef\x7c\x5f\xa9\x16\x06\xf5\x7a\xe4\x07\x96\x3d\x2c\xec\xdd\x6a\xbf\x47\x0f\x9f\xbe\x37\xb7\x1c\x93\x93\xf0\x11\x7a\xfb\x14\xc0\xd3\xef\x14\x54\x54\xb6\xb3\x55\x2c\xb1\x5a\x09\x09\xe4\x54\x16\xf3\xab\xfd\xaf\xd1\xcf\x7d\x96\x8d\x9f\x97\xbb\x9f\x99\x92\xbe\x68\x31\x73\xf8\xf9\xd7\x33\x08\xcd\x80\x08\xc2\xdd\xcd\xff\x06\x00\x00\xff\xff\xbb\x6b\x7b\xe8\xc5\x3f\x00\x00")

func configCrdsKudoDev_operatorversionsYamlBytes() ([]byte, error) {
	return bindataRead(
		_configCrdsKudoDev_operatorversionsYaml,
		"config/crds/kudo.dev_operatorversions.yaml",
	)
}

func configCrdsKudoDev_operatorversionsYaml() (*asset, error) {
	bytes, err := configCrdsKudoDev_operatorversionsYamlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "config/crds/kudo.dev_operatorversions.yaml", size: 0, mode: os.FileMode(0), modTime: time.Unix(0, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

// Asset loads and returns the asset for the given name.
// It returns an error if the asset could not be found or
// could not be loaded.
func Asset(name string) ([]byte, error) {
	cannonicalName := strings.Replace(name, "\\", "/", -1)
	if f, ok := _bindata[cannonicalName]; ok {
		a, err := f()
		if err != nil {
			return nil, fmt.Errorf("Asset %s can't read by error: %v", name, err)
		}
		return a.bytes, nil
	}
	return nil, fmt.Errorf("Asset %s not found", name)
}

// MustAsset is like Asset but panics when Asset would return an error.
// It simplifies safe initialization of global variables.
func MustAsset(name string) []byte {
	a, err := Asset(name)
	if err != nil {
		panic("asset: Asset(" + name + "): " + err.Error())
	}

	return a
}

// AssetInfo loads and returns the asset info for the given name.
// It returns an error if the asset could not be found or
// could not be loaded.
func AssetInfo(name string) (os.FileInfo, error) {
	cannonicalName := strings.Replace(name, "\\", "/", -1)
	if f, ok := _bindata[cannonicalName]; ok {
		a, err := f()
		if err != nil {
			return nil, fmt.Errorf("AssetInfo %s can't read by error: %v", name, err)
		}
		return a.info, nil
	}
	return nil, fmt.Errorf("AssetInfo %s not found", name)
}

// AssetNames returns the names of the assets.
func AssetNames() []string {
	names := make([]string, 0, len(_bindata))
	for name := range _bindata {
		names = append(names, name)
	}
	return names
}

// _bindata is a table, holding each asset generator, mapped to its name.
var _bindata = map[string]func() (*asset, error){
	"config/crds/kudo.dev_instances.yaml":        configCrdsKudoDev_instancesYaml,
	"config/crds/kudo.dev_operators.yaml":        configCrdsKudoDev_operatorsYaml,
	"config/crds/kudo.dev_operatorversions.yaml": configCrdsKudoDev_operatorversionsYaml,
}

// AssetDir returns the file names below a certain
// directory embedded in the file by go-bindata.
// For example if you run go-bindata on data/... and data contains the
// following hierarchy:
//     data/
//       foo.txt
//       img/
//         a.png
//         b.png
// then AssetDir("data") would return []string{"foo.txt", "img"}
// AssetDir("data/img") would return []string{"a.png", "b.png"}
// AssetDir("foo.txt") and AssetDir("notexist") would return an error
// AssetDir("") will return []string{"data"}.
func AssetDir(name string) ([]string, error) {
	node := _bintree
	if len(name) != 0 {
		cannonicalName := strings.Replace(name, "\\", "/", -1)
		pathList := strings.Split(cannonicalName, "/")
		for _, p := range pathList {
			node = node.Children[p]
			if node == nil {
				return nil, fmt.Errorf("Asset %s not found", name)
			}
		}
	}
	if node.Func != nil {
		return nil, fmt.Errorf("Asset %s not found", name)
	}
	rv := make([]string, 0, len(node.Children))
	for childName := range node.Children {
		rv = append(rv, childName)
	}
	return rv, nil
}

type bintree struct {
	Func     func() (*asset, error)
	Children map[string]*bintree
}

var _bintree = &bintree{nil, map[string]*bintree{
	"config": &bintree{nil, map[string]*bintree{
		"crds": &bintree{nil, map[string]*bintree{
			"kudo.dev_instances.yaml":        &bintree{configCrdsKudoDev_instancesYaml, map[string]*bintree{}},
			"kudo.dev_operators.yaml":        &bintree{configCrdsKudoDev_operatorsYaml, map[string]*bintree{}},
			"kudo.dev_operatorversions.yaml": &bintree{configCrdsKudoDev_operatorversionsYaml, map[string]*bintree{}},
		}},
	}},
}}

// RestoreAsset restores an asset under the given directory
func RestoreAsset(dir, name string) error {
	data, err := Asset(name)
	if err != nil {
		return err
	}
	info, err := AssetInfo(name)
	if err != nil {
		return err
	}
	err = os.MkdirAll(_filePath(dir, filepath.Dir(name)), os.FileMode(0755))
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(_filePath(dir, name), data, info.Mode())
	if err != nil {
		return err
	}
	err = os.Chtimes(_filePath(dir, name), info.ModTime(), info.ModTime())
	if err != nil {
		return err
	}
	return nil
}

// RestoreAssets restores an asset under the given directory recursively
func RestoreAssets(dir, name string) error {
	children, err := AssetDir(name)
	// File
	if err != nil {
		return RestoreAsset(dir, name)
	}
	// Dir
	for _, child := range children {
		err = RestoreAssets(dir, filepath.Join(name, child))
		if err != nil {
			return err
		}
	}
	return nil
}

func _filePath(dir, name string) string {
	cannonicalName := strings.Replace(name, "\\", "/", -1)
	return filepath.Join(append([]string{dir}, strings.Split(cannonicalName, "/")...)...)
}
