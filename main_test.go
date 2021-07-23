package main

import (
	"testing"
)

var ()

const (
	lastblock = "416dc20fb261ec2dcf72147be57efc372fb765b1"
	newblock = "dfa67daef0bbac93da38772c7bbd6e28b839bc43"
	validateDiff = uint64(175514)
	difficulty = uint64(7500)
	xxLast = "f48abd686b70ffd5615fbd8c6aa8156c0425b09b"
	xxExpected = "74c5967877c25e22"
	xxDiff = uint64(100000)
	xxValidateDiff = uint64(4069510)
)

func TestHashingJobS1A(t *testing.T) {
	job := &Job{
		Algorithm: "ducos1a",
		NewBlock: lastblock,
		ExpectedBlock: newblock,
		Difficulty: difficulty,
		Nonce: validateDiff,
	}

	err := job.ducoJob()

	if err != nil {
		t.Errorf("error returned %s", err)
	}

	if job.Nonce != validateDiff {
		t.Errorf("Validate Hash Failed Got %v wanted %v", job.Nonce, validateDiff)
	}
}

func TestHashingJobXXH(t *testing.T) {
	job := &Job{
		Algorithm: "xxhash",
		NewBlock: xxLast,
		ExpectedBlock: xxExpected,
		Difficulty: xxDiff,
		Nonce: xxValidateDiff,
	}

	err := job.ducoJob()

	if err != nil {
		t.Errorf("error returned %s", err)
	}

	if job.Nonce != xxValidateDiff {
		t.Errorf("Validate Hash Failed Got %v wanted %v", job.Nonce, validateDiff)
	}
}

func BenchmarkDUCOS1A(b *testing.B) {
	job := &Job{
		NewBlock: lastblock,
		ExpectedBlock: newblock,
		Difficulty: difficulty,
	}

	for i := 0; i < b.N; i++ {
		job.Nonce += uint64(i)
		ducos1a(job)
	}
}

func BenchmarkDUCOS1A2(b *testing.B) {
	job := &Job{
		NewBlock: lastblock,
		ExpectedBlock: newblock,
		Difficulty: difficulty,
	}

	for i := 0; i < b.N; i++ {
		job.Nonce += uint64(i)
		ducos1a2(job)
	}
}

func BenchmarkDUCOS1A3(b *testing.B) {
	job := &Job{
		NewBlock: lastblock,
		ExpectedBlock: newblock,
		Difficulty: difficulty,
	}

	for i := 0; i < b.N; i++ {
		job.Nonce += uint64(i)
		ducos1a3(job)
	}
}

func BenchmarkXXHash(b *testing.B) {
	job := &Job{
		NewBlock: xxLast,
		ExpectedBlock: xxExpected,
		Difficulty: xxDiff,
	}

	for i := 0 ; i < b.N ; i++ {
		job.Nonce += uint64(i)
		ducos1xxh(job)
	}
}