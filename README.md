# qprof

A tool for profiling the performance of InfluxQL queries.

`qprof` works by executing a provided query one or more times against an InfluxDB 
server and profiling the server before, during, and after execution. Profiles are
bundled into an archive for easy dissemination.

Because `qprof` gathers profiles before and after all queries have executed, 
delta profiles become possible, which can help to identify the cost directly 
associated with running the queries.

The following queries are captured:

  - CPU profile (before, 15s into qprof execution and after);
  - Heap profile (before and after);
  - goroutine profile (before, 15s into qprof execution and after);
  - blocking profile (before and after); and
  - mutex profile (before and after).

Finally, `qprof` stores all emitted data from the program execution, as well as
the flags that `qprof` was initialised with, in a file called `info.txt`.

## Usage

Ideally the server under test should have as little traffic as possible, so as to
ensure accurate profiles.

The tool can either run a query `n` number of times, or it can keep running the 
query for a fixed duration of time (using `-d`).

**Note**: for the _during_ profiles to be of use, it's important that `qprof` run for a reasonable
amount of time. It should run for at least one minute. If the query portion of the
program does not last for at least a minute, the following warning will be emitted:

```
***** NOTICE - QUERY EXECUTION 42.637249ms *****
This tool works most effectively if queries are executed for at least one minute
when capturing CPU profiles. Consider increasing `-n` or setting `-t 1m`.
```

### Options

```
⇒  qprof -help
Usage of qprof:
  -cpu
    	Include CPU profile (will take at least 30s) (default true)
  -db string
    	Database to query (required)
  -host string
    	scheme://host:port of server/cluster/load balancer. (default: http://localhost:8086) 
  -k	Skip SSL certificate validation
  -n int
    	Repeat query n times (default 1)
  -out string
    	Output directory (default pwd) (default ".")
  -pass string
    	Password if using authentication (optional)
  -t duration
    	Repeat query for this period of time (optional and overrides -n)
  -user string
    	Username if using authentication (optional)

```

### Examples

Here are some example uses:

```
// Run this query for 10m.
$ qprof -db mydb -host http://example.com:8086 -t 10m "SELECT count(*) FROM m9 WHERE time > now() - 4m"
```

```
// Connect to a TLS (SSL) enabled server with a self-signed certificate.
$ qprof -db mydb -k -host https://example.com:8086 -t 10m "SELECT count(*) FROM m9 WHERE time > now() - 4m"
```

```
// Save the output archive to /tmp directory
$ qprof -db mydb -host http://example.com:8086 -out /tmp -t 10m "SELECT count(*) FROM m9"
```

```
// Use authentication and run query 5 times only.
$ qprof -db mydb -host https://secret.com:8086 -user admin -pass qwertyui -n 5 "SELECT count(*) FROM m9 WHERE time > now() - 4m"
```

```
// Don't get a CPU profile
$ qprof -db mydb -host http://example.com:8086 -cpu=false -n 5 "SELECT count(*) FROM m9 WHERE time > now() - 4m"
```

The output will look something like this:

