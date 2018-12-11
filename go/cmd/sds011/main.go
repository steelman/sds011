// Copyright 2017 Ryszard Szopa <ryszard.szopa@gmail.com>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// sds011 is a simple reader for the SDS011 Air Quality Sensor. It
// outputs data in TSV to standard output (timestamp formatted
// according to RFC3339, PM2.5 levels, PM10 levels).
package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/ryszard/sds011/go/sds011"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var interval time.Duration

var (
	interval = flag.Duration("interval", 0, "measurement interval (e.g. 30s, 15m, 1h20m)")
	portPath = flag.String("port_path", "/dev/ttyUSB0", "serial port path")
	samples = flag.Int("samples", 1, "number of samples per measurement")
	unix = flag.Bool("unix", false, "print timestamps as number of seconds since 1970-01-01 00:00:00 UTC")
	addr = flag.String("listen-address", "", "The address to listen on for HTTP requests.")
)

var (
	pm25mt = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "pm25",
			Help: "Data from PM2.5 sensor",
		},
	)
	pm10mt = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "pm10",
			Help: "Data from PM10 sensor",
		},
	)
)

func init() {
	flag.Usage = func() {
		fmt.Fprint(os.Stderr,
			`sds011 reads data from the SDS011 sensor and sends them to stdout as CSV.

The columns are: an RFC3339 timestamp, the PM2.5 level, the PM10 level.`)
		fmt.Fprintf(os.Stderr, "\n\nUsage of %s:\n", os.Args[0])
		flag.PrintDefaults()
	}

	prometheus.MustRegister(pm25mt)
	prometheus.MustRegister(pm10mt)
}

func listen_http() {
	// Expose the registered metrics via HTTP.
	http.Handle("/metrics", promhttp.Handler())
	log.Fatal(http.ListenAndServe(*addr, nil))
}

func main() {
	flag.Parse()

	if (len(*addr) > 0) {
		go listen_http()
	}

	sensor, err := sds011.New(*portPath)
	if err != nil {
		log.Fatal(err)
	}
	defer sensor.Close()

	for {
		var pm10, pm25 float64
		var ts string
		var t1 time.Time
		var awake bool

		if awake, err = sensor.IsAwake(); !awake {
			sensor.Awake()
		}

		t1 = time.Now()
		for i:=0; i < *samples; i++ {
			point, err := sensor.Get()
			if err != nil {
				log.Printf("ERROR: sensor.Get: %v", err)
				continue
			}
			pm10 += point.PM10
			pm25 += point.PM25
			if *unix {
				ts = fmt.Sprintf("%v", point.Timestamp.Unix())
			} else {
				ts = point.Timestamp.Format(time.RFC3339)
			}
		}

		pm25 = pm25 / float64(*samples)
		pm10 = pm10 / float64(*samples)
		fmt.Fprintf(os.Stdout, "%s,%.2f,%.2f\n", ts, pm25, pm10)
		pm10mt.Set(pm10)
		pm25mt.Set(pm25)

		if (interval > 1 * time.Second) {
			sensor.Sleep()
			time.Sleep(time.Until(t1.Add(interval)))
			sensor.Awake()
		}
	}
}
