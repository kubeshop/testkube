package config

// defaultDirectory is the default directory for the config file
var defaultDirectory = ""

func Load() (Data, error) {
	storage, err := GetStorage(defaultDirectory)
	if err != nil {
		return Data{}, err
	}
	return storage.Load()
}

func Save(data Data) error {
	storage, err := GetStorage(defaultDirectory)
	if err != nil {
		return err
	}
	return storage.Save(data)
}
