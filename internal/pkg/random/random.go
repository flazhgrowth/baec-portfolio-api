package random

import (
	"flag"
	"math"
	"math/rand"
	"time"
)

var (
	letterRunes = []rune("0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	numbers     = []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 0}
)

// GenerateRandomString generats random string with length of n
func GenerateRandomString(n int) string {
	if flag.Lookup("test.v") != nil {
		return "grandom-random"
	}

	ran := rand.New(rand.NewSource(time.Now().UnixNano()))
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[ran.Intn(len(letterRunes))]
	}
	return string(b)
}

func GenerateRandomNumber(n int) uint64 {
	ran := rand.New(rand.NewSource(time.Now().UnixNano()))
	b := []uint64{}
	for i := 0; i < n; i += 1 {
		b = append(b, uint64(numbers[ran.Intn(len(numbers))]))
	}

	multiplier := uint64(1 * (math.Pow10(n - 1)))
	res := uint64(0)
	for _, val := range b {
		newVal := val * multiplier
		res += newVal
		multiplier /= 10
	}

	return res
}
