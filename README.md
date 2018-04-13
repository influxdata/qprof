# qprof

A tool for profiling the performance of InfluxQL queries.

`qprof` works by executing a provided query one or more times against an InfluxDB 
server and profiling the server both before and after execution. Profiles are
bundled into an archive for easy dissemination.

Because `qprof` gathers profiles before and after all queries have executed, 
delta profiles become possible, which can help to identify the cost directly 
associated with running the queries.

## Usage

Ideally the server under test should have as little traffic as possible, so as to
ensure accurate profiles.

The tool can either run a query `n` number of times, or it can keep running the 
query fora fixed duration of time (using `-d`).

### Examples

```
// Run this query for 10m.
$ qprof -db mydb -host http://example.com:8086 -t 10m "SELECT count(*) FROM m9 WHERE time > now() - 4m"
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
⇒  qprof -cpu=false -db db -t 1m -out /tmp "SELECT sum(v0) from m0 where tag2 =~ /value2|value2/"
2018/04/13 19:24:41 Host http://localhost:8086 responded to a ping in 3.625853ms
2018/04/13 19:24:41 "block" profile captured...
2018/04/13 19:24:41 "goroutine" profile captured...
2018/04/13 19:24:41 "heap" profile captured...
2018/04/13 19:24:41 "mutex" profile captured...
2018/04/13 19:24:41 Begin query execution...
2018/04/13 19:24:43 Query "SELECT sum(v0) from m0 where tag2 =~ /value2|value2/" took 2.098742827s to execute.
2018/04/13 19:24:47 Query "SELECT sum(v0) from m0 where tag2 =~ /value2|value2/" took 3.998868535s to execute.
2018/04/13 19:24:49 Query "SELECT sum(v0) from m0 where tag2 =~ /value2|value2/" took 2.220021018s to execute.
2018/04/13 19:24:51 Query "SELECT sum(v0) from m0 where tag2 =~ /value2|value2/" took 2.314364732s to execute.
2018/04/13 19:24:56 Query "SELECT sum(v0) from m0 where tag2 =~ /value2|value2/" took 4.426774156s to execute.
2018/04/13 19:24:58 Query "SELECT sum(v0) from m0 where tag2 =~ /value2|value2/" took 2.372273025s to execute.
2018/04/13 19:25:03 Query "SELECT sum(v0) from m0 where tag2 =~ /value2|value2/" took 4.5063537s to execute.
2018/04/13 19:25:05 Query "SELECT sum(v0) from m0 where tag2 =~ /value2|value2/" took 2.418578912s to execute.
2018/04/13 19:25:07 Query "SELECT sum(v0) from m0 where tag2 =~ /value2|value2/" took 2.499156031s to execute.
2018/04/13 19:25:12 Query "SELECT sum(v0) from m0 where tag2 =~ /value2|value2/" took 4.568461898s to execute.
2018/04/13 19:25:15 Query "SELECT sum(v0) from m0 where tag2 =~ /value2|value2/" took 2.746777449s to execute.
2018/04/13 19:25:20 Query "SELECT sum(v0) from m0 where tag2 =~ /value2|value2/" took 4.944668605s to execute.
2018/04/13 19:25:22 Query "SELECT sum(v0) from m0 where tag2 =~ /value2|value2/" took 2.644228295s to execute.
2018/04/13 19:25:25 Query "SELECT sum(v0) from m0 where tag2 =~ /value2|value2/" took 3.019800333s to execute.
2018/04/13 19:25:31 Query "SELECT sum(v0) from m0 where tag2 =~ /value2|value2/" took 5.338790052s to execute.
2018/04/13 19:25:33 Query "SELECT sum(v0) from m0 where tag2 =~ /value2|value2/" took 2.365137645s to execute.
2018/04/13 19:25:38 Query "SELECT sum(v0) from m0 where tag2 =~ /value2|value2/" took 5.372300161s to execute.
2018/04/13 19:25:42 Query "SELECT sum(v0) from m0 where tag2 =~ /value2|value2/" took 3.050213143s to execute.
2018/04/13 19:25:42 Queries executed for at least 1m0s
2018/04/13 19:25:42 Taking final profiles...
2018/04/13 19:25:42 "block" profile captured...
2018/04/13 19:25:42 "goroutine" profile captured...
2018/04/13 19:25:42 "heap" profile captured...
2018/04/13 19:25:42 "mutex" profile captured...
2018/04/13 19:25:42 All profiles gathered and saved at /tmp/profiles.tar.gz. Total query executions: 18.
```


