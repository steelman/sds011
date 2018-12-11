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
	"os"
	"time"

	"github.com/ryszard/sds011/go/sds011"
)

var interval time.Duration

var (
	interval = flag.Duration("interval", 0, "measurement interval (e.g. 30s, 15m, 1h20m)")
	portPath = flag.String("port_path", "/dev/ttyUSB0", "serial port path")
	samples = flag.Int("samples", 1, "number of samples per measurement")
	unix = flag.Bool("unix", false, "print timestamps as number of seconds since 1970-01-01 00:00:00 UTC")
)

func init() {
	flag.Usage = func() {
		fmt.Fprint(os.Stderr,
			`sds011 reads data from the SDS011 sensor and sends them to stdout as CSV.

The columns are: an RFC3339 timestamp, the PM2.5 level, the PM10 level.`)
		fmt.Fprintf(os.Stderr, "\n\nUsage of %s:\n", os.Args[0])
		flag.PrintDefaults()
	}

}

func main() {
	flag.Parse()

	sensor, err := sds011.New(*portPath)
	if err != nil {
		log.Fatal(err)
	}
	defer sensor.Close()

	for {
		var pm10, pm25 float64
		var ts string
		var t1 time.Time

		t1 = time.Now()
		sensor.Awake()
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
		fmt.Fprintf(os.Stdout, "%s,%.2f,%.2f\n", ts, pm25 / float64(*samples), pm10 / float64(*samples))

		if (interval > 1 * time.Second) {
			sensor.Sleep()
			time.Sleep(time.Until(t1.Add(interval)))
		}
	}
}
