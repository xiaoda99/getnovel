package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/missdeer/golib/fsutil"
)

func readLocalBookSource() {
	matches, err := filepath.Glob("booksource/*")
	if err != nil {
		panic(err)
	}

	for _, configFile := range matches {
		fd, err := os.OpenFile(configFile, os.O_RDONLY, 0644)
		if err != nil {
			log.Println("opening book source file ", configFile, " for reading failed ", err)
			continue
		}

		c, err := ioutil.ReadAll(fd)
		if err != nil {
			log.Println("reading book source file ", configFile, " failed ", err)
			continue
		}
		fd.Close()

		// read the content
		fmt.Println(string(c))
	}
}

func parseConfigurations(content []byte, opts *Options) bool {
	var options map[string]interface{}
	if err := json.Unmarshal(content, &options); err != nil {
		log.Println("unmarshal configurations failed", err)
		return false
	}

	oe := reflect.ValueOf(opts).Elem()
	for i := 0; i < oe.NumField(); i++ {
		fieldName := oe.Type().Field(i).Name
		key := strings.ToLower(fieldName[:1]) + fieldName[1:]
		if f, ok := options[key]; ok {
			of := oe.Field(i)
			switch of.Kind() {
			case reflect.String:
				if v := f.(string); len(v) > 0 {
					of.SetString(v)
				}
			case reflect.Float64:
				if v := f.(float64); v > 0 {
					of.SetFloat(v)
				}
			case reflect.Int, reflect.Int64:
				if v := f.(float64); v > 0 {
					of.SetInt(int64(v))
				}
			}
		}
	}
	return true
}

func readRemotePreset(opts *Options) bool {
	u := "https://cdn.jsdelivr.net/gh/missdeer/getnovel/preset/" + opts.ConfigFile
	client := &http.Client{}
	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		log.Println("Could not parse preset request:", err)
		return false
	}

	resp, err := client.Do(req)
	if err != nil {
		log.Println("Could not send request:", err)
		return false
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		log.Println("response not 200:", resp.StatusCode, resp.Status)
		return false
	}

	c, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println("reading content failed")
		return false
	}

	return parseConfigurations(c, opts)
}

func readLocalConfigFile(opts *Options) bool {
	configFile := opts.ConfigFile
	if b, e := fsutil.FileExists(configFile); e != nil || !b {
		configFile = filepath.Join("preset", opts.ConfigFile)
		if b, e = fsutil.FileExists(configFile); e != nil || !b {
			log.Println("cannot find configuration file", opts.ConfigFile, "on local file system")
			return false
		}
	}

	contentFd, err := os.OpenFile(configFile, os.O_RDONLY, 0644)
	if err != nil {
		log.Println("opening config file", configFile, "for reading failed", err)
		return false
	}

	contentC, err := ioutil.ReadAll(contentFd)
	contentFd.Close()
	if err != nil {
		log.Println("reading config file", configFile, "failed", err)
		return false
	}

	return parseConfigurations(contentC, opts)
}
