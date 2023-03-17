package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	syslog "github.com/RackSec/srslog"
	"github.com/gin-gonic/contrib/static"
	"github.com/gin-gonic/gin"
	"github.com/hashicorp/vault/api"
	auth "github.com/hashicorp/vault/api/auth/approle"
	"github.com/joho/godotenv"
)

var httpClient = &http.Client{
	Timeout: 10 * time.Second,
}
var wrapTTL = "10m"
var client *api.Client

var (
	lowerCharSet   = "abcdedfghijklmnopqrst"
	upperCharSet   = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	specialCharSet = "!@#$%&*"
	numberSet      = "0123456789"
	allCharSet     = lowerCharSet + upperCharSet + specialCharSet + numberSet
)

type VaultAppRole struct {
	RoleID   string
	SecretID *auth.SecretID
}

var vaultAppRole VaultAppRole
var vaultAddr string
var w *syslog.Writer

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	w, err = syslog.Dial("tcp", "irc-docker.brnv.rw:1523", syslog.LOG_INFO, "otl")
	if err != nil {
		log.Fatal("failed to dial syslog")
	}
	w.Info("Starting service OTL")
	rand.Seed(time.Now().Unix())
	gin.SetMode(gin.ReleaseMode)
	vaultAddr = os.Getenv("VAULT_ADDR")

	roleID := os.Getenv("ROLE_ID")
	secretID := os.Getenv("SECRET_ID")
	vaultAppRole.RoleID = roleID
	vaultAppRole.SecretID = &auth.SecretID{FromString: secretID}

	// Set the router as the default one shipped with Gin
	router := gin.Default()

	// Serve frontend static files
	router.StaticFile("/favicon.ico", "./resources/favicon.ico")
	router.Use(static.Serve("/", static.LocalFile("./views", true)))

	// Setup route group for the API
	api := router.Group("/api")
	{
		api.GET("/", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"message": "pong",
			})
		})
	}
	// Our API will consit of just two routes
	// /jokes - which will retrieve a list of jokes a user can see
	// /jokes/like/:jokeID - which will capture likes sent to a particular joke
	api.GET("/token", GetLink)
	api.GET("/password", GetPassword)
	api.POST("/token", AddLink)

	// Start and run the server
	router.Run(":3000")
}

func clientRefeshToken() error {
	client, _ = api.NewClient(&api.Config{Address: vaultAddr, HttpClient: httpClient})
	appRoleAuth, err := auth.NewAppRoleAuth(vaultAppRole.RoleID, vaultAppRole.SecretID)
	if err != nil {
		return errors.New(fmt.Sprintf("unable to initialize AppRole auth method: %w", err))
	}
	authInfo, err := client.Auth().Login(context.Background(), appRoleAuth)
	if err != nil {
		return errors.New(fmt.Sprintf("unable to login to Vault AppRole auth method: %w", err))
	}
	if authInfo == nil {
		return errors.New(fmt.Sprintf("no auth info was returned after login to Vault"))
	}
	return nil
}

func GetPassword(c *gin.Context) {
	minSpecialChar := 1
	minNum := 1
	minUpperCase := 1
	pLength := c.Query("len") // shortcut for c.Request.URL.Query().Get("lastname")
	if pLength == "" {
		pLength = "8"
	}
	passwordLength, err := strconv.Atoi(pLength)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusNotAcceptable, gin.H{"message": "Неверная длина пароля"})
		return
	}
	password := generatePassword(passwordLength, minSpecialChar, minNum, minUpperCase)
	c.Header("Content-Type", "application/json")
	c.JSON(http.StatusOK, password)
}

