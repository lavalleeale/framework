package framework

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"reflect"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/google/uuid"
)

type Queue[T any] interface {
	Dequeue(jobIdentifier string, payload T) error
	Error(jobIdentifier string, payload T, err error)
}

type QueueHandler struct {
	queues    map[string]reflect.Value
	wg        sync.WaitGroup
	framework *Framework
}

func NewHandler(framework *Framework) *QueueHandler {
	return &QueueHandler{
		queues:    map[string]reflect.Value{},
		wg:        sync.WaitGroup{},
		framework: framework,
	}
}

func RegisterQueue[T any](q *QueueHandler, queue Queue[T]) {
	queueValue := reflect.ValueOf(queue)
	jobType := reflect.ValueOf(queue).MethodByName("Dequeue").Type().In(1)
	q.queues[jobType.Name()] = queueValue
}

func (q *QueueHandler) Run(numWorkers int) {
	q.wg.Add(numWorkers)
	for i := 0; i < numWorkers; i++ {
		go q.queueWorker()
	}
	q.wg.Wait()
}

func (q *QueueHandler) AddJob(job interface{}) error {
	return q.AddJobWithDelay(job, time.Time{})
}

func (q *QueueHandler) AddJobWithDelay(job interface{}, after time.Time) error {
	jobName := reflect.TypeOf(job).Name()
	_, ok := q.queues[jobName]
	if !ok {
		return QueueNotFoundError{}
	}
	jobJson, err := json.Marshal(job)
	if err != nil {
		return err
	}
	uuid, err := uuid.NewRandom()
	if err != nil {
		return err
	}
	var key string
	if after.IsZero() {
		key = fmt.Sprintf("%s_%s", jobName, uuid.String())
	} else {
		key = fmt.Sprintf("%s_%s_%s", jobName,
			uuid.String(), after.Format(time.RFC3339))
	}
	// add job to redis
	err = q.framework.Rdb.RPush(context.Background(), "jobs",
		key).Err()
	if err != nil {
		return err
	}
	err = q.framework.Rdb.Set(context.Background(), key, jobJson, 0).Err()
	if err != nil {
		return err
	}
	return nil
}

func (q *QueueHandler) queueWorker() {
	defer q.wg.Done()
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	for {
		if len(c) != 0 {
			return
		}
		// get next job in list from redis
		job, err := q.framework.Rdb.LPop(context.Background(), "jobs").Result()
		if err != nil {
			time.Sleep(1 * time.Second)
			continue
		}
		// split job into queue and uuid
		parts := strings.Split(job, "_")
		if len(parts) != 2 && len(parts) != 3 {
			continue
		}
		if len(parts) == 3 {
			// parse time
			t, err := time.Parse(time.RFC3339, parts[2])
			if err != nil {
				continue
			}
			// check if time is in the future
			if time.Now().Before(t) {
				len, err := q.framework.Rdb.RPush(context.Background(), "jobs", job).Result()
				if err == nil && len == 1 {
					time.Sleep(1 * time.Second)
				}
				continue
			}
		}
		// get queue
		queueType, ok := q.queues[parts[0]]
		if !ok {
			continue
		}
		// get job data from redis
		queueData, err := q.framework.Rdb.Get(context.Background(), job).Result()
		if err != nil {
			continue
		}

		dequeueMethod := queueType.MethodByName("Dequeue")
		errorMethod := queueType.MethodByName("Error")
		if !dequeueMethod.IsValid() || !errorMethod.IsValid() {
			continue
		}

		// Unmarshal json into payload (pointer to jobType)
		payload := reflect.New(dequeueMethod.Type().In(1))
		err = json.Unmarshal([]byte(queueData), payload.Interface())
		if err != nil {
			log.Println(err)
			continue
		}

		// call queue
		jobIdentifier := parts[1]
		result := dequeueMethod.Call([]reflect.Value{
			reflect.ValueOf(jobIdentifier), payload.Elem(),
		})
		// If there is an error, call the error method
		if !result[0].IsNil() {
			errorMethod.Call([]reflect.Value{
				reflect.ValueOf(jobIdentifier), payload.Elem(), result[0],
			})
		}
	}
}
