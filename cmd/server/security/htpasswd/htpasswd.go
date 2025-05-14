package htpasswd

import (
	"os"
	"strings"
)

type (
	File struct {
		data map[string]string
	}
)

func Open(path string) (File, error) {
	c, err := os.ReadFile(path)
	if err != nil {
		return File{}, err
	}

	f := File{
		data: make(map[string]string),
	}
	creds := strings.Split(string(c), "\n")
	for _, cred := range creds {
		kv := strings.Split(cred, ":")
		if len(kv) != 2 {
			continue
		}
		f.data[kv[0]] = kv[1]
	}

	return f, nil
}

func (f File) Content() map[string]string {
	return f.data
}
