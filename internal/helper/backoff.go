package helper

import "time"

func ExponentialBackoff(sleepTime *time.Duration) {
	*sleepTime *= 2

	if *sleepTime >= 30*time.Second {
		*sleepTime = 30 * time.Second
	}
	time.Sleep(*sleepTime)
}
