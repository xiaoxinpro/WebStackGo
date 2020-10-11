package main

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

type JsLogin struct {
	Username		string			`json:"username"`
	Password		string			`json:"password"`
	Path			string			`json:"path"`
}

type JsConfig struct {
	Title			string			`json:"title"`
	Url				string			`json:"url"`
	Port			int				`json:"port"`
	Keywords		string			`json:"keywords"`
	Description		string			`json:"description"`
	IsFbOpenGraph	bool			`json:"isFbOpenGraph"`
	IsTwitterCards	bool			`json:"isTwitterCards"`
	Recordcode		string			`json:"recordcode"`
	Footer			string			`json:"footer"`
}

type JsMenu struct {
	Menu			string			`json:"menu"`
	Name			string			`json:"name"`
	Icon			string			`json:"icon"`
	Sub				[]JsMenu		`json:"sub"`
	Url				string			`json:"url"`
}

type JsClassItem struct {
	Url				string			`json:"url"`
	Img				string			`json:"img"`
	Name			string			`json:"name"`
	Mark			string			`json:"mark"`
}

type JsClass struct {
	Name			string			`json:"name"`
	Rows			[]JsClassItem	`json:"rows"`
}

type JsWebStack struct {
	Menu			[]JsMenu		`json:"Menu"`
	Class			[]JsClass		`json:"Class"`
}

var (
	Login			JsLogin
	Config			JsConfig
	WebStack		JsWebStack
)

func main() {
	var err error
	err = LoadJsonFile("./json/login.json", &Login)
	if err != nil {
		fmt.Print("加载登陆文件login.json错误：")
		fmt.Println(err)
		return
	}

	err = LoadJsonFile("./json/config.json", &Config)
	if err != nil {
		fmt.Print("加载配置文件config.json错误：")
		fmt.Println(err)
		return
	} else {
		if Config.Port <= 0 || Config.Port > 65535 {
			Config.Port = 2802
		}
	}

	err = LoadJsonFile("./json/webstack.json", &WebStack)
	if err != nil {
		fmt.Print("加载页面文件webstack.json错误： ")
		fmt.Println(err)
		return
	}

	r:=gin.Default()
	r.Static("/assets", "./public")
	r.LoadHTMLGlob("views/**/*")
	r.GET("/", GetIndex)
	r.GET("/index.html", GetIndex)
	r.GET("/about.html", GetAbout)
	r.GET(Login.Path, GetLogin)
	r.POST(Login.Path, PostLogin)
	r.Use(AuthMiddleWare())
	{
		r.GET("/admin", AuthMiddleWare(), GetAdmin)
	}

	r.Run(fmt.Sprintf("%s:%d",Config.Url, Config.Port))
}

func GetIndex(c *gin.Context)  {
	c.HTML(http.StatusOK, "index/index.html", gin.H{
		"config": Config,
		"webstack": WebStack,
		//"body": template.HTML("<body>I 'm body<body>"),
	})
}

func GetAbout(c *gin.Context)  {
	c.HTML(http.StatusOK, "index/about.html", gin.H{
		"config": Config,
	})
}

func GetAdmin(c *gin.Context)  {
	cmd := c.DefaultQuery("cmd", "null")
	switch cmd {
	case "logout":
		c.SetCookie("webstackgo_token", "", -1, "/", "localhost", false, true)
		c.JSON(http.StatusOK, gin.H{
			"cmd": cmd,
			"message": "退出登陆成功",
			"err": 0,
		})
	default:
		c.HTML(http.StatusOK, "admin/index.html", gin.H{
			"login": Login,
			"config": Config,
			"webstack": WebStack,
		})
	}
}

func GetLogin(c *gin.Context)  {
	c.HTML(http.StatusOK, "admin/login.html", gin.H{
		"config": Config,
		"success": false,
		"message": "",
	})
}

func PostLogin(c *gin.Context)  {
	username := strings.TrimSpace(c.DefaultPostForm("username", ""))
	password := GetMD5(strings.TrimSpace(c.DefaultPostForm("password", "webstackgo")))
	if username == Login.Username && password == Login.Password {
		now := time.Now()
		token := GetToken(username, password, now.Unix())
		fmt.Println(token, now)
		c.SetCookie("webstackgo_token", token, 7200, "/", "localhost", false, true)
		c.HTML(http.StatusOK, "admin/login.html", gin.H{
			"config": Config,
			"success": true,
			"message": "登陆成功！",
		})
	} else {
		c.HTML(http.StatusUnauthorized, "admin/login.html", gin.H{
			"config": Config,
			"success": false,
			"message": "登陆失败：用户名或密码错误。",
		})
	}

}

func AuthMiddleWare() gin.HandlerFunc {
	return func(c *gin.Context) {
		if cookie, err := c.Request.Cookie("webstackgo_token"); err == nil {
			token, _ := url.QueryUnescape(cookie.Value)
			arr := strings.Split(token, "|")
			//fmt.Println(token, arr)
			if len(arr) == 2 {
				if intNow, err2 := strconv.ParseInt(arr[1], 10, 64); err2==nil && token==GetToken(Login.Username, Login.Password, intNow) {
					if time.Now().Unix() - intNow < 3600 {
						token = GetToken(Login.Username, Login.Password, time.Now().Unix())
						c.SetCookie("webstackgo_token", token, 7200, "/", "localhost", false, true)
					}
					c.Next()
					return
				}
			}
		}
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Unauthorized",
		})
		c.Abort()
		return
	}
}

func IsExist(f string) bool {
	_, err := os.Stat(f)
	return err == nil || os.IsExist(err)
}

func GetMD5(str string) string {
	h := md5.New()
	h.Write([]byte(str))
	return hex.EncodeToString(h.Sum(nil))
}

func GetToken(username, password string, now int64) string {
	return fmt.Sprintf("%s|%d",GetMD5(fmt.Sprintf("%s|%s|%d", username, password, now)), now)
}

func LoadFile(path string) ([]byte, error) {
	content, err := ioutil.ReadFile(path)
	if err != nil {
		log.Fatal(err)
		return nil, err
	}
	return content, err
}

func LoadJsonFile(path string, obj interface{}) error {
	content, err := LoadFile(path)
	if err != nil {
		fmt.Println(err)
		return err
	}
	err = json.Unmarshal(content, obj)
	if err != nil {
		fmt.Println(err)
	}
	return err
}