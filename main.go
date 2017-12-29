// Craig Hesling and Anthony Rowe
// December 27, 2017
//
// This is an OpenChirp service that captures the GPS coordinates created by a device
// and plots them using leaflet and OpenStreetMaps
package main

import (
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/openchirp/framework"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

// gpsCoordMap is a Global structure for storing device GPS Coordinates
var gpsCoordMap = make(map[string]GPScoord)

var nameLookup chan string

// Mutex to lock access to GPScoord
var mutex = &sync.Mutex{}

const (
	version string = "1.0"
)

const (
	// Set this value to true to have the service publish a service status of
	// "Running" each time it receives a device update event
	//
	// This could be used as a service alive pulse if enabled
	// Otherwise, the service status will indicate "Started" at the time the
	// service "Started" the client
	runningStatus = true
)

const (
	// The subscription key used to identify a messages types
	latitudeKey  = 0
	longitudeKey = 1
	altitudeKey  = 2
)

// Device holds any data you want to keep around for a specific
// device that has linked your service.
//
// In this example, we will keep track of the rawrx and rawtx message counts
type Device struct {
	devCoord GPScoord
}

// NewDevice is called by the framework when a new device has been linked.
func NewDevice() framework.Device {
	d := new(Device)
	// The following initialization is redundant in Go
	//d.devCoord.Lat = 0
	//d.rawTxCount = 0
	// Change type to the Device interface
	return framework.Device(d)
}

// ProcessLink is called once, during the initial setup of a
// device, and is provided the service config for the linking device.
func (d *Device) ProcessLink(ctrl *framework.DeviceControl) string {
	// This simply sets up console logging for our program.
	// Any time this logitem is use to print messages,
	// the key/value string "deviceid=<device_id>" is prepended to the line.
	logitem := log.WithField("deviceid", ctrl.Id())
	logitem.Debug("Linking with config:", ctrl.Config())

	//d.devCoord.deviceOptions =
	//var m map[string]string
	m := ctrl.Config()
	d.devCoord.deviceOptions = m["Marker Type"]
	d.devCoord.deviceID = ctrl.Id()
	linkStr := fmt.Sprintf("Device ID: <a href='http://www.openchirp.io/home/device/%s'>%s</a>", ctrl.Id(), ctrl.Id())
	d.devCoord.deviceName = linkStr // Change this later to be the name you want displayed...
	d.devCoord.timestamp = time.Now()

	// Test if we don't already have data for this entry
	_, ok := gpsCoordMap[d.devCoord.deviceID]
	if ok == false {
		mutex.Lock()
		gpsCoordMap[d.devCoord.deviceID] = d.devCoord
		mutex.Unlock()
	}
	//PrintCoord(d.devCoord)
	// Subscribe to subtopic "transducer/xxx"
	ctrl.Subscribe(framework.TransducerPrefix+"/latitude", latitudeKey)
	ctrl.Subscribe(framework.TransducerPrefix+"/longitude", longitudeKey)
	ctrl.Subscribe(framework.TransducerPrefix+"/altitude", altitudeKey)

	// Send name to lookup goroutine, this will get populate later
	// and then eventually saved
	nameLookup <- ctrl.Id()

	logitem.Debug("Finished Linking")

	// This message is sent to the service status for the linking device
	return "Success"
}

// ProcessUnlink is called once, when the service has been unlinked from
// the device.
func (d *Device) ProcessUnlink(ctrl *framework.DeviceControl) {
	logitem := log.WithField("deviceid", ctrl.Id())
	mutex.Lock()
	delete(gpsCoordMap, ctrl.Id())
	mutex.Unlock()
	logitem.Debug("Unlinked:")
	// The framework already handles unsubscribing from all
	// Device associted subtopics, so we don't need to call
	// ctrl.Unsubscribe.
}

// ProcessConfigChange is intended to handle a service config updates.
// If your program does not need to handle incremental config changes,
// simply return false, to indicate the config update was unhandled.
// The framework will then automatically issue a ProcessUnlink and then a
// ProcessLink, instead. Note, NewDevice is not called.
//
// For more information about this or other Device interface functions,
// please see https://godoc.org/github.com/OpenChirp/framework#Device .
func (d *Device) ProcessConfigChange(ctrl *framework.DeviceControl, cchanges, coriginal map[string]string) (string, bool) {
	logitem := log.WithField("deviceid", ctrl.Id())

	logitem.Debug("Ignoring Config Change:", cchanges)
	return "", false

	// If we have processed this config change, we should return the
	// new service status message and true.
	//
	//logitem.Debug("Processing Config Change:", cchanges)
	//return "Sucessfully updated", true
}

// ProcessMessage is called upon receiving a pubsub message destined for
// this device.
// Along with the standard DeviceControl object, the handler is provided
// a Message object, which contains the received message's payload,
// subtopic, and the provided Subscribe key.
func (d *Device) ProcessMessage(ctrl *framework.DeviceControl, msg framework.Message) {
	logitem := log.WithField("deviceid", ctrl.Id())
	logitem.Debugf("Processing Message: %v: [ % #x ]", msg.Key(), msg.Payload())
	var err error

	// Copy values from global map here so as to not overwrite existing values...
	mutex.Lock()
	d.devCoord = gpsCoordMap[ctrl.Id()]
	mutex.Unlock()

	//linkStr := fmt.Sprintf("Device ID: <a href='http://www.openchirp.io/home/device/%s'>%s</a>", ctrl.Id(), ctrl.Id())
	//d.devCoord.deviceName = linkStr // Change this later to be the name you want displayed...

	//myCoord.deviceOptions = "LoRa Gateway"
	d.devCoord.timestamp = time.Now()

	if msg.Key().(int) == latitudeKey {
		payloadStr := fmt.Sprintf("%s", msg.Payload())
		log.Info("Setting Lat:" + payloadStr + " for " + ctrl.Id())
		d.devCoord.Lat, err = strconv.ParseFloat(payloadStr, 64)
		if err == nil {
			mutex.Lock()
			gpsCoordMap[d.devCoord.deviceID] = d.devCoord
			mutex.Unlock()
		}
	} else if msg.Key().(int) == longitudeKey {
		payloadStr := fmt.Sprintf("%s", msg.Payload())
		log.Info("Setting Lon:" + payloadStr + " for " + ctrl.Id())
		d.devCoord.Lon, err = strconv.ParseFloat(payloadStr, 64)
		if err == nil {
			mutex.Lock()
			gpsCoordMap[d.devCoord.deviceID] = d.devCoord
			mutex.Unlock()
		}

	} else {
		//logitem.Errorln("Received unassociated message")
	}
}

// deviceNameLookup waits on the "name" channel for a device to lookup
// and then makes the REST call and pushes it back into the data structure
func deviceNameLookup(c framework.Client) {

	for {
		name := <-nameLookup
		log.Info("Name Lookup for: " + name)
		dev, err := c.FetchDeviceInfo(name)
		check(err)

		mutex.Lock()
		myCoord := gpsCoordMap[name]
		mutex.Unlock()
		linkStr := fmt.Sprintf("%s: <a href='http://www.openchirp.io/home/device/%s'>%s</a>", myCoord.deviceOptions, name, dev.Name)
		mutex.Lock()
		myCoord.deviceName = linkStr
		gpsCoordMap[name] = myCoord
		mutex.Unlock()
		log.Info("Setting Name: " + dev.Name)
	}
}

func gpsMapperWorker(outputFile string, updateSecs int) {

	for {
		time.Sleep(time.Duration(updateSecs * 1000000000))
		mutex.Lock()
		GenerateMap(gpsCoordMap, outputFile)
		SaveCoords(gpsCoordMap, "gpsDB.dat")
		mutex.Unlock()
		log.Info("Updated Map Output")

	}
}

// run is the main function that gets called once from main()
func run(ctx *cli.Context) error {
	/* Set logging level (verbosity) */
	log.SetLevel(log.Level(uint32(ctx.Int("log-level"))))

	log.Info("Starting GPS Mapper Service")

	nameLookup = make(chan string, 1000)

	mutex.Lock()
	LoadCoords(gpsCoordMap, "gpsDB.dat")
	mutex.Unlock()
	log.Info("Loaded Device Location Data from file")

	go gpsMapperWorker(ctx.String("geojson-path"), 10)

	// LoadMapTemplate("header.txt", "footer.txt")
	// log.Info("Loaded Map Templates")

	/* Start framework service client */
	//c, err := framework.StartServiceClientManaged(
	c, err := framework.StartServiceClientManaged(

		ctx.String("framework-server"),
		ctx.String("mqtt-server"),
		ctx.String("service-id"),
		ctx.String("service-token"),
		"Unexpected disconnect!",
		NewDevice)
	if err != nil {
		log.Error("Failed to StartServiceClient: ", err)
		return cli.NewExitError(nil, 1)
	}

	defer c.StopClient()
	log.Info("Started service")

	go deviceNameLookup(c.Client)

	/* Post service's global status */
	if err := c.SetStatus("Starting"); err != nil {
		log.Error("Failed to publish service status: ", err)
		return cli.NewExitError(nil, 1)
	}
	log.Info("Published Service Status")

	/* Setup signal channel */
	signals := make(chan os.Signal)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)

	/* Post service status indicating I started */
	if err := c.SetStatus("Started"); err != nil {
		log.Error("Failed to publish service status: ", err)
		return cli.NewExitError(nil, 1)
	}
	log.Info("Published Service Status")

	/* Wait on a signal */
	sig := <-signals
	log.Info("Received signal ", sig)
	log.Warning("Shutting down")

	/* Post service's global status */
	if err := c.SetStatus("Shutting down"); err != nil {
		log.Error("Failed to publish service status: ", err)
	}
	log.Info("Published service status")

	return nil
}

