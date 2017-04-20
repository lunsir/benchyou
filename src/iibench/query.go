/*
 * benchyou
 * xelabs.org
 *
 * Copyright (c) XeLabs
 * GPL License
 *
 */

package iibench

import (
	"fmt"
	"log"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"
	"xcommon"
	"xworker"
)

type Query struct {
	stop     bool
	requests uint64
	conf     *xcommon.BenchConf
	workers  []xworker.Worker
	lock     sync.WaitGroup
}

func NewQuery(conf *xcommon.BenchConf, workers []xworker.Worker) xworker.QueryHandler {
	return &Query{
		conf:    conf,
		workers: workers,
	}
}

func (q *Query) Run() {
	threads := len(q.workers)
	for i := 0; i < threads; i++ {
		q.lock.Add(1)
		go q.Query(&q.workers[i], threads, i)
	}
}

func (q *Query) Stop() {
	q.stop = true
	q.lock.Wait()
}

func (q *Query) Rows() uint64 {
	return atomic.LoadUint64(&q.requests)
}

func (q *Query) Query(worker *xworker.Worker, num int, id int) {
	session := worker.S
	for !q.stop {
		table := rand.Int31n(int32(worker.N))
		sql := fmt.Sprintf("select price,dateandtime,customerid from purchases_index%d force index (pdc) where (price>=%.2f) order by price,dateandtime,customerid limit 1",
			table,
			float32(rand.Int31n(10000))/100)

		t := time.Now()
		if err := session.Exec(sql); err != nil {
			log.Panicf("query.error[%v]", err)
		}
		elapsed := time.Since(t)

		// stats
		nsec := uint64(elapsed.Nanoseconds())
		worker.M.QCosts += nsec
		if worker.M.QMax == 0 && worker.M.QMin == 0 {
			worker.M.QMax = nsec
			worker.M.QMin = nsec
		}
		if nsec > worker.M.QMax {
			worker.M.QMax = nsec
		}
		if nsec < worker.M.QMin {
			worker.M.QMin = nsec
		}
		worker.M.QNums++
		atomic.AddUint64(&q.requests, 1)
	}
	q.lock.Done()
}
