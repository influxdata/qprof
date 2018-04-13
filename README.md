# qprof

A tool for profiling the performance of InfluxQL queries.

`qprof` works by executing a provided query one or more times against an InfluxDB 
server and profiling the server both before and after execution. Profiles are
bundled into an archive for easy dissemination.

Because `qprof` gathers profiles before and after all queries have executed, 
delta profiles become possible, which can help to identify the cost directly 
associated with running the queries.