func main() {
	/* Parse arguments and environemtnal variable */
	app := cli.NewApp()
	app.Name = "gpsMapper-service"
	app.Usage = ""
	app.Copyright = "See https://github.com/openchirp/example-service for copyright information"
	app.Version = version
	app.Action = run
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:   "framework-server",
			Usage:  "OpenChirp framework server's URI",
			Value:  "http://localhost:7000",
			EnvVar: "FRAMEWORK_SERVER",
		},
		cli.StringFlag{
			Name:   "mqtt-server",
			Usage:  "MQTT server's URI (e.g. scheme://host:port where scheme is tcp or tls)",
			Value:  "tls://localhost:1883",
			EnvVar: "MQTT_SERVER",
		},
		cli.StringFlag{
			Name:   "service-id",
			Usage:  "OpenChirp service id",
			EnvVar: "SERVICE_ID",
		},
		cli.StringFlag{
			Name:   "service-token",
			Usage:  "OpenChirp service token",
			EnvVar: "SERVICE_TOKEN",
		},
		cli.IntFlag{
			Name:   "log-level",
			Value:  4,
			Usage:  "debug=5, info=4, warning=3, error=2, fatal=1, panic=0",
			EnvVar: "LOG_LEVEL",
		},
		cli.StringFlag{
			Name:   "geojson-path",
			Usage:  "Path to geojson file for leaflet",
			Value:  "oc-geojason.js",
			EnvVar: "GEOJSON_PATH",
		},
	}

	/* Launch the application */
	app.Run(os.Args)
}
