package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"strings"
	"time"

	"github.com/perenecabuto/CatchCatch/server/execfunc"
	protocol "github.com/quorzz/redis-protocol"
	"github.com/tidwall/gjson"
)


// DetectEvent ...
type DetectEvent string

// DetectEvent none,inside,enter,exit,outside
const (
	None    DetectEvent  = ""
	Inside  DetectEvent  = "inside"
	Enter   DetectEvent  = "enter"
	Exit    DetectEvent  = "exit"
	Outside DetectEvent  = "outside"
)

// Detection represents an detected event
type Detection struct {
	FeatID       string      `json:"feat_id"`
	Lat          float64     `json:"lat"`
	Lon          float64     `json:"lon"`
	NearByFeatID string      `json:"near_by_feat_id"`
	NearByMeters float64     `json:"near_by_meters"`
	Intersects   DetectEvent `json:"intersects"`
	Coordinates  string      `json:"coordinates"`
}

func (d Detection) String() string {
	data, _ := json.Marshal(d)
	return string(data)
}

// DetectionHandler is called when a an event is detected
type DetectionHandler func(*Detection) error

// DetectionError ...
type DetectionError string

func (err DetectionError) Error() string {
	return string("DetectionError: " + err)
}

// EventStream listen to geofence events and notifiy detection
type EventStream interface {
	StreamNearByEvents(ctx context.Context, nearByKey, roamKey, roamID string, meters int, callback DetectionHandler) error
	StreamIntersects(ctx context.Context, intersectKey, onKey, onKeyID string, callback DetectionHandler) error
}

// Tile38EventStream Tile38 implementation of EventStream
type Tile38EventStream struct {
	addr string
}

// NewEventStream creates a Tile38EventStream
func NewEventStream(addr string) EventStream {
	return &Tile38EventStream{addr}
}

// StreamNearByEvents stream proximation events
func (es *Tile38EventStream) StreamNearByEvents(ctx context.Context, nearByKey, roamKey string, roamID string, meters int, callback DetectionHandler) error {
	cmd := query{"NEARBY", nearByKey, "FENCE", "ROAM", roamKey, roamID, meters}
	return es.StreamDetection(ctx, cmd, callback)
}

// StreamIntersects stream events of the objects (of type intersectKey) moving inside object (onKey + onKeyID)
func (es *Tile38EventStream) StreamIntersects(ctx context.Context, intersectKey, onKey, onKeyID string, callback DetectionHandler) error {
	cmd := query{"INTERSECTS", intersectKey, "FENCE", "DETECT", "inside,enter,exit", "GET", onKey, onKeyID}
	callback = overrideNearByFeatIDWrapper(onKeyID, callback)
	return es.StreamDetection(ctx, cmd, callback)
}

func (es *Tile38EventStream) StreamDetection(ctx context.Context, q query, callback DetectionHandler) error {
	interval := 50 * time.Millisecond
	conn, err := listenTo(es.addr, q)
	if err != nil {
		return err
	}
	defer conn.Close()

	buf, n := make([]byte, 4096), 0
	t := time.NewTicker(interval)
	defer t.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Printf("eventstream:query:stop:%s", q.String())
			return nil
		case <-t.C:
			conn.SetReadDeadline(time.Now().Add(interval))
			if n, err = conn.Read(buf); err != nil {
				continue
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
				err = execfunc.WithRecover(func() error {
					return callback(detected)
				})
				if err != nil {
					return err
				}
			}
		}
	}
}

func overrideNearByFeatIDWrapper(nearByFeatID string, handler DetectionHandler) DetectionHandler {
	return func(d *Detection) error {
		d.NearByFeatID = nearByFeatID
		return handler(d)
	}
}

func handleDetection(msg string) (*Detection, error) {
	featID := gjson.Get(msg, "id").String()
	lat, lon := 0.0, 0.0
	latlon := gjson.Get(msg, "object.coordinates").Array()
	if len(latlon) == 2 {
		lat, lon = latlon[1].Float(), latlon[0].Float()
	}
	nearByFeatID, nearByMeters := gjson.Get(msg, "nearby.id").String(), gjson.Get(msg, "nearby.meters").Float()
	intersects := None
	if gjson.Get(msg, "command").String() == "del" {
		intersects = Exit
	} else if detect := gjson.Get(msg, "detect").String(); detect != "" {
		if detect == "roam" {
			intersects = Inside
		} else {
			intersects = DetectEvent(detect)
		}
	}
	coords := fmt.Sprintf(`{"type":"%s","coordinates":%s}`,
		gjson.Get(msg, "object.type").String(),
		gjson.Get(msg, "object.coordinates").String(),
	)
	return &Detection{featID, lat, lon, nearByFeatID, nearByMeters, intersects, coords}, nil
}

func listenTo(addr string, q query) (net.Conn, error) {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return nil, err
	}

	log.Println("EventStream: REDIS DEBUG:", q)
	if _, err = fmt.Fprintf(conn, q.cmd()); err != nil {
		return nil, err
	}
	buf := make([]byte, 4096)
	n, err := conn.Read(buf)
	res := string(buf[:n])
	if res != "+OK\r\n" {
		return nil, fmt.Errorf("expected OK, got '%v' - query: %s", res, q)
	}
	return conn, nil
}

type query []interface{}

func (q query) String() string {
	args := q
	return fmt.Sprintln(args...)
}

func (q query) cmd() string {
	cmd, _ := protocol.PackCommand(q...)
	return string(cmd)
}
