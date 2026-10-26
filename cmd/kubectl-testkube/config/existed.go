package config

import "os"

// Every command creates the file on load, so check before that.
var existedAtStart = func() bool {
	storage := Storage{Dir: defaultDirectory}
	path, err := storage.getPath()
	if err != nil {
		return true
	}
	_, err = os.Stat(path)
	return !os.IsNotExist(err)
}()

// ExistedAtStart tells a first-ever CLI run from a configured one.
func ExistedAtStart() bool {
	return existedAtStart
}
