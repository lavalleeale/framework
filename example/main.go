package main

import (
	"crypto/sha512"
	"database/sql"
	"encoding/json"
	"io"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/lavalleeale/framework"
)

type LemmyPostDetails struct {
	Name string `json:"name"`
}

type LemmyPost struct {
	Post LemmyPostDetails `json:"post"`
}

type LemmyResponse struct {
	Posts []LemmyPost `json:"posts"`
}

type UserQueue struct{}

type UserJob struct {
	Name string `json:"name"`
}

func (h UserQueue) Dequeue(jobIdentifier string, payload UserJob) error {
	resp, err := http.Get("https://lemmy.world/api/v3/user?username=" + payload.Name)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	resBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	var lemmyResponse LemmyResponse
	err = json.Unmarshal(resBody, &lemmyResponse)
	if err != nil {
		return err
	}
	if len(lemmyResponse.Posts) == 0 {
		return nil
	}
	log.Println(payload.Name, lemmyResponse.Posts[0].Post.Name)

	return nil
}

func (h UserQueue) Error(jobIdentifier string, payload UserJob, err error) {
	log.Println(err)
}

type HashQueue struct{}

type HashJob struct {
	Input string `json:"input"`
	Count int    `json:"count"`
}

func (h HashQueue) Dequeue(jobIdentifier string, payload HashJob) error {
	var a []byte = []byte(payload.Input)
	for i := 0; i < payload.Count; i++ {
		h := sha512.New()
		h.Write(a)
		a = h.Sum(nil)
	}
	log.Println(a)
	return nil
}

func (h HashQueue) Error(jobIdentifier string, payload HashJob, err error) {
	log.Println(err)
}

type testMigration struct{}

func (m testMigration) Up(db *sql.DB) error {
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS test (id SERIAL PRIMARY KEY, name VARCHAR(255))`)
	return err
}

func (m testMigration) Down(db *sql.DB) error {
	_, err := db.Exec(`DROP TABLE IF EXISTS test`)
	return err
}

func main() {
	f := framework.NewFramework()
	f.ConnectDb("postgres://development:development@localhost:5432/development?sslmode=disable")
	f.Migrate(testMigration{})
	framework.RegisterQueue(f.QueueHandler, HashQueue{})
	framework.RegisterQueue(f.QueueHandler, UserQueue{})
	f.ConnectRedis("localhost:6379")

	f.Web.Router.POST("/user", func(c *gin.Context) {
		var job UserJob
		c.BindJSON(&job)
		// convert job to json
		err := f.QueueHandler.AddJob(job)
		if err != nil {
			log.Println(err)
			c.JSON(http.StatusInternalServerError, gin.H{"status": "error"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	f.Web.Router.POST("/hash", func(c *gin.Context) {
		var job HashJob
		c.BindJSON(&job)
		// convert job to json
		err := f.QueueHandler.AddJob(job)
		if err != nil {
			log.Println(err)
			c.JSON(http.StatusInternalServerError, gin.H{"status": "error"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	f.Run()
}
