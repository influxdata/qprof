package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"time"

	client "github.com/influxdata/influxdb/client/v2"
)

type profile struct {
	Name  string
	Fname string
	Debug int
}

// cpu must always be the first profile.
var profiles = []profile{
	{Name: "profile", Fname: "cpu.pb.gz"},
	{Name: "block", Fname: "block.txt", Debug: 1},
	{Name: "goroutine", Fname: "goroutine.txt", Debug: 1},
	{Name: "heap", Fname: "heap.pb.gz", Debug: 1},
	{Name: "mutex", Fname: "mutex.txt", Debug: 1},
}

const example = `
Example usage: $ qprof -db mydb -d 5m "SELECT * FROM cpu WHERE tag1 = 'foo'"
	
`

// Server connection flags
var (
	host       string
	user, pass string
)

// Database and query options
var (
	query string
	db    string
	n     int
	d     time.Duration
)

// Program options
var (
	out string
	cpu bool
)

var clt client.Client
var err error

// Used to store relevant data around the execution of this program.
var infoBuf bytes.Buffer
var archivePath string
var totalExecutions int
var totalTime time.Duration

// Duplicates writes to os.Stderr and file in archive.
var stderr io.Writer
var logger *log.Logger

func main() {
	flag.StringVar(&host, "host", "http://localhost:8086", "scheme://host:port of server/cluster/load balancer. (default: http://localhost:8086)")
	flag.StringVar(&user, "user", "", "Username if using authentication (optional)")
	flag.StringVar(&pass, "pass", "", "Password if using authentication (optional)")

	flag.StringVar(&db, "db", "", "Database to query (required)")
	flag.IntVar(&n, "n", 1, "Repeat query n times (default 1 if -d not specified)")
	flag.DurationVar(&d, "t", 0, "Repeat query for this period of time (optional and overrides -n)")

	flag.StringVar(&out, "out", ".", "Output directory (default pwd)")
	flag.BoolVar(&cpu, "cpu", true, "Include CPU profile (will take at least 30s)")
	flag.Parse()

	if len(flag.Args()) < 1 {
		fmt.Fprintln(os.Stderr, "Please provide query as positional argument:\n", example)
		os.Exit(1)
	} else if len(flag.Args()) > 1 {
		fmt.Fprintln(os.Stderr, "Query partially parsed. Is it quoted properly?\n", example)
		os.Exit(1)
	}
	query = flag.Arg(0)

	// Tee output to user and file inside profile.
	stderr = io.MultiWriter(&infoBuf, os.Stderr)
	logger = log.New(stderr, "", log.LstdFlags)

	// Store options set.
	infoBuf.WriteString("Flags:\n")
	flag.VisitAll(func(f *flag.Flag) {
		if _, err := infoBuf.WriteString(fmt.Sprintf("-%s %v\n", f.Name, f.Value.String())); err != nil {
			fmt.Fprintln(stderr, err)
			os.Exit(1)
		}
	})
	infoBuf.WriteString("\n")

	if !cpu {
		profiles = profiles[1:]
	}

	// Check out directory is available.
	if err := os.MkdirAll(out, 0600); err != nil {
		fmt.Fprintln(stderr, err)
		os.Exit(1)
	}
	archivePath = filepath.Join(out, "profiles.tar.gz")

	// Check we can connect to the server.
	if clt, err = NewClient(); err != nil {
		fmt.Fprintln(stderr, err)
		os.Exit(1)
	}
	defer clt.Close()

	if err := run(); err != nil {
		fmt.Fprintln(stderr, err)
		os.Exit(1)
	}
}

