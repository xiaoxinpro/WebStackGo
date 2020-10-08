package main

import (
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"io/ioutil"
	"log"
	"net/http"
	"os"
)

type JsMenu struct {
	Menu string		`json:"menu"`
	Name string		`json:"name"`
	Icon string		`json:"icon"`
	Sub []JsMenu	`json:"sub"`
	Url string		`json:"url"`
}

type JsClassItem struct {
	Url string		`json:"url"`
	Img string		`json:"img"`
	Name string		`json:"name"`
	Mark string		`json:"mark"`
}

type JsClass struct {
	Name string		`json:"name"`
	Rows []JsClassItem		`json:"rows"`
}

type JsWebStack struct {
	Menu []JsMenu	`json:"Menu"`
	Class []JsClass	`json:"Class"`
}

var (
	WebStack JsWebStack
)

func main() {
	fmt.Println("Hello WebStack Go")

	err := LoadJsonFile("./json/webstack.json")
	if err != nil {
		fmt.Println(err)
		return
	}

	r:=gin.Default()
	r.Static("/assets", "./public")
	r.Static("/json", "./json")
	r.LoadHTMLGlob("views/**/*")
	r.GET("/", GetIndex)
	r.GET("/index.html", GetIndex)
	r.GET("/about.html", GetAbout)
	r.GET("/admin", GetAdmin)

	r.Run(":2802")
}

func GetIndex(c *gin.Context)  {
	c.HTML(http.StatusOK, "index/index.html", gin.H{
		"title": "WebStackGo",
		"webstack": WebStack,
	})
}

func GetAbout(c *gin.Context)  {
	c.HTML(http.StatusOK, "index/about.html", gin.H{
		"title": "WebStackGo About!",
	})
}

func GetAdmin(c *gin.Context)  {
	c.HTML(http.StatusOK, "admin/index.html", gin.H{
		"title": "WebStackGo Admin!",
		"webstack": WebStack,
	})
}

func IsExist(f string) bool {
	_, err := os.Stat(f)
	return err == nil || os.IsExist(err)
}

func LoadFile(path string) ([]byte, error) {
	content, err := ioutil.ReadFile(path)
	if err != nil {
		log.Fatal(err)
		return nil, err
	}
	return content, err
}

func LoadJsonFile(path string) error {
	content, err := LoadFile(path)
	if err != nil {
		fmt.Println(err)
		return err
	}
	err = json.Unmarshal(content, &WebStack)
	if err != nil {
		fmt.Println(err)
	}
	return err
}