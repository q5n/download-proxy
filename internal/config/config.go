package config


import (
	"os"

	"gopkg.in/yaml.v3"
)


type Config struct {

	Listen string `yaml:"listen"`

	Secret string `yaml:"secret"`

	MaxExpireSeconds int64 `yaml:"max_expire_seconds"`


	AllowedDomains []string `yaml:"allowed_domains"`
}



func Load(path string) (*Config,error){

	data,err:=os.ReadFile(path)

	if err!=nil{
		return nil,err
	}


	var cfg Config

	err=yaml.Unmarshal(
		data,
		&cfg,
	)


	if err!=nil{
		return nil,err
	}


	return &cfg,nil
}