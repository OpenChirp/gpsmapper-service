package main

import (
	"bufio"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"strings"
	"time"
)

var headerStr, footerStr string

// GPScoord struct stores the device ID and associated
// GPS data including timestamp
type GPScoord struct {
	Lat, Lon, Alt float64
	timestamp     time.Time
	deviceID      string
	deviceName    string
	deviceOptions string
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}

// PrintCoord is a debug function that prints a GPScoord struct on the terminal
func PrintCoord(v GPScoord) {
	fmt.Println("DeviceID: ", v.deviceID)
	fmt.Println("\tDeviceName: ", v.deviceName)
	fmt.Printf("\tLat Lon Alt: %f %f %f\n", v.Lat, v.Lon, v.Alt)
	fmt.Println("\tTimestamp: ", v.timestamp.Format(time.RFC3339))
	fmt.Println("\tDeviceOptions: ", v.deviceOptions)

}

// LoadCoords will read the device and coordinate file from disk
// into memory.
func LoadCoords(m map[string]GPScoord, coordfile string) error {
	fileHandle, _ := os.Open(coordfile)
	defer fileHandle.Close()

	fileScanner := bufio.NewScanner(fileHandle)
	for fileScanner.Scan() {

		r := csv.NewReader(strings.NewReader(fileScanner.Text()))
		var myCoord GPScoord

		for {
			record, err := r.Read()
			if err == io.EOF {
				break
			}
			if err != nil {
				log.Fatal(err)
			}
			myCoord.deviceID = record[0]
			myCoord.Lat, err = strconv.ParseFloat(record[1], 64)
			myCoord.Lon, err = strconv.ParseFloat(record[2], 64)
			myCoord.Alt, err = strconv.ParseFloat(record[3], 64)
			tparse, err := time.Parse(time.RFC3339, record[4])
			myCoord.deviceName = record[5]
			myCoord.deviceOptions = record[6]
			myCoord.timestamp = tparse
			m[record[0]] = myCoord
		}

	}
	return nil
}

// SaveCoords writes the current GPScoord map file to disk.
// The function uses the path specfified by coordfile
func SaveCoords(m map[string]GPScoord, coordfile string) error {
	//var dev string
	//var err error
	f, err := os.Create(coordfile)
	check(err)
	defer f.Close()

	for k := range m {
		var myCoord GPScoord
		myCoord = m[k]
		devStr := fmt.Sprintf("\"%s\",%f,%f,%f,%s,\"%s\",\"%s\"\n", myCoord.deviceID, myCoord.Lat, myCoord.Lon, myCoord.Alt, myCoord.timestamp.Format(time.RFC3339), myCoord.deviceName, myCoord.deviceOptions)
		n3, err := f.WriteString(devStr)
		if n3 < 1 {
			err = errors.New("not able to write to file")
		}
		check(err)
	}

	f.Sync()

	return nil
}

// LoadMapTemplate loads the html header and footer into
// memory from the specified files. The GenerateMap function
// will then write the header, custom data, footer to the map
// index.html file
func LoadMapTemplate(header, footer string) error {
	b, err := ioutil.ReadFile(header) // just pass the file name
	if err != nil {
		fmt.Print(err)
	}

	headerStr = string(b) // convert content to a 'string'

	//fmt.Println(headerStr) // print the content as a 'string'

	c, err := ioutil.ReadFile(footer) // just pass the file name
	if err != nil {
		fmt.Print(err)
	}

	footerStr = string(c) // convert content to a 'string'

	//fmt.Println(footerStr) // print the content as a 'string'
	return nil
}

// GenerateMap will write the latest GPScoord map to file
func GenerateMap(m map[string]GPScoord, mapFile string) error {
	f, err := os.Create(mapFile)
	check(err)
	defer f.Close()
	n, err := f.WriteString(headerStr)
	if n < 1 {
		err = errors.New("not able to write header")
	}

	//	L.marker([40.44362, -79.94313]).addTo(mymap).bindPopup("<b>Hello world!</b><br />I am a popup.").openPopup();
	for k := range m {
		var myCoord GPScoord
		myCoord = m[k]
		devStr := fmt.Sprintf("L.marker([%f,%f]).addTo(mymap).bindPopup(\"%s\");\n", myCoord.Lat, myCoord.Lon, myCoord.deviceName)
		n2, err := f.WriteString(devStr)
		if n2 < 1 {
			err = errors.New("not able to write to file")
		}
		f.Sync()
		check(err)
	}

	n, err = f.WriteString(footerStr)
	if n < 1 {
		err = errors.New("not able to write header")
	}
	f.Sync()
	check(err)
	return nil
}

func testrig() {

	m := make(map[string]GPScoord)

	LoadMapTemplate("header.txt", "footer.txt")

	LoadCoords(m, "gpsDB.dat")

	// Define a new device with coordinate
	var myCoord GPScoord

	myCoord.Lat = 40.44436
	myCoord.Lon = -79.91911
	myCoord.Alt = 0.0
	myCoord.deviceID = "59f61c52f230cf7055615d2f"
	myCoord.deviceName = "agr LoRa Gateway"
	myCoord.deviceOptions = "LoRa Gateway"
	myCoord.timestamp = time.Now()

	// Add a device to the map
	m[myCoord.deviceID] = myCoord

	GenerateMap(m, "leaflet/index.html")

	//PrintCoord(m["12345"])

	SaveCoords(m, "gpsDB.dat")

}
