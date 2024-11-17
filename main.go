package framework

import (
	"log"
	"sync"

	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
)

type Framework struct {
	QueueHandler *QueueHandler
	Rdb          *redis.Client
	Web          *Web
}

func NewFramework() *Framework {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	framework := &Framework{
		Web: NewWeb(),
	}
	framework.QueueHandler = NewHandler(framework)
	return framework
}

func (f *Framework) ConnectRedis(addr string) {
	f.Rdb = redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: "", // no password set
		DB:       0,  // use default DB
	})
}

func (f *Framework) Run() {
	wg := sync.WaitGroup{}
	wg.Add(2)
	go func() {
		defer wg.Done()
		f.Web.Run()
	}()
	go func() {
		defer wg.Done()
		f.QueueHandler.Run(10)
	}()
	wg.Wait()
}
