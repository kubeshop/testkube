package adapter

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/stretchr/testify/assert"

	"github.com/kubeshop/testkube/pkg/logs/events"
	"github.com/kubeshop/testkube/pkg/utils"
)

const hugeString = "82vbUcyQ0chpR665zbXY2mySOk7DGDFQCF1iLFjDNUYtNV8oQNaX3IYgJR30zBVhmDVjoZDJXO479tGSirHilZWEbzhjKJOdUwGb2HWOSOOjGh5r5wH0EHxRiOp8mBJv2rwdB2SoKF7JTBFgRt9M8F0JKp2Zx5kqh8eOGB1DGj64NLmwIpfuevJSv0wbDLrls5kEL5hHkszXPsuufVjJBsjNrxCoafuk93L2jE3ivVrMlkmLd9XAWKdop0oo0yRMJ9Vs1T5SZTkM6KXJB5hY3c14NsoPiG9Ay4EZmXrGpzGWI3RLAU6snXL8kV9sVLCG5DuRDnW047VR8eb78fpVj8YY3o9xpZd7xYPAhsmK0SwznHfrb0etAqdjQO6LFS9Blwre3G94DG5scVFH8RfteVNgKJXa8lTp8kKjtQLKNNA9mqyWfJ7uy8yjnVKwl7rodKqdtU6wjH2hf597MXA3roIS2xVhFpsCAVDybo9TVvZpoGfE9povhApoUR6Rmae9zvXPRoDbClOrvDElFkfgkJFzuoY2rPoV3dKuiTNwhYgPm36WPRk3SeFf2NiBQnWJBvjbRMIk5DsGfxcEiXQBfDvY4hgFctjwZ3USvWGriqT1cPsJ90LMLxbp38TRD1KVJ8ZgpqdvKTTi8dBqgEtob7okhdrkOahHJ3EKPtqV4PmaHvXSaIJvDG9c8jza64wxYBwMkHGt22i3HhCcIi8KmmfVo1ruqQLqKvINJg8eD5rKGV1mX9IipQcnrqADYnAj1wls7NSxsL0VZZm2pxRaGN494o2LCicHGEcOYkVLHufXY4Gv3friOIZSrT1r3NUgDBufpXWiG2b02TrRyFhgwRSS1a2OyMjHkT9tALmlIwFGF5HdaZphN6Mo5TFGdJyp65YU1scnlSGAVXzVdhsoD0RDZPSetdK2fzJC20kncaujAujHtSKnXrJNIhObnOjgMhCkx5E4z0oIH26DlfrbxS7k5SBQb1Zo3papQOk4uTNIdMBW4cE3V7AB8r6v4en3"

func init() {
	rand.New(rand.NewSource(time.Now().UnixNano()))
}

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func RandString(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}

func TestLogs(t *testing.T) {
	t.Skip("skipping test")
	ctx := context.Background()
	consumer, _ := NewMinioAdapter("localhost:9000", "minio", "minio123", "", "", "test-1", false, false, "", "", "")
	id := "test-bla"
	for i := 0; i < 1000; i++ {
		fmt.Println("sending", i)
		consumer.Notify(ctx, id, events.Log{Time: time.Now(),
			Content: fmt.Sprintf("Test %d: %s", i, hugeString),
			Type_:   "test", Source: strconv.Itoa(i)})
		time.Sleep(100 * time.Millisecond)
	}
	err := consumer.Stop(ctx, id)
	assert.NoError(t, err)
}

func BenchmarkLogs(b *testing.B) {
	ctx := context.Background()
	randomString := RandString(5)
	bucket := "test-bench"
	consumer, _ := NewMinioAdapter("localhost:9000", "minio", "minio123", "", "", bucket, false, false, "", "", "")
	id := "test-bench" + "-" + randomString + "-" + strconv.Itoa(b.N)
	totalSize := 0
	for i := 0; i < b.N; i++ {
		consumer.Notify(ctx, id, events.Log{Time: time.Now(),
			Content: fmt.Sprintf("Test %d: %s", i, hugeString),
			Type_:   "test", Source: strconv.Itoa(i)})
		totalSize += len(hugeString)
	}
	sizeInMB := float64(totalSize) / 1024 / 1024
	err := consumer.Stop(ctx, id)
	assert.NoError(b, err)
	b.Logf("Total size for %s logs is %f MB", id, sizeInMB)
}

