package main

import (
	"io"
	"os"
	"fmt"
	"net"
	"flag"
	"errors"
	"strings"
	"strconv"
	"crypto/sha1"
	"encoding/hex"
	"github.com/OneOfOne/xxhash"
)

const ()

var (
	server = flag.String("server", os.Getenv("DUCOSERVER"), "Server Address and Port, environment variable DUCOSERVER")
	name = flag.String("name", os.Getenv("MINERNAME"), "Miner Name, enviromnet variable MINERNAME")
	id = flag.String("id", os.Getenv("HOSTNAME"), "Rig ID, environment variable HOSTNAME")
	diff = flag.String("diff", os.Getenv("DIFF"), "Difficulty LOW/MEDIUM/NET, environment variable DIFF")
	algo = flag.String("algo", os.Getenv("ALGO"), "Algorithm select xxhash/ducos1a, environment variable ALGO")
	quiet = flag.Bool("quiet", false, "Turn off Console Logging")
	jobStr string
	minername = "ds-go-miner"
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

func main() {
	flag.Parse()
	setDefaults()

	logger("Connecting to Server")
	logger(*server)

	conn, err := net.Dial("tcp", *server)
	if err != nil {
		fmt.Println("error", err)
		os.Exit(1)
	}

	buff := make([]byte, 1024)
	_, err = conn.Read(buff)
	if err != nil {
		fmt.Println("error", err)
		os.Exit(1)
	}

	logger("Connected to Server")
	logger(string(buff))

	defer conn.Close()

	switch (*algo) {
	case "ducos1a":
		jobStr = "JOB,%s,%s\n"
	case "xxhash":
		jobStr = "JOBXX,%s,%s\n"
	default:
		logger("error in setting job string")
		os.Exit(1)
	}

	for {
		buff = make([]byte, 1024)
		fmt.Fprintf(conn, jobStr, *name, *diff) //lint:ignore //false SA1006
		
		_, err = conn.Read(buff)
		if err != nil {
			continue
		}

		logger(string(buff))

		str := strings.Split(string(buff), ",")
		if len(str) < 2 {
			logger("len problem")
			continue
		}

		str[2] = strings.TrimRight(str[2], "\x00")
		str[2] = strings.TrimRight(str[2], "\n")
		difficulty, err := strconv.ParseUint(str[2], 10, 64)
		if err != nil {
			fmt.Println("strconv error", err)
			continue
		}

		job := Job{
			Algorithm: *algo,
			NewBlock: str[0],
			ExpectedBlock: str[1],
			Difficulty: difficulty,
		}

		err = job.ducoJob()
		if err != nil {
			logger(err)
			continue
		}

		err = job.reportJob(conn)
		if err != nil {
			logger(err)
			continue
		}

	}
}

//Reports the Job Result
func (j *Job) reportJob(conn net.Conn) (err error) {
	nonce := strconv.FormatUint(j.Nonce, 10)
	rate := 0
	ver := fmt.Sprintf("%s %s", minername, version)

	rpt := fmt.Sprintf("%v,%v,%v,%v\n", nonce, rate, ver, *id)
	_, err = fmt.Fprintf(conn, rpt)
	if err != nil {
		return
	}

	buf := make([]byte, 1024)
	_, err = conn.Read(buf)
	if err != nil {
		return
	}

	logger(string(buf))

	return
}

//Job is a struct for the job
type Job struct {
	Algorithm string
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
	j.Difficulty = j.Difficulty * 100 + 1

	//Set the algo function
	var f func(*Job) error
	switch j.Algorithm {
		case "xxhash":
			f = func(j *Job) (err error) {
				return ducos1xxh(j)
			}
		case "ducos1a":
			f = func(j *Job) (err error) {
				return ducos1a(j)
			}
		default:
			return errors.New("unimplemented algo")
	}

	//Main job
	for j.Nonce = 0; j.Nonce < j.Difficulty; j.Nonce++ {

		err = f(j)

		if (err != nil || j.Result == j.ExpectedBlock) {
			//j.Nonce should be the answer.
			break;
		}
	}

	return
}

// About the same as the first for performance, less readable.
func ducos1a3(j *Job) (err error) {
	data := []byte(j.NewBlock + strconv.FormatUint(j.Nonce, 10) )
	j.Result = fmt.Sprintf("%x", (sha1.Sum(data)))
	return
}

// Benchmark differences, this one is slower.
func ducos1a2(j *Job) (err error) {
	num := strconv.FormatUint(j.Nonce, 10)
	data := []byte(j.NewBlock + num)
	h := sha1.New()
	h.Write(data)
	j.Result = hex.EncodeToString(h.Sum(nil))
	return
}

// About the same, but more readable.
func ducos1a(j *Job) (err error) {
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

func logger (msg interface{}) {
	if *quiet {
		return
	}

	fmt.Print(msg)
}