```
⇒  qprof -db db -t 1m -out /tmp "SELECT sum(v0) from m0"
2018/04/16 13:19:04 Host http://localhost:8086 responded to a ping in 6.317739ms
2018/04/16 13:19:04 Capturing CPU profile. This will take 30s...
2018/04/16 13:19:34 "profile" profile captured...
2018/04/16 13:19:34 "block" profile captured...
2018/04/16 13:19:34 "goroutine" profile captured...
2018/04/16 13:19:34 "heap" profile captured...
2018/04/16 13:19:34 "mutex" profile captured...
2018/04/16 13:19:34 Begin query execution...
2018/04/16 13:19:34 Waiting 15 seconds before taking concurrent profiles.
2018/04/16 13:19:39 Query "SELECT sum(v0) from m0" took 5.484053033s to execute.
2018/04/16 13:19:46 Query "SELECT sum(v0) from m0" took 6.989616778s to execute.
2018/04/16 13:19:49 Capturing CPU profile. This will take 30s...
2018/04/16 13:19:56 Query "SELECT sum(v0) from m0" took 9.835411294s to execute.
2018/04/16 13:20:03 Query "SELECT sum(v0) from m0" took 7.251047777s to execute.
2018/04/16 13:20:11 Query "SELECT sum(v0) from m0" took 7.595724158s to execute.
2018/04/16 13:20:17 Query "SELECT sum(v0) from m0" took 6.16515614s to execute.
2018/04/16 13:20:19 "profile" profile captured...
2018/04/16 13:20:20 "goroutine" profile captured...
2018/04/16 13:20:25 Query "SELECT sum(v0) from m0" took 7.584774771s to execute.
2018/04/16 13:20:31 Query "SELECT sum(v0) from m0" took 5.982338931s to execute.
2018/04/16 13:20:36 Query "SELECT sum(v0) from m0" took 5.886193173s to execute.
2018/04/16 13:20:36 Queries executed for at least 1m0s
2018/04/16 13:20:36 Taking final profiles...
2018/04/16 13:20:36 Capturing CPU profile. This will take 30s...
2018/04/16 13:21:06 "profile" profile captured...
2018/04/16 13:21:06 "block" profile captured...
2018/04/16 13:21:06 "goroutine" profile captured...
2018/04/16 13:21:07 "heap" profile captured...
2018/04/16 13:21:07 "mutex" profile captured...
2018/04/16 13:21:07 All profiles gathered and saved at /tmp/profiles.tar.gz. Total query executions: 9.
```

## Analysis

Where appropriate, profiles are captured in the `debug=1` Go `pprof` format, which
makes them both human readable, and supported by the `pprof` tool. If the profile
file ends in a `.txt`, then they're in this format.

#### How much memory was allocated during the executions of these queries?

You can answer this by running generating a delta heap profile using the `-alloc_space` flag:

```
edd@Alameda:~|⇒  go tool pprof -alloc_space -base /tmp/profiles/base-heap.pb.gz $GOPATH/bin/influxd /tmp/profiles/heap.pb.gz
File: influxd
Type: alloc_space
Entering interactive mode (type "help" for commands, "o" for options)
(pprof) top
Showing nodes accounting for 17921.69MB, 71.16% of 25184.02MB total
Dropped 118 nodes (cum <= 125.92MB)
Showing top 10 nodes out of 86
      flat  flat%   sum%        cum   cum%
 3916.29MB 15.55% 15.55% 14998.68MB 59.56%  github.com/influxdata/influxdb/tsdb/engine/tsm1.(*Engine).createVarRefSeriesIterator
 2947.41MB 11.70% 27.25%  2947.41MB 11.70%  github.com/influxdata/influxdb/query.newFloatReduceFloatIterator
 2348.64MB  9.33% 36.58%  2351.15MB  9.34%  runtime.mapassign_faststr
 1590.29MB  6.31% 42.89%  1590.29MB  6.31%  github.com/influxdata/influxdb/models.parseTags
 1509.09MB  5.99% 48.89%  1515.20MB  6.02%  github.com/influxdata/influxdb/tsdb/index/inmem.(*measurement).TagSets
 1462.17MB  5.81% 54.69%  2644.22MB 10.50%  github.com/influxdata/influxdb/tsdb/engine/tsm1.newKeyCursor
 1432.11MB  5.69% 60.38%  2956.24MB 11.74%  github.com/influxdata/influxdb/query.(*floatReduceFloatIterator).reduce
 1067.07MB  4.24% 64.62%  1336.57MB  5.31%  github.com/influxdata/influxdb/query.encodeTags
  850.54MB  3.38% 67.99%  1123.05MB  4.46%  github.com/influxdata/influxdb/tsdb/engine/tsm1.(*FileStore).locations
  798.06MB  3.17% 71.16%   798.06MB  3.17%  github.com/influxdata/influxdb/query.newSumIterator.func1
(pprof)
```

