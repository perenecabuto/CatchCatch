package main

import (
	"fmt"
	"log"
	"net"
	"strings"
	"time"

	"github.com/tidwall/gjson"
)

type EventStream interface {
	StreamNearByEvents(nearByKey, roamKey string, meters int, callback DetectionHandler) error
}

type Tile38EventStream struct {
	addr string
}

func NewEventStream(addr string) EventStream {
	return &Tile38EventStream{addr}
}

type DetectionHandler func(*Detection)

// StreamGeofenceEvents ...
func (es *Tile38EventStream) StreamNearByEvents(nearByKey, roamKey string, callback DetectionHandler) error {
	conn, err := net.Dial("tcp", es.addr)
	if err != nil {
		return err
	}
	defer conn.Close()

	cmd := fmt.Sprintf("NEARBY %s FENCE ROAM %s * 1000\r\n", nearByKey, roamKey)
	log.Println("REDIS DEBUG:", cmd)
	if _, err = fmt.Fprintf(conn, cmd); err != nil {
		return err
	}
	buf := make([]byte, 4096)
	n, err := conn.Read(buf)
	res := string(buf[:n])
	if res != "+OK\r\n" {
		return fmt.Errorf("expected OK, got '%v'", res)
	}

	t := time.NewTicker(100 * time.Microsecond)
	for range t.C {
		if n, err = conn.Read(buf); err != nil {
			return err
		}
		for _, line := range strings.Split(string(buf[:n]), "\n") {
			if len(line) == 0 || line[0] != '{' {
				continue
			}
			detected, err := handleDetection(line)
			if err != nil {
				log.Println("Failed to handleDetection:", err)
				continue
			}
			callback(detected)
		}
	}

	return nil
}

type Detection struct {
	FeatID       string  `json:"feat_id"`
	CheckpointID string  `json:"checkpoint_id"`
	Lon          float64 `json:"lon"`
	Lat          float64 `json:"lat"`
	Distance     float64 `json:"distance"`
}

type DetectionError string

func (err DetectionError) Error() string {
	return string("DetectionError: " + err)
}

func handleDetection(msg string) (*Detection, error) {
	featID, coords := gjson.Get(msg, "id").String(), gjson.Get(msg, "object.coordinates").Array()
	if len(coords) != 2 {
		return nil, DetectionError("invalid coords - msg:\n" + msg)
	}
	lon, lat := coords[0].Float(), coords[1].Float()
	checkpointID, distance := gjson.Get(msg, "nearby.id").String(), gjson.Get(msg, "nearby.meters").Float()
	return &Detection{featID, checkpointID, lon, lat, distance}, nil
}
