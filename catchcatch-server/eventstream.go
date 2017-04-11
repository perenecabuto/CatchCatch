package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"strings"
	"time"

	"github.com/tidwall/gjson"
)

// EventStream listen to geofence events and notifiy detection
type EventStream interface {
	StreamNearByEvents(nearByKey, roamKey string, meters int, callback DetectionHandler) error
	StreamIntersects(intersectKey, onKey, onKeyID string, callback DetectionHandler) error
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
func (es *Tile38EventStream) StreamNearByEvents(nearByKey, roamKey string, meters int, callback DetectionHandler) error {
	cmd := fmt.Sprintf("NEARBY %s FENCE ROAM %s * %d", nearByKey, roamKey, meters)
	return streamDetection(es.addr, cmd, callback)
}

// StreamIntersects stream intersection events
func (es *Tile38EventStream) StreamIntersects(intersectKey, onKey, onKeyID string, callback DetectionHandler) error {
	//INTERSECTS player FENCE DETECT inside,enter,exit GET geofences uuu
	cmd := fmt.Sprintf("INTERSECTS %s FENCE DETECT inside,enter,exit GET %s %s", intersectKey, onKey, onKeyID)
	return streamDetection(es.addr, cmd, overrideNearByFeatIDWrapper(onKeyID, callback))
}

func overrideNearByFeatIDWrapper(nearByFeatID string, handler DetectionHandler) DetectionHandler {
	return func(d *Detection) {
		d.NearByFeatID = nearByFeatID
		handler(d)
	}
}

// IntersectsEvent ...
type IntersectsEvent string

// IntersectsEvent none,inside,enter,exit,outside
const (
	None    IntersectsEvent = ""
	Inside  IntersectsEvent = "inside"
	Enter   IntersectsEvent = "enter"
	Exit    IntersectsEvent = "exit"
	Outside IntersectsEvent = "outside"
)

// Detection represents an detected event
type Detection struct {
	FeatID       string          `json:"feat_id"`
	Lat          float64         `json:"lat"`
	Lon          float64         `json:"lon"`
	NearByFeatID string          `json:"near_by_feat_id"`
	NearByMeters float64         `json:"near_by_meters"`
	Intersects   IntersectsEvent `json:"intersects"`
}

func (d Detection) String() string {
	data, _ := json.Marshal(d)
	return string(data)
}

// DetectionHandler is called when a an event is detected
type DetectionHandler func(*Detection)

// DetectionError ...
type DetectionError string

func (err DetectionError) Error() string {
	return string("DetectionError: " + err)
}

func streamDetection(addr string, cmd string, callback DetectionHandler) error {
	conn, err := listenTo(addr, cmd)
	if err != nil {
		return err
	}
	defer conn.Close()

	buf, n := make([]byte, 4096), 0
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

func handleDetection(msg string) (*Detection, error) {
	featID, coords := gjson.Get(msg, "id").String(), gjson.Get(msg, "object.coordinates").Array()
	if len(coords) != 2 {
		return nil, DetectionError("invalid coords - msg:\n" + msg)
	}
	lat, lon := coords[0].Float(), coords[1].Float()
	nearByFeatID, nearByMeters := gjson.Get(msg, "nearby.id").String(), gjson.Get(msg, "nearby.meters").Float()
	detect := gjson.Get(msg, "detect").String()
	intersects := IntersectsEvent(detect)
	return &Detection{featID, lat, lon, nearByFeatID, nearByMeters, intersects}, nil
}

func listenTo(addr, cmd string) (net.Conn, error) {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return nil, err
	}

	log.Println("REDIS DEBUG:", cmd)
	if _, err = fmt.Fprintf(conn, cmd+"\r\n"); err != nil {
		return nil, err
	}
	buf := make([]byte, 4096)
	n, err := conn.Read(buf)
	res := string(buf[:n])
	if res != "+OK\r\n" {
		return nil, fmt.Errorf("expected OK, got '%v'", res)
	}
	return conn, nil
}