func run() error {
	var allBuf bytes.Buffer // Buffer for entire archive.
	var buf bytes.Buffer    // Temporary buffer for each profile result.

	gz := gzip.NewWriter(&allBuf)
	defer gz.Close()
	tw := tar.NewWriter(gz)
	defer tw.Close()

	captureProfile := func(p profile) error {
		if p.Name == "profile" {
			logger.Print("Capturing CPU profile. This will take 30s...")
		}

		if err := takeProfile(&buf, p.Name, p.Debug); err != nil {
			return err
		}

		err := tw.WriteHeader(&tar.Header{
			Name:    p.Fname,
			Mode:    0600,
			ModTime: time.Now().UTC(),
			Size:    int64(buf.Len()),
		})
		if err != nil {
			return err
		}

		// Write the profile file's data to the tar writer.
		if _, err = io.Copy(tw, &buf); err != nil {
			return err
		}

		// Reset the buffer for the next profile.
		buf.Reset()
		logger.Printf("%q profile captured...\n", p.Name)
		return nil
	}

	// Take the base profiles.
	for _, p := range profiles {
		p.Fname = fmt.Sprintf("base-%s", p.Fname)
		if err := captureProfile(p); err != nil {
			return err
		}
	}

	// Run the queries
	logger.Print("Begin query execution...")
	now := time.Now()
	if d == 0 {
		for i := 0; i < n; i++ {
			if err := runQuery(); err != nil {
				return err
			}
		}
	} else {
		timer := time.NewTimer(d)
	OUTER:
		for {
			select {
			case <-timer.C:
				logger.Printf("Queries executed for at least %v", d)
				break OUTER
			default:
				if err := runQuery(); err != nil {
					return err
				}
			}
		}
	}
	totalTime = time.Since(now)

	if totalTime < time.Minute && cpu {
		fmt.Fprintf(stderr, "\n***** NOTICE - QUERY EXECUTION %v *****\nThis tool works most effectively if queries are executed for at least one minute\nwhen capturing CPU profiles. Consider increasing `-n` or setting `-t 1m`.\n\n", totalTime)
	}

	// Take the final profiles
	logger.Print("Taking final profiles...")
	for _, p := range profiles {
		if err := captureProfile(p); err != nil {
			return err
		}
	}
	logger.Printf("All profiles gathered and saved at %s. Total query executions: %d.", archivePath, totalExecutions)

	// Finally, write the general data about the running of this program.
	err := tw.WriteHeader(&tar.Header{
		Name:    "info.txt",
		Mode:    0600,
		ModTime: time.Now().UTC(),
		Size:    int64(infoBuf.Len()),
	})
	if err != nil {
		return err
	}

	// Write the info data to the tar writer.
	if _, err = io.Copy(tw, &infoBuf); err != nil {
		return err
	}

	// Close the tar writer.
	if err := tw.Close(); err != nil {
		return err
	}

	// Close the gzip writer.
	if err := gz.Close(); err != nil {
		return err
	}

	fd, err := os.Create(filepath.Join(out, "profiles.tar.gz"))
	if err != nil {
		return err
	}
	defer fd.Close()

	_, err = io.Copy(fd, &allBuf)
	return err
}

// takeProfile takes the named profile and writes the result to w.
func takeProfile(w io.Writer, name string, debug int) error {
	u, err := url.Parse(host)
	if err != nil {
		return err
	}
	u.Path = path.Join("/debug/pprof/", name)

	if debug > 0 {
		q := url.Values{"debug": []string{fmt.Sprint(debug)}}
		u.RawQuery = q.Encode()
	}

	resp, err := http.Get(u.String())
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, err = io.Copy(w, resp.Body)
	return err
}

// NewClient returns a new InfluxDB client, which has successfully connected to
// a running server.
func NewClient() (client.Client, error) {
	c, err := client.NewHTTPClient(client.HTTPConfig{
		Addr:     host,
		Username: user,
		Password: pass,
	})
	if err != nil {
		return nil, err
	}

	dur, _, err := c.Ping(2 * time.Second)
	if err != nil {
		return nil, err
	}

	logger.Printf("Host %s responded to a ping in %v\n", host, dur)
	return c, nil
}

// runQuery executes query against the cluster.
func runQuery() error {
	totalExecutions++

	now := time.Now()
	defer func() {
		took := time.Since(now)
		logger.Print(fmt.Sprintf("Query %q took %v to execute.", query, took))
	}()

	resp, err := clt.Query(client.NewQuery(query, db, ""))
	if err != nil {
		return err
	}
	return resp.Error()
}
