package main

import (
	"bufio"
	"context"
	"flag"
	"log"
	"log/slog"
	"net"
	"os"
	"regexp"
	"strconv"
	"syscall"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/pkg/errors"
)

type Option struct {
	server         net.TCPAddr
	password       string
	containerID    string
	timeOutSeconds int
}

func loadOption() Option {
	s := flag.String("server", "127.0.0.1:12345", "server:port")
	p := flag.String("password", "", "rcon password")
	c := flag.String("container", "", "factorio container Id")
	t := flag.Int("timeout", 0, "Timeout in seconds for waiting the RCON server")

	flag.Parse()

	ip, err := net.ResolveTCPAddr("tcp", *s)
	if err != nil {
		log.Fatal(err)
	}

	return Option{server: *ip, password: *p, containerID: *c, timeOutSeconds: *t}
}

type FactorioState struct {
	peerID   uint64
	oldState string
	newState string
}

func waitStartRconServer(ctx context.Context, server net.TCPAddr) error {
	interval := 2 * time.Second

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			conn, err := net.DialTCP("tcp", nil, &server)
			if t, ok := err.(*net.OpError); ok {
				if tt, ok := t.Err.(*os.SyscallError); ok && tt.Err == syscall.ECONNREFUSED {
					slog.Info("waiting rcon server...")
					time.Sleep(interval)
					continue
				}
			}

			if err != nil {
				return err
			}

			conn.Close()
			return nil
		}
	}
}

func waitEvent(ctx context.Context, docker *client.Client, containerID string, c chan interface{}) {
	defer close(c)
	containerOptions := types.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     true,
		Tail:       "0",
	}

	r, err := docker.ContainerLogs(ctx, containerID, containerOptions)
	if err != nil {
		log.Fatal(err)
	}
	defer r.Close()

	reader := bufio.NewReader(r)

	re := regexp.MustCompile(`received stateChanged peerID\((?P<peerID>\d+)\) oldState\((?P<oldState>\w+)\) newState\((?P<newState>\w+)\)`)

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			c <- errors.Wrap(err, "Error reading log")
			return
		}

		m := re.FindStringSubmatch(line)
		if len(m) > 0 {
			peerID, err := strconv.ParseUint(m[re.SubexpIndex("peerID")], 10, 64)
			if err != nil {
				c <- errors.Wrap(err, "failed parse peerID")
				return
			}
			oldState := m[re.SubexpIndex("oldState")]
			newState := m[re.SubexpIndex("newState")]
			c <- FactorioState{peerID: peerID, oldState: oldState, newState: newState}
			continue
		}
	}
}

func main() {
	opt := loadOption()

	ctx := context.Background()

	docker, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		log.Fatal(err)
	}
	docker.NegotiateAPIVersion(ctx)

	{
		timeoutCtx := ctx
		var cancel context.CancelFunc
		if opt.timeOutSeconds > 0 {
			timeoutCtx, cancel = context.WithTimeout(ctx, time.Duration(opt.timeOutSeconds)*time.Second)
			defer cancel()
		}

		if err := waitStartRconServer(timeoutCtx, opt.server); err != nil {
			log.Fatal(err)
		}
	}

	factorio, err := NewFactorioRcon(opt.server.String(), opt.password)
	if err != nil {
		log.Fatal(err)
	}

	loadingPeerState := make(map[uint64]FactorioState)
	ch := make(chan interface{})
	go waitEvent(ctx, docker, opt.containerID, ch)
	for {
		e, ok := <-ch
		if !ok {
			break
		}

		switch v := e.(type) {
		case FactorioState:
			slog.Info("Updated", "FactorioState", v)

			if v.newState == "ConnectedDownloadingMap" {
				loadingPeerState[v.peerID] = v
				slog.Info("Execute Pause.")
				if err := factorio.Shout("Pause the game for player loading."); err != nil {
					log.Fatal(err)
				}

				if err := factorio.Pause(); err != nil {
					log.Fatal(err)
				}
			}
			if len(loadingPeerState) == 0 {
				continue
			}

			if v.newState == "WaitingForCommandToStartSendingTickClosures" {
				delete(loadingPeerState, v.peerID)
			}
			if v.newState == "DisconnectScheduled" {
				delete(loadingPeerState, v.peerID)
			}

			if len(loadingPeerState) == 0 {
				slog.Info("Execute UnPause.")

				if err := factorio.UnPause(); err != nil {
					log.Fatal(err)
				}
			}
		default:
			log.Fatal(v)
		}
	}
}
