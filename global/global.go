package global

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/facebookgo/inject"
)

type ConfigStruct struct {
	StatsDServerIP string
	RootFolder     string
}

var Config ConfigStruct

func LoadConfig(config string, rootfolder string) error {
	fmt.Println("Loading Config: ", config)

	file, err := os.Open(config)
	if err != nil {
		return fmt.Errorf("Unable to open config")
	}

	decoder := json.NewDecoder(file)
	Config = ConfigStruct{}
	err = decoder.Decode(&Config)
	Config.RootFolder = rootfolder

	fmt.Println(Config)

	return nil
}

func SetupInjection(objects ...*inject.Object) error {
	var g inject.Graph

	var err error

	err = g.Provide(objects...)

	if err != nil {
		fmt.Println(err)
		return err
	}

	// Here the Populate call is creating instances of NameAPI &
	// PlanetAPI, and setting the HTTPTransport on both to the
	// http.DefaultTransport provided above:
	if err := g.Populate(); err != nil {
		fmt.Println(err)
		return err
	}

	return nil
}