func GetLink(c *gin.Context) {
	err := clientRefeshToken()
	if err != nil {
		c.AbortWithStatusJSON(http.StatusNotAcceptable, gin.H{"message": "Невозможно создать секрет", "error": err.Error()})
		return
	}
	token := c.Query("token") // shortcut for c.Request.URL.Query().Get("lastname")
	if token == "null" {
		c.AbortWithStatusJSON(http.StatusNotAcceptable, gin.H{"message": "Невозможно обработать пустой токен"})
		return

	}
	ip := ReadUserIP(c.Request)
	client.SetWrappingLookupFunc(nil)
	unwrappedResponse, err := client.Logical().Write("/sys/wrapping/unwrap", map[string]interface{}{
		"token": token,
	})
	if err != nil {
		w.Err(fmt.Sprintf("ERROR unwrapping token %s from ip %s, MSG: %s", secureToken(token), ip, err.Error()))
		c.AbortWithStatusJSON(http.StatusNotAcceptable, gin.H{"message": "Сообщение не найдено или истёк срок его жизни"})
		return
	}
	w.Info(fmt.Sprintf("Unwrapped token %s from ip %s ", secureToken(token), ip))

	c.Header("Content-Type", "application/json")
	c.JSON(http.StatusOK, unwrappedResponse.Data)
}

type TokenData struct {
	Text string `json:"text"`
	TTL  string `json:"ttl"`
}

// LikeJoke increments the likes of a particular joke Item
func AddLink(c *gin.Context) {

	err := clientRefeshToken()
	if err != nil {
		c.AbortWithStatusJSON(http.StatusNotAcceptable, gin.H{"message": "Невозможно создать секрет", "error": err.Error()})
		return
	}

	var form TokenData

	err = c.BindJSON(&form)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusNotAcceptable, gin.H{"message": "Невозможно получить данные запроса", "error": err.Error()})
		return
	}
	if form.Text == "" {
		c.AbortWithStatusJSON(http.StatusNotAcceptable, gin.H{"message": "Невозможно создать секрет для пустого сообщения", "error": "Could not generate link for empty text"})
		return

	}
	wrapTTL = form.TTL
	client.SetWrappingLookupFunc(func(operation, path string) string {
		return wrapTTL
	})
	wrapData := map[string]interface{}{
		"data": form.Text,
	}

	//	wrapTTL = "10m"
	wrappedResponse, err := client.Logical().Write("/sys/wrapping/wrap", wrapData)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusNotAcceptable, gin.H{"message": "Ошибка создания секрета", "error": err.Error()})
		return
	}
	if wrappedResponse == nil {
		c.AbortWithStatusJSON(http.StatusNotAcceptable, gin.H{"message": "Ошибка создания секрета", "error": err.Error()})
		return
	}
	ip := ReadUserIP(c.Request)
	w.Info(fmt.Sprintf("Wrapped info to token %s from ip %s ", secureToken(wrappedResponse.WrapInfo.Token), ip))

	c.JSON(http.StatusOK, wrappedResponse.WrapInfo.Token)
}

func generatePassword(passwordLength, minSpecialChar, minNum, minUpperCase int) string {
	var password strings.Builder

	//Set special character
	for i := 0; i < minSpecialChar; i++ {
		random := rand.Intn(len(specialCharSet))
		password.WriteString(string(specialCharSet[random]))
	}

	//Set numeric
	for i := 0; i < minNum; i++ {
		random := rand.Intn(len(numberSet))
		password.WriteString(string(numberSet[random]))
	}

	//Set uppercase
	for i := 0; i < minUpperCase; i++ {
		random := rand.Intn(len(upperCharSet))
		password.WriteString(string(upperCharSet[random]))
	}

	remainingLength := passwordLength - minSpecialChar - minNum - minUpperCase
	for i := 0; i < remainingLength; i++ {
		random := rand.Intn(len(allCharSet))
		password.WriteString(string(allCharSet[random]))
	}
	inRune := []rune(password.String())
	rand.Shuffle(len(inRune), func(i, j int) {
		inRune[i], inRune[j] = inRune[j], inRune[i]
	})
	return string(inRune)
}

func ReadUserIP(r *http.Request) string {
	IPAddress := r.Header.Get("X-Real-Ip")
	if IPAddress == "" {
		IPAddress = r.Header.Get("X-Forwarded-For")
	}
	if IPAddress == "" {
		IPAddress = r.RemoteAddr
	}
	if strings.Index(IPAddress, "::1") > -1 {
		return "127.0.0.1"
	}
	i := strings.Index(IPAddress, ":")
	if i > -1 {
		return IPAddress[:i]
	} else {
		return IPAddress
	}

}

func secureToken(token string) string {
	return token[:8] + "*-*" + token[28:]
}
