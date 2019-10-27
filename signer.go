package main

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

func main() {
	input := []int{0, 1}

	fmt.Println(time.Now())
	ExecutePipeline(
		job(func(in, out chan interface{}) {
			for _, number := range input {
				out <- number
			}
		}),
		job(SingleHash),
		job(MultiHash),
		job(CombineResults),
		job(func(in, out chan interface{}) {
			dataRaw := <-in
			testResult, ok := dataRaw.(string)
			fmt.Println("done", ok, testResult)
		}),
	)
	fmt.Println(time.Now())
}

func ExecutePipeline(jobs ...job) {
	in := make(chan interface{})
	wgJob := &sync.WaitGroup{}
	for _, work := range jobs {
		wgJob.Add(1)
		out := make(chan interface{})
		go func(wgJob *sync.WaitGroup, work job, in, out chan interface{}) {
			defer wgJob.Done()
			defer close(out)
			work(in, out)
		}(wgJob, work, in, out)
		in = out
	}
	wgJob.Wait()
}

func SingleHash(in, out chan interface{}) {
	wgSingle := &sync.WaitGroup{}
	muSingle := &sync.Mutex{}
	for number := range in {
		wgSingle.Add(1)

		go func(wgSingle *sync.WaitGroup, muSingle *sync.Mutex, number interface{}) {
			defer wgSingle.Done()

			numberStr := strconv.Itoa(number.(int))
			var hash1, hash2 string

			wg := &sync.WaitGroup{}
			wg.Add(2)

			go func(number string, wg *sync.WaitGroup) {
				defer wg.Done()
				hash1 = DataSignerCrc32(numberStr)
			}(numberStr, wg)

			go func(number string, wg *sync.WaitGroup, mu *sync.Mutex) {
				defer wg.Done()
				var md5hash string
				mu.Lock()
				md5hash = DataSignerMd5(numberStr)
				mu.Unlock()
				hash2 = DataSignerCrc32(md5hash)
			}(numberStr, wg, muSingle)

			wg.Wait()
			out <- fmt.Sprintf("%s~%s", hash1, hash2)
		}(wgSingle, muSingle, number)
	}
	wgSingle.Wait()
}

func MultiHash(in, out chan interface{}) {
	wgMulti := &sync.WaitGroup{}
	for singleHash := range in {
		wgMulti.Add(1)

		go func(wgMulti *sync.WaitGroup) {
			hashes := make([]string, 6)
			defer wgMulti.Done()
			wg := &sync.WaitGroup{}
			mu := &sync.Mutex{}
			for _, num := range []string{"0", "1", "2", "3", "4", "5"} {
				wg.Add(1)

				var hash string
				go func(hash string, num string) {
					hash = DataSignerCrc32(num + singleHash.(string))
					key, _ := strconv.Atoi(num)
					mu.Lock()
					hashes[key] = hash
					mu.Unlock()
					wg.Done()
				}(hash, num)
			}
			wg.Wait()

			out <- strings.Join(hashes, "")
		}(wgMulti)
	}
	wgMulti.Wait()
}

func CombineResults(in, out chan interface{}) {
	var hashes []string
	for multiHash := range in {
		hashes = append(hashes, multiHash.(string))
	}

	sort.Slice(hashes, func(i, j int) bool {
		return hashes[i] < hashes[j]
	})

	out <- strings.Join(hashes, "_")
}
