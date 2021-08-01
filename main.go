package main

import (
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"github.com/OneOfOne/xxhash"
	"io"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	s1ajob    = "JOB,%v,%v"
	xxhjob    = "JOBXX,%v,%v"
	minername = "dgm"
	report    = "%v,%v,%v %v,%v"
	SEPERATOR = ","
	NEWLINE   = "\n"
	NULL      = "\x00"
	BUF_SIZE  = 256
)

var (
	server  = flag.String("server", os.Getenv("DUCOSERVER"), "Server Address and Port, environment variable DUCOSERVER")
	name    = flag.String("name", os.Getenv("MINERNAME"), "Miner Name, enviromnet variable MINERNAME")
	id      = flag.String("id", os.Getenv("HOSTNAME"), "Rig ID, environment variable HOSTNAME")
	diff    = flag.String("diff", os.Getenv("DIFF"), "Difficulty LOW/MEDIUM/NET, environment variable DIFF")
	algo    = flag.String("algo", os.Getenv("ALGO"), "Algorithm select xxhash/ducos1a, environment variable ALGO")
	quiet   = flag.Bool("quiet", false, "Turn off Console Logging")
	debug   = flag.Bool("debug", false, "console log send/receive messages.")
	skip    = flag.Bool("skip", false, "Skip the first 'Difficulty' Hash Range")
	threads = flag.Int("threads", 1, "Number of Threads to Run")
	version = "0.2"
)

func init() {}

func setDefaults() {
	if *name == "" {
		flag.PrintDefaults()
		os.Exit(1)
	}

	if *server == "" {
		*server = "149.91.88.18:6000"
	}

	if *algo == "" {
		*algo = "ducos1a"
	}

	if *diff == "" {
		*diff = "MEDIUM"
	}

	if *id == "" {
		*id = "SETID"
	}

	if *threads <= 0 {
		*threads = 1
	}
}

func connect(TID int) (conn net.Conn, err error) {
	logger(fmtID(TID), "Connecting to Server: ", *server)

	conn, err = net.Dial("tcp", *server)
	if err != nil {
		return
	}

	resp, err := read(conn, TID)
	if err != nil {
		return
	}

	logger(fmtID(TID), "Connected to Server Version: ", resp)

	return
}

func main() {
	flag.Parse()
	setDefaults()

	logger("Starting ds-go-miner version ", version)

	var wg sync.WaitGroup

	for i := 0; i < *threads; i++ {
		wg.Add(1)
		go workLoop(i, &wg)
	}

	wg.Wait()
}

func workLoop(TID int, wg *sync.WaitGroup) {
	defer wg.Done()

	conn, err := connect(TID)
	if err != nil {
		fmt.Println("error", err)
		return
	}

	for {
		job := &Job{
			Algorithm: *algo,
			TID:       TID,
		}

		err = job.getJob(conn)
		if err != nil {
			loggerDebug(fmtID(TID), "error with getjob ", err)
			if err != io.EOF {
				conn.Close()
			}

			if conn != nil {
				conn = nil
			}

			conn, _ = connect(TID)
			continue
		}

		err = job.ducoJob()
		if err != nil {
			loggerDebug(fmtID(TID), "error with ducoJob ", err)
			continue
		}

		err = job.reportJob(conn)
		if err != nil {
			loggerDebug(fmtID(TID), "error with reportJob ", err)
			continue
		}
	}
}

func (j *Job) getJob(conn net.Conn) (err error) {
	var getjobrequest string
	switch j.Algorithm {
	case "xxhash":
		getjobrequest = fmt.Sprintf(xxhjob, *name, *diff)
	default:
		getjobrequest = fmt.Sprintf(s1ajob, *name, *diff)
	}

	err = send(conn, getjobrequest, j.TID)
	if err != nil {
		return
	}

	resp, err := read(conn, j.TID)
	if err != nil {
		return
	}

	logger(fmtID(j.TID), "Get Job Response ", resp)

	str := strings.Split(resp, SEPERATOR)
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

// parses string to uint64 base 10
func parseUint(str string) (uint64, error) {
	return strconv.ParseUint(str, 10, 64)
}

//Reports the Job Result
func (j *Job) reportJob(conn net.Conn) (err error) {
	nonce := strconv.FormatUint(j.Nonce, 10)
	rate := 0
	ID := fmt.Sprintf("%vx%v", j.TID, *id)
	rpt := fmt.Sprintf(report, nonce, rate, minername, version, ID)

	err = send(conn, rpt, j.TID)
	if err != nil {
		return
	}

	resp, err := read(conn, j.TID)
	if err != nil {
		return
	}

	logger(fmtID(j.TID), "Submit Job Response: ", resp)

	return
}

//Job is a struct for the job
type Job struct {
	Algorithm     string
	AlgoFunc      func(*Job) error
	NewBlock      string
	ExpectedBlock string
	Result        string
	Difficulty    uint64
	Efficency     float32
	Nonce         uint64
	Sum64         uint64
	TID           int
}

func (j *Job) ducoJob() (err error) {
	//Set the difficulty
	if *skip {
		j.Nonce = j.Difficulty
	}

	j.Difficulty = j.Difficulty*100 + 1

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

	if *skip && j.Nonce >= j.Difficulty {
		//Getting here means searching the space prior to skip
		j.Difficulty = (j.Difficulty - 1) / 100
		j.Nonce = 0
		loggerDebug(fmtID(j.TID), "searching skipped space ", j.Nonce, " ", j.Difficulty)
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
		if err != nil || j.Result == j.ExpectedBlock {
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
	if *quiet {
		return
	}

	tm := time.Now().Format(time.RFC3339)
	fmt.Printf("[%s] ", tm)

	for _, v := range msg {
		fmt.Print(v)
	}

	fmt.Println()
}

func loggerDebug(msg ...interface{}) {
	if !*debug {
		return
	}

	dbgmsg := []interface{}{"[DEBUG] "}
	msg = append(dbgmsg, msg...)

	logger(msg...)
}

// cleanString cleans a string
func cleanString(str string) (ret string) {
	ret = strings.TrimRight(str, NULL)
	ret = strings.TrimRight(ret, NEWLINE)
	return
}

// read is a helper for reciving a string
func read(conn net.Conn, TID int) (ret string, err error) {
	buf := make([]byte, BUF_SIZE)
	n, err := conn.Read(buf)

	//if error, or no bytes read
	if err != nil || n <= 0 {
		return
	}

	ret = cleanString(string(buf))
	loggerDebug(fmtID(TID), "read ", n, " bytes ", ret)
	return
}

// send is a helper for sending a string
func send(conn net.Conn, str string, TID int) (err error) {
	n, err := fmt.Fprintln(conn, str)
	loggerDebug(fmtID(TID), "send ", n, " bytes ", str)
	return
}

// Quick Helper Function to Format Thread ID Logging
func fmtID(id int) string {
	return fmt.Sprintf("[Thread %v] ", id)
}
