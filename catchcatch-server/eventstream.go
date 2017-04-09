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
func (es *Tile38EventStream) StreamNearByEvents(nearByKey, roamKey string, meters int, callback DetectionHandler) error {
	cmd := fmt.Sprintf("NEARBY %s FENCE ROAM %s * %d", nearByKey, roamKey, meters)
	return es.streamDetection(cmd, callback)
}

func (es *Tile38EventStream) streamDetection(cmd string, callback DetectionHandler) error {
	conn, err := net.Dial("tcp", es.addr)
	if err != nil {
		return err
	}
	defer conn.Close()

	log.Println("REDIS DEBUG:", cmd)
	if _, err = fmt.Fprintf(conn, cmd+"\r\n"); err != nil {
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
	Lon          float64 `json:"lon"`
	Lat          float64 `json:"lat"`
	NearByFeatID string  `json:"near_by_feat_id"`
	NearByMeters float64 `json:"near_by_meters"`
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
	nearByFeatID, nearByMeters := gjson.Get(msg, "nearby.id").String(), gjson.Get(msg, "nearby.meters").Float()
	return &Detection{featID, lon, lat, nearByFeatID, nearByMeters}, nil
}
