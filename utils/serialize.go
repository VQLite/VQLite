package utils

import (
	"encoding/gob"
	"os"
)

func init() {
	gob.Register(map[string]interface{}{})
	gob.Register([]interface{}{})
}

func Dump(v interface{}, filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()
	encoder := gob.NewEncoder(file)
	return encoder.Encode(v)
}

func Load(v interface{}, filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()
	decoder := gob.NewDecoder(file)
	return decoder.Decode(v)
}
