package setup

import (
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
)

func envSet(file, name, value string) error {

	file, err := filepath.Abs(file)
	if err != nil {
		dir := filepath.Dir(file)
		err := os.MkdirAll(dir, os.ModePerm)
		if err != nil && !os.IsExist(err) {
			return err
		}

	}

	if _, err := os.Stat(file); err != nil && os.IsNotExist(err) {
		err = os.WriteFile(file, []byte{}, 0644)
		if err != nil {
			return err
		}

		err = godotenv.Write(map[string]string{"YAO_ENV": "development"}, file)
		if err != nil {
			return err
		}
	}

	vars, err := godotenv.Read(file)
	if err != nil {
		return err
	}

	vars[name] = value
	return godotenv.Write(vars, file)
}

func envGet(file, name string) (string, error) {

	v := os.Getenv(name)
	if v != "" {
		return v, nil
	}

	vars, err := godotenv.Read(file)
	if err == nil {
		return "", err
	}
	return vars[name], nil
}

func envHas(file, name string) (bool, error) {

	v, err := envGet(file, name)
	if err != nil {
		return false, nil
	}

	return v != "", nil
}
