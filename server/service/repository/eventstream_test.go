package repository_test

import (
	"context"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/quorzz/redis-protocol"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	repo "github.com/perenecabuto/CatchCatch/server/service/repository"
)

func TestNearByPoint(t *testing.T) {
	addr := "localhost:9851"
	stream := repo.NewEventStream(addr)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cmds := []repo.CommandEvent{repo.Del, repo.Set}
	detect := []repo.DetectEvent{repo.Enter, repo.Inside, repo.Outside,
		repo.Exit}
	lat, lon, dist := 1.0, 1.5, 0.0

	resultChan := make(chan *repo.Detection, 1)
	cb := func(d *repo.Detection) error {
		go func() { resultChan <- d }()
		return nil
	}

	collection := "observed-type-objs"

	go func() {
		err := stream.StreamNearByPoint(ctx, collection, cmds, detect, lat, lon, dist, cb)
		require.NoError(t, err)
	}()

	conn, err := net.Dial("tcp", addr)
	require.NoError(t, err)
	defer conn.Close()
	err = conn.SetWriteDeadline(time.Now().Add(time.Second))
	require.NoError(t, err)
	cmd, _ := protocol.PackCommand("SET", collection, "obj1", "POINT", lat, lon)
	for i := 0; i < 100; i++ {
		go fmt.Fprint(conn, string(cmd))
	}

	select {
	case d := <-resultChan:
		assert.Equal(t, "obj1", d.FeatID)
		assert.Equal(t, repo.Inside, d.Intersects)
	case <-time.NewTimer(time.Second * 5).C:
		t.Error("timeout")
	}
}
