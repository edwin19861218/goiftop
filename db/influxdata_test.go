package db

import (
	"math/rand"
	"testing"
)

func TestWriteInflux(t *testing.T) {
	client, _ := New("http://10.8.1.132:8086/?token=my-super-secret-auth-token&bucket=my-bucket&org=my-org")
	defer client.Close()
	for i := 1; i < 10; i++ {
		client.Write("tcp", "10.8.1.131:6865", "10.8.1.132:9999(distinct)", int64(259+rand.Int31n(1000)), int64(259+rand.Int31n(1000)))
	}
	client.WriteFlush()
}
