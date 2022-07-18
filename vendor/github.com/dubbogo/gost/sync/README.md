# Sync

## WorkerPool

WorkerPool is an interface which defined the behaviors of worker pool.

## baseWorkerPool

baseWorkerPool is a versatile goroutine pool with multiple queues. You may refer to [base_worker_pool.go](./base_worker_pool.go) for the architecture. **`baseWorkerPool`** provides some basic functions, like initialization, shutdown, etc. Developers are allowed to determine how to dispatch tasks to each queue, for example, dispatch the tasks to each queue fairly using Round Robin.

### Worker LifeCycle

**`p.dispatch(numWorkers, wg)`** creates workers and dispatches them to each queue equally. The method receives an instance of WaitGroup in order to block `newBaseWorkerPool` method until all workers are available. 

At the end, the worker will be killed if task queue they monitored is closed.

### Why multi-queue structure? 

In general, the worker pool has a queue only. **`baseWokerPool`** of gost, however, has multiple queues. Why? We found that multi-queue structure could improve performance greatly, especially in the case of using large number of goroutines.

There are some of benchmark results for your reference. The environment settings are:

- MacBook Air 2020 with M1 Chip
- Memory: 16GB
- Golang: go1.16.6 darwin/arm64
- @workers: 700
- Tasks: CPUTask, IOTask and RandomTask

**TaskPool(baseWorkerPool based)**

```
BenchmarkTaskPool_CPUTask/AddTaskBalance-8        	  138986	     14779 ns/op
BenchmarkTaskPool_IOTask/AddTaskBalance-8  	          2872380	     440.2 ns/op
BenchmarkTaskPool_RandomTask/AddTaskBalance-8  	      2365293	     530.0 ns/op
```

**[WorkerPool](https://github.com/gammazero/workerpool)**

```
BenchmarkWorkerPool_CPUTask/Submit-8         	   70400	     17939 ns/op
BenchmarkWorkerPool_IOTask/Submit-8         	   1000000	   1011 ns/op
BenchmarkWorkerPool_RandomTask/Submit-8          1858268	   645.5 ns/op
```

**WorkerPool with single queue**

```
BenchmarkConnectionPool/CPUTask-8           	 1844893	     16738 ns/op
BenchmarkConnectionPool/IOTask-8            	 1000000	     16047 ns/op
BenchmarkConnectionPool/RandomTask-8        	 1000000	      1143 ns/op
```

## ConnectionPool

ConnectionPool is a pool designed for managing connection based on baseWorkerPool. The pool will reject new tasks insertion if reaches the limitation.

When a new task is arriving, the task will be put into a queue using Round-Robin algorithm at the first time. If failed, the task will be put to a random queue within `len(p.taskQueues)/2` times. If all attempts are failed, it means the pool reaches the limitation, and the task will be rejected eventually.

