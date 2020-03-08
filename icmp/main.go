package main

import (
	"bufio"
	"database/sql"
	"flag"
	"io/ioutil"
	"log"
	"os/exec"
	"strconv"
	"strings"

	_ "github.com/lib/pq"
	yaml "gopkg.in/yaml.v2"
)

var db *sql.DB

type config struct {
	DB        string   `yaml:"db"`
	FPingPath string   `yaml:"fping_path"`
	Targets   []string `yaml:"targets"`
}

type data struct {
	target string
	sent   int
	recv   int
	loss   int
	min    string
	max    string
	avg    string
}

func slashSplitter(c rune) bool {
	return c == '/'
}

func readPoints(conf config) {
	args := []string{
		"--backoff=1",
		"--timestamp",
		"--retry=0",
		"--tos=0",
		"--squiet=60",
		"--period=6000",
		"--loop",
	}
	for _, row := range conf.Targets {
		args = append(args, row)
	}

	cmd := exec.Command(conf.FPingPath, args...)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatal(err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		log.Fatal(err)
	}

	if err := cmd.Start(); err != nil {
		log.Fatal(err)
	}

	buff := bufio.NewScanner(stderr)
	for buff.Scan() {
		text := buff.Text()
		fields := strings.Fields(text)

		// Ignore timestamp
		if len(fields) == 1 {
			continue
		}

		writeData := data{
			target: fields[0],
		}

		data := fields[4]
		dataSplitted := strings.FieldsFunc(data, slashSplitter)
		// remove comma char
		dataSplitted[2] = strings.TrimRight(dataSplitted[2], "%,")

		writeData.sent, _ = strconv.Atoi(dataSplitted[0])
		writeData.recv, _ = strconv.Atoi(dataSplitted[1])
		writeData.loss, _ = strconv.Atoi(dataSplitted[2])

		// Ping times
		if len(fields) > 5 {
			times := fields[7]
			td := strings.FieldsFunc(times, slashSplitter)
			writeData.min = td[0]
			writeData.avg = td[1]
			writeData.max = td[2]
		}

		log.Printf(
			"target:%s, sent:%d, recv:%d, loss:%d, min:%s, avg:%s, max:%s",
			writeData.target,
			writeData.sent, writeData.recv, writeData.loss,
			writeData.min, writeData.avg, writeData.max,
		)

		writePoints(writeData)
	}

	std := bufio.NewReader(stdout)
	line, err := std.ReadString('\n')
	if err != nil {
		log.Fatal(err)
	}

	log.Println("stdout:", line)
}

func writePoints(row data) {
	if row.min != "" && row.max != "" && row.avg != "" {
		min, _ := strconv.ParseFloat(row.min, 64)
		max, _ := strconv.ParseFloat(row.max, 64)
		avg, _ := strconv.ParseFloat(row.avg, 64)

		stmt, err := db.Prepare("INSERT INTO measurement_icmp (time, target, sent, recv, loss, min, avg, max) VALUES (now(), $1, $2, $3, $4, $5, $6, $7)")
		if err != nil {
			log.Fatal(err)
		}

		if _, err := stmt.Exec(row.target, row.sent, row.recv, row.loss, min, max, avg); err != nil {
			log.Fatal(err)
		}
	} else {
		stmt, err := db.Prepare("INSERT INTO measurement_icmp (time, target, sent, recv, loss) VALUES (now(), $1, $2, $3, $4)")
		if err != nil {
			log.Fatal(err)
		}

		if _, err := stmt.Exec(row.target, row.sent, row.recv, row.loss); err != nil {
			log.Fatal(err)
		}
	}
}

func main() {
	var conf config

	configName := flag.String("config", "config.yaml", "Path to config file")
	flag.Parse()

	yamlFile, err := ioutil.ReadFile(*configName)
	if err != nil {
		log.Fatal(err)
	}

	if err := yaml.Unmarshal(yamlFile, &conf); err != nil {
		log.Fatal(err)
	}

	db, err = sql.Open("postgres", conf.DB)
	if err != nil {
		log.Fatal(err)
	}

	if err := db.Ping(); err != nil {
		log.Fatal(err)
	}

	readPoints(conf)
}