func BenchmarkLogs2(b *testing.B) {
	bucket := "test-bench"
	consumer, _ := NewMinioAdapter("localhost:9000", "minio", "minio123", "", "", bucket, false, false, "", "", "")
	idChan := make(chan string, 100)
	go verifyConsumer(idChan, bucket, consumer.minioClient)
	var counter atomic.Int32
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			randomString := strconv.Itoa(int(counter.Add(1)))
			id := "test-bench" + "-" + randomString
			testOneConsumer(consumer, id)
			idChan <- id
		}()
	}
	wg.Wait()
}

func testOneConsumer(consumer *MinioAdapter, id string) {
	ctx := context.Background()
	fmt.Println("#####starting", id)
	totalSize := 0
	numberOFLogs := rand.Intn(100000)
	for i := 0; i < numberOFLogs; i++ {
		consumer.Notify(ctx, id, events.Log{Time: time.Now(),
			Content: fmt.Sprintf("Test %d: %s", i, hugeString),
			Type_:   "test", Source: strconv.Itoa(i)})
		totalSize += len(hugeString)
		time.Sleep(time.Duration(rand.Intn(10)) * time.Millisecond)
	}
	sizeInMB := float64(totalSize) / 1024 / 1024
	err := consumer.Stop(ctx, id)
	if err != nil {
		fmt.Println("#####error stopping", err)
	}
	fmt.Printf("#####Total size for %s logs is %f MB\n\n\n", id, sizeInMB)
}

func verifyConsumer(idChan chan string, bucket string, minioClient *minio.Client) {
	okSlice := make([]string, 0)
	notOkSlice := make([]string, 0)
	for id := range idChan {
		reader, err := minioClient.GetObject(context.Background(), bucket, id, minio.GetObjectOptions{})
		if err != nil {
			fmt.Println("######error getting object", err)
		}
		count := 0

		r := bufio.NewReader(reader)
		isOk := true
		for {
			line, err := utils.ReadLongLine(r)
			if err != nil {
				if err == io.EOF {
					err = nil
				}
				break
			}
			var LogChunk events.Log
			err = json.Unmarshal(line, &LogChunk)
			if err != nil {
				fmt.Printf("for id %s error %v unmarshalling %s\n\n\n", id, err, string(line))
				isOk = false
				break
			}
			if LogChunk.Source == "" || LogChunk.Source != strconv.Itoa(count) {
				fmt.Printf("for id %s not equal for count %d line %s \n logChunk %+v\n\n\n", id, count, string(line), LogChunk)
				isOk = false
				break
			}
			count++
		}
		if isOk {
			okSlice = append(okSlice, id)
		} else {
			notOkSlice = append(notOkSlice, id)
		}
	}
	fmt.Println("##### number of ok", len(okSlice))
	fmt.Println("#####verified ok", okSlice)
	fmt.Println("##### number of not ok", len(notOkSlice))
	fmt.Println("#####verified not ok", notOkSlice)
}

func DoRunBenchmark() {
	numberOfConsumers := 100
	bucket := "test-bench"
	consumer, _ := NewMinioAdapter("testkube-minio-service-testkube:9000", "minio", "minio123", "", "", bucket, false, false, "", "", "")

	idChan := make(chan string, numberOfConsumers)
	DoRunBenchmark2(idChan, numberOfConsumers, consumer)
	verifyConsumer(idChan, bucket, consumer.minioClient)
}

func DoRunBenchmark2(idChan chan string, numberOfConsumers int, consumer *MinioAdapter) {
	var counter atomic.Int32
	var wg sync.WaitGroup
	for i := 0; i < numberOfConsumers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			randomString := strconv.Itoa(int(counter.Add(1)))
			id := "test-bench" + "-" + randomString
			testOneConsumer(consumer, id)
			idChan <- id
		}()
	}
	wg.Wait()
	close(idChan)
	fmt.Printf("#####Done buffInfo is %+v\n\n\n", consumer.buffInfos)
}
