package main

import (
	"io"
	"os"
	"fmt"
	"net"
	"flag"
	"time"
	"errors"
	"strings"
	"strconv"
	"crypto/sha1"
	"encoding/hex"
	"github.com/OneOfOne/xxhash"
)

const (
	s1ajob = "JOB,%v,%v"
	xxhjob = "JOBXX,%v,%v"
	minername = "dgm"
	report = "%v,%v,%v %v,%v"
	SEPERATOR = ","
	NEWLINE = "\n"
	NULL = "\x00"
)

var (
	server = flag.String("server", os.Getenv("DUCOSERVER"), "Server Address and Port, environment variable DUCOSERVER")
	name = flag.String("name", os.Getenv("MINERNAME"), "Miner Name, enviromnet variable MINERNAME")
	id = flag.String("id", os.Getenv("HOSTNAME"), "Rig ID, environment variable HOSTNAME")
	diff = flag.String("diff", os.Getenv("DIFF"), "Difficulty LOW/MEDIUM/NET, environment variable DIFF")
	algo = flag.String("algo", os.Getenv("ALGO"), "Algorithm select xxhash/ducos1a, environment variable ALGO")
	quiet = flag.Bool("quiet", false, "Turn off Console Logging")
	debug = flag.Bool("debug", false, "console log send/receive messages.")
	skip = flag.Bool("skip", false, "Skip the first 'Difficulty' Hash Range")
	version = "0.1"
)

func init() {}

func setDefaults() {
	if (*name == "") {
		flag.PrintDefaults()
		os.Exit(1)
	}

	if (*server == "") {
		*server = "149.91.88.18:6000"
	}

	if (*algo == "") {
		*algo = "ducos1a"
	}

	if (*diff == "") {
		*diff = "MEDIUM"
	}

	if (*id == "") {
		*id = "SETID"
	}
}

func connect() (conn net.Conn, err error) {
	logger("Connecting to Server: ", *server, NEWLINE)

	conn, err = net.Dial("tcp", *server)
	if err != nil {
		return
	}

	resp, err := read(conn)
	if err != nil {
		return
	}

	logger("Connected to Server Version: ", resp, NEWLINE)

	return
}

func main() {
	flag.Parse()
	setDefaults()

	logger("Starting ds-go-miner version ", version, NEWLINE)

	conn, err := connect()
	if err != nil {
		fmt.Println("error", err)
		os.Exit(1)
	}

	for {
		job := &Job{
			Algorithm: *algo,
		}

		err = job.getJob(conn)
		if err != nil {
			loggerDebug("error with getjob ", err)
			if err == io.EOF {
				conn, _ = connect()
			}
			continue
		}

		err = job.ducoJob()
		if err != nil {
			loggerDebug("error with ducoJob ", err)
			continue
		}

		err = job.reportJob(conn)
		if err != nil {
			loggerDebug("error with reportJob ", err)
			continue
		}
	}
}

func (j *Job) getJob(conn net.Conn) (err error) {
	var getjobrequest string
	switch (j.Algorithm) {
		case "xxhash":
			getjobrequest = fmt.Sprintf(xxhjob, *name, *diff)
		default:
			getjobrequest = fmt.Sprintf(s1ajob, *name, *diff)
	}

	err = send(conn, getjobrequest)
	if err != nil {
		return
	}

	resp, err := read(conn)
	if err != nil {
		return
	}

	logger("Get Job Response ", resp)

	str := strings.Split(resp, ",")
	if len(str) < 2 {
		return errors.New("str split error")
	}

	diff, err := parseUint(str[2])
	if err != nil {
		return
	}

	j.NewBlock = str[0]
	j.ExpectedBlock = str[1]
	j.Difficulty = diff

	return
}

// parses a string input, cleans it, and returns a uint64
func parseUint(str string) (ret uint64, err error) {
	str = strings.TrimRight(str, NULL)
	str = strings.TrimRight(str, NEWLINE)
	ret, err = strconv.ParseUint(str, 10, 64)
	return
}

