// START: distributed_log_test_intro
package log_test

import (
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"reflect"
	"testing"
	"time"

	api "github.com/YeonwooSung/dislog/api/v1"
	"github.com/YeonwooSung/dislog/internal/log"
	"github.com/hashicorp/raft"
	"github.com/stretchr/testify/require"
	"github.com/travisjeffery/go-dynaport"
)

func TestMultipleNodes(t *testing.T) {
	var logs []*log.DistributedLog
	nodeCount := 3
	ports := dynaport.Get(nodeCount)

	for i := 0; i < nodeCount; i++ {
		dataDir, err := ioutil.TempDir("", "distributed-log-test")
		require.NoError(t, err)
		defer func(dir string) {
			_ = os.RemoveAll(dir)
		}(dataDir)

		ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", ports[i]))
		require.NoError(t, err)

		config := log.Config{}
		config.Raft.StreamLayer = log.NewStreamLayer(ln, nil, nil)
		config.Raft.LocalID = raft.ServerID(fmt.Sprintf("%d", i))
		config.Raft.HeartbeatTimeout = 50 * time.Millisecond
		config.Raft.ElectionTimeout = 50 * time.Millisecond
		config.Raft.LeaderLeaseTimeout = 50 * time.Millisecond
		config.Raft.CommitTimeout = 5 * time.Millisecond
		config.Raft.BindAddr = ln.Addr().String()
		// END: distributed_log_test_intro

		// START: distributed_log_test_cont

		if i == 0 {
			config.Raft.Bootstrap = true
		}

		l, err := log.NewDistributedLog(dataDir, config)
		require.NoError(t, err)

		if i != 0 {
			err = logs[0].Join(
				fmt.Sprintf("%d", i), ln.Addr().String(),
			)
			require.NoError(t, err)
		} else {
			err = l.WaitForLeader(3 * time.Second)
			require.NoError(t, err)
		}

		logs = append(logs, l)
	}
	// END: distributed_log_test_cont

	// START: distributed_log_test_replicate
	records := []*api.Record{
		{Value: []byte("first")},
		{Value: []byte("second")},
	}
	for _, record := range records {
		off, err := logs[0].Append(record)
		require.NoError(t, err)

		require.Eventually(t, func() bool {
			for j := 0; j < nodeCount; j++ {
				got, err := logs[j].Read(off)
				if err != nil {
					return false
				}
				record.Offset = off
				if !reflect.DeepEqual(got.Value, record.Value) {
					return false
				}
			}
			return true
		}, 500*time.Millisecond, 50*time.Millisecond)
	}
	// END: distributed_log_test_replicate

	// START: get_servers
	servers, err := logs[0].GetServers()
	require.NoError(t, err)
	require.Equal(t, 3, len(servers))
	require.True(t, servers[0].IsLeader)
	require.False(t, servers[1].IsLeader)
	require.False(t, servers[2].IsLeader)

	// START: distributed_log_test_leave
	err = logs[0].Leave("1") //<label id="before_leave" />
	require.NoError(t, err)  //<label id="after_leave" />

	time.Sleep(50 * time.Millisecond) //<label id="second_leave" />

	servers, err = logs[0].GetServers()
	require.NoError(t, err)
	require.Equal(t, 2, len(servers))
	require.True(t, servers[0].IsLeader)
	require.False(t, servers[1].IsLeader)
	// END: get_servers

	off, err := logs[0].Append(&api.Record{
		Value: []byte("third"),
	})
	require.NoError(t, err)

	time.Sleep(50 * time.Millisecond)

	record, err := logs[1].Read(off)
	require.IsType(t, api.ErrOffsetOutOfRange{}, err)
	require.Nil(t, record)

	record, err = logs[2].Read(off)
	require.NoError(t, err)
	require.Equal(t, []byte("third"), record.Value)
	require.Equal(t, off, record.Offset) //<label id="second_leave" />
}

// END: distributed_log_test_leave