#### How many objects were allocated during the executions of these queries?

Sometimes it's useful to know if all that allocated memory was done via many tiny allocations,
or a few larger ones. Lots of small allocations make garbage collection slower due
to all the extra scanning and checking of small objects on the heap. The `-alloc_objects` will
tell you how many objects were allocated during the query executions:

```
 edd@Alameda:~|⇒  go tool pprof -alloc_objects -base /tmp/profiles/base-heap.pb.gz $GOPATH/bin/influxd /tmp/profiles/heap.pb.gz
File: influxd
Type: alloc_objects
Entering interactive mode (type "help" for commands, "o" for options)
(pprof) top
Showing nodes accounting for 182098616, 61.49% of 296136057 total
Dropped 123 nodes (cum <= 1480680)
Showing top 10 nodes out of 81
      flat  flat%   sum%        cum   cum%
  32637426 11.02% 11.02%   49654035 16.77%  github.com/influxdata/influxdb/models.Tags.Map
  26223442  8.86% 19.88%   35152994 11.87%  github.com/influxdata/influxdb/tsdb/engine/tsm1.(*FileStore).locations
  23721094  8.01% 27.89%   46621988 15.74%  github.com/influxdata/influxdb/query.(*floatReduceFloatIterator).reduce
  17503214  5.91% 33.80%   60389515 20.39%  github.com/influxdata/influxdb/tsdb/engine/tsm1.newKeyCursor
  17488255  5.91% 39.70%   26319500  8.89%  github.com/influxdata/influxdb/query.encodeTags
  16048840  5.42% 45.12%   16048840  5.42%  github.com/influxdata/influxdb/query.newFloatReduceFloatIterator
  15597912  5.27% 50.39%   15597912  5.27%  github.com/influxdata/influxdb/query.newFloatMergeIterator
  14970312  5.06% 55.44%   14970312  5.06%  github.com/influxdata/influxdb/query.newSumIterator.func1
   8978569  3.03% 58.48%    8978569  3.03%  github.com/influxdata/influxdb/tsdb/engine/tsm1.DecodeFloatBlock
   8929552  3.02% 61.49%    8929552  3.02%  github.com/influxdata/influxdb/tsdb/engine/tsm1.readEntries
(pprof)
```

#### How much time was spent blocking during the query execution?

Blocking profiles tell you about areas where they may be contention over locks or 
other areas of contention.

```
edd@Alameda:~|⇒  go tool pprof -base /tmp/profiles/base-block.txt $GOPATH/bin/ossinfluxd-v1.5.0 /tmp/profiles/block.txt
File: ossinfluxd-v1.5.0
Type: delay
Entering interactive mode (type "help" for commands, "o" for options)
(pprof) top
Showing nodes accounting for 985.13s, 100% of 985.13s total
Dropped 12 nodes (cum <= 4.93s)
Showing top 10 nodes out of 57
      flat  flat%   sum%        cum   cum%
   888.29s 90.17% 90.17%    888.29s 90.17%  runtime.selectgo
    67.22s  6.82% 96.99%     67.22s  6.82%  runtime.chanrecv2
    29.63s  3.01%   100%     29.63s  3.01%  sync.(*WaitGroup).Wait
         0     0%   100%     63.94s  6.49%  github.com/bmizerany/pat.(*PatternServeMux).ServeHTTP
         0     0%   100%     29.63s  3.01%  github.com/influxdata/influxdb/coordinator.(*LocalShardMapping).CreateIterator
         0     0%   100%     32.91s  3.34%  github.com/influxdata/influxdb/coordinator.(*StatementExecutor).ExecuteStatement
         0     0%   100%     29.63s  3.01%  github.com/influxdata/influxdb/coordinator.(*StatementExecutor).createIterators
         0     0%   100%     32.91s  3.34%  github.com/influxdata/influxdb/coordinator.(*StatementExecutor).executeSelectStatement
         0     0%   100%     88.42s  8.98%  github.com/influxdata/influxdb/monitor.(*Monitor).storeStatistics
         0     0%   100%     32.91s  3.34%  github.com/influxdata/influxdb/query.(*QueryExecutor).executeQuery
(pprof)
```

