package framework

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	sessionseal "github.com/lavalleeale/SessionSeal"
)

func Session(c *gin.Context) {
	cookie, err := c.Request.Cookie("session")
	if err == nil {
		sessionData, err := VerifySession(cookie.Value)
		if err == nil {
			c.Set("session", sessionData)
		} else {
			c.Set("session", map[string]string{})
		}
	} else {
		c.Set("session", map[string]string{})
	}
	c.Set("url", c.Request.URL.Path)
	c.Set("flash", GetFlash(c))
	c.Next()
	sessionData := c.MustGet("session").(map[string]string)
	session, err := json.Marshal(sessionData)
	if err != nil {
		// We have created map so marshalling it should never fail
		panic(err)
	}
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(
		"session",
		sessionseal.Seal(os.Getenv("JWT_SECRET"), session),
		2*60*60,
		"/",
		os.Getenv("DOMAIN"),
		os.Getenv("APP_ENV") == "PRODUCTION",
		false,
	)
}
func UpdateSession(c *gin.Context) {
	sessionData := c.MustGet("session").(map[string]string)
	session, err := json.Marshal(sessionData)
	if err != nil {
		// We have created map so marshalling it should never fail
		panic(err)
	}
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(
		"session",
		sessionseal.Seal(os.Getenv("JWT_SECRET"), session),
		2*60*60,
		"/",
		os.Getenv("DOMAIN"),
		os.Getenv("APP_ENV") == "PRODUCTION",
		false,
	)
}

func SetSession(c *gin.Context, key string, value string) {
	sessionData := c.MustGet("session").(map[string]string)
	sessionData[key] = value
	c.Set("session", sessionData)
}

func DeleteSession(c *gin.Context, key string) {
	sessionData := c.MustGet("session").(map[string]string)
	delete(sessionData, key)
	c.Set("session", sessionData)
}

func VerifySession(sessionString string) (map[string]string, error) {
	marshaledData, err := sessionseal.Unseal(os.Getenv("JWT_SECRET"), sessionString)
	if err != nil {
		return nil, err
	}
	var data map[string]string
	json.Unmarshal(marshaledData, &data)
	return data, nil
}
func Flash(c *gin.Context, message FlashMessage) {
	SetSession(c, "flash", fmt.Sprintf("%s|%s", message.Type, message.Message))
}

func GetFlash(c *gin.Context) *FlashMessage {
	session := c.MustGet("session").(map[string]string)
	flash, ok := session["flash"]
	if ok {
		DeleteSession(c, "flash")
		parts := strings.Split(flash, "|")
		return &FlashMessage{
			Type:    FlashType(parts[0]),
			Message: parts[1],
		}
	}
	return nil
}

type FlashType string

const (
	Error   FlashType = "error"
	Success FlashType = "success"
)

type FlashMessage struct {
	Message string
	Type    FlashType
}