//Reports the Job Result
func (j *Job) reportJob(conn net.Conn) (err error) {
	nonce := strconv.FormatUint(j.Nonce, 10)
	rate := 0
	rpt := fmt.Sprintf(report, nonce, rate, minername, version, *id)

	err = send(conn, rpt)
	if err != nil {
		return
	}

	resp, err := read(conn)
	if err != nil {
		return
	}

	logger("Submit Job Response: ", resp)

	return
}

//Job is a struct for the job
type Job struct {
	Algorithm string
	AlgoFunc func(*Job) error
	NewBlock string
	ExpectedBlock string
	Result string
	Difficulty uint64
	Efficency float32
	Nonce uint64
	Sum64 uint64
}

func (j *Job) ducoJob() (err error) {
	//Set the difficulty
	if (*skip) {
		j.Nonce = j.Difficulty
	}

	j.Difficulty = j.Difficulty * 100 + 1

	//Set the algo function
	//var f func(*Job) error
	switch j.Algorithm {
		case "xxhash":
			j.AlgoFunc = func(j *Job) (err error) {
				return ducos1xxh(j)
			}
		case "ducos1a":
			j.AlgoFunc = func(j *Job) (err error) {
				return ducos1a(j)
			}
		default:
			return errors.New("unimplemented algo")
	}

	//Main job Loop
	err = j.jobLoop()
	if err != nil {
		return
	}

	if (*skip && j.Nonce >= j.Difficulty) {
		//Getting here means searching the space prior to skip
		j.Difficulty = (j.Difficulty-1) / 100
		j.Nonce = 0
		loggerDebug("searching skipped space ", j.Nonce, " ", j.Difficulty, NEWLINE)
		err = j.jobLoop()
	}

	return
}

func (j *Job) jobLoop() (err error) {
	if j.AlgoFunc == nil {
		return errors.New("algo func nil")
	}

	for ; j.Nonce < j.Difficulty; j.Nonce++ {
		err = j.AlgoFunc(j)
		if (err != nil || j.Result == j.ExpectedBlock) {
			break
		}
	}
	return
}

func ducos1a(j *Job) (err error) {
	nonce := strconv.FormatUint(j.Nonce, 10)
	data := []byte(j.NewBlock + nonce)
	h := sha1.New()
	h.Write(data)
	j.Result = hex.EncodeToString(h.Sum(nil))
	return
}

func ducos1a2(j *Job) (err error) {
	nonce := strconv.FormatUint(j.Nonce, 10)
	data := []byte(j.NewBlock + nonce)
	sum := sha1.Sum(data)
	j.Result = fmt.Sprintf("%x", sum)
	return
}

func ducos1xxh(j *Job) (err error) {
	xx := xxhash.NewS64(uint64(2811))
	nonce := strconv.FormatUint(j.Nonce, 10)
	src := strings.NewReader(j.NewBlock + nonce)
	
	_, err = io.Copy(xx, src)

	if err != nil {
		return
	}

	sum := xx.Sum64()
	j.Result = fmt.Sprintf("%x", sum)
	return
}

// logger is the general purpose logger
// which can be turned off w/ cmd line switch
func logger(msg ...interface{}) {
	if (*quiet) {
		return
	}

	tm := time.Now().Format(time.RFC3339)
	fmt.Printf("[%s] ", tm)

	for _, v := range msg {
		fmt.Print(v)
	}
}

func loggerDebug(msg ...interface{}) {
	if (!*debug) {
		return
	}

	logger(msg...)
}

// read is a helper for reciving a string
func read(conn net.Conn) (ret string, err error) {
	buf := make([]byte, 128)
	_, err = conn.Read(buf)

	if err != nil {
		return
	}

	ret = string(buf)

	loggerDebug("read ", ret)
	return
}

// send is a helper for sending a string
func send(conn net.Conn, str string) (err error) {
	fmt.Fprintln(conn, str)
	loggerDebug("send ", str, NEWLINE)
	return
}