In this case there isn't much of interest.

#### What was the CPU doing _during_ the query execution?

`qprof` captures a CPU profile _during_ the queries' executions. This is useful for 
understanding what the server was doing. This profile is prefixed with `concurrent`.

```
edd@Alameda:~|⇒  go tool pprof $GOPATH/bin/ossinfluxd-v1.5.0 /tmp/profiles/concurrent-cpu.pb.gz
File: ossinfluxd-v1.5.0
Type: cpu
Time: Apr 16, 2018 at 1:31pm (BST)
Duration: 30.15s, Total samples = 1.20mins (239.10%)
Entering interactive mode (type "help" for commands, "o" for options)
(pprof) top
Showing nodes accounting for 39.09s, 54.22% of 72.09s total
Dropped 215 nodes (cum <= 0.36s)
Showing top 10 nodes out of 194
      flat  flat%   sum%        cum   cum%
     8.62s 11.96% 11.96%     18.04s 25.02%  runtime.scanobject
     7.89s 10.94% 22.90%      7.89s 10.94%  runtime.usleep
     6.45s  8.95% 31.85%      6.45s  8.95%  runtime.heapBitsForObject
     3.73s  5.17% 37.02%      3.73s  5.17%  runtime.greyobject
     3.33s  4.62% 41.64%      3.33s  4.62%  runtime.mach_semaphore_signal
     2.96s  4.11% 45.75%     22.52s 31.24%  runtime.mallocgc
     1.88s  2.61% 48.36%      1.88s  2.61%  runtime.(*mspan).refillAllocCache
     1.47s  2.04% 50.40%      1.47s  2.04%  runtime.duffcopy
     1.40s  1.94% 52.34%      1.40s  1.94%  runtime.heapBitsSetType
     1.36s  1.89% 54.22%      4.04s  5.60%  runtime.mapassign_faststr
(pprof)
```

In this example, most of the time is being spent doing GC. Ignoring the `runtime` package
provides a bit more insight.

```
(pprof) top -runtime
Active filters:
   ignore=runtime
Showing nodes accounting for 6.58s, 9.13% of 72.09s total
Dropped 62 nodes (cum <= 0.36s)
Showing top 10 nodes out of 116
      flat  flat%   sum%        cum   cum%
     1.15s  1.60%  1.60%      1.15s  1.60%  github.com/influxdata/influxdb/tsdb/engine/tsm1.(*indirectIndex).search.func1
     0.96s  1.33%  2.93%      2.11s  2.93%  github.com/influxdata/influxdb/pkg/bytesutil.SearchBytesFixed
     0.78s  1.08%  4.01%      9.60s 13.32%  github.com/influxdata/influxdb/tsdb/engine/tsm1.(*Engine).createVarRefSeriesIterator
     0.72s     1%  5.01%      0.72s     1%  github.com/influxdata/influxdb/models.parseTags.func1
     0.59s  0.82%  5.83%     10.19s 14.14%  github.com/influxdata/influxdb/tsdb/engine/tsm1.(*Engine).createTagSetGroupIterators
     0.59s  0.82%  6.64%      0.59s  0.82%  sync.(*RWMutex).RLock
     0.55s  0.76%  7.41%      0.67s  0.93%  github.com/influxdata/influxdb/query.(*floatMergeHeap).Less
     0.44s  0.61%  8.02%      0.44s  0.61%  sync.(*RWMutex).RUnlock
     0.43s   0.6%  8.61%      0.86s  1.19%  github.com/influxdata/influxdb/query.encodeTags
     0.37s  0.51%  9.13%      0.37s  0.51%  github.com/influxdata/influxdb/query.(*TagSet).Less
(pprof)
```