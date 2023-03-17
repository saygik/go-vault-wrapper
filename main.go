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

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
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
	router.StaticFile("/settings.svg", "./resources/settings.svg")
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
	client.SetWrappingLookupFunc(nil)
	unwrappedResponse, err := client.Logical().Write("/sys/wrapping/unwrap", map[string]interface{}{
		"token": token,
	})
	if err != nil {
		c.AbortWithStatusJSON(http.StatusNotAcceptable, gin.H{"message": "Сообщение не найдено или истёк срок его жизни"})
		return
	}
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