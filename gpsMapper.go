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
	deviceType    string
	ownerID       string
	isPublic      string
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
	fmt.Println("\tdeviceType: ", v.deviceType)

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
			myCoord.deviceType = record[6]
			myCoord.ownerID = record[7]
			myCoord.isPublic = record[8]
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
		devStr := fmt.Sprintf("\"%s\",%f,%f,%f,%s,\"%s\",\"%s\",\"%s\",\"%s\"\n",
			myCoord.deviceID, myCoord.Lat, myCoord.Lon, myCoord.Alt, myCoord.timestamp.Format(time.RFC3339),
			myCoord.deviceName, myCoord.deviceType, myCoord.ownerID, myCoord.isPublic)
		n3, err := f.WriteString(devStr)
		if n3 < 1 {
			err = errors.New("not able to write to file")
		}
		check(err)
	}

	f.Sync()

	return nil
}

// GenerateStaticMap will write the latest GPScoord map to file
func GenerateStaticMap(m map[string]GPScoord, mapFile string) error {
	f, err := os.Create(mapFile)
	check(err)
	defer f.Close()
	var n int
	idCnt := 1
	// Write JSON header
	n, err = f.WriteString("var gatewayCoords = { \"type\": \"FeatureCollection\",\"features\": [\n")
	if n < 1 {
		err = errors.New("not able to write to file")
	}

	//	L.marker([40.44362, -79.94313]).addTo(mymap).bindPopup("<b>Hello world!</b><br />I am a popup.").openPopup();
	for k := range m {
		var myCoord GPScoord
		myCoord = m[k]
		validGPS := false

		if (myCoord.Lat) != 0 && (myCoord.Lon) != 0 {
			validGPS = true
		}

		isPublic := false

		// Default to public if option skipped...
		if len(myCoord.isPublic) == 0 ||
			strings.Contains(strings.ToLower(myCoord.isPublic), "1") == true ||
			strings.Contains(strings.ToLower(myCoord.isPublic), "public") == true ||
			strings.Contains(strings.ToLower(myCoord.isPublic), "true") == true ||
			strings.Contains(strings.ToLower(myCoord.isPublic), "yes") == true {
			isPublic = true
		}

		if (validGPS == true) && isPublic == true && strings.Contains(strings.ToLower(myCoord.deviceType), "gateway") == true {
			linkStr := fmt.Sprintf("%s: <a href='http://www.openchirp.io/home/device/%s' target='_blank' >%s</a>", myCoord.deviceType, myCoord.deviceID, myCoord.deviceName)
			devStr := fmt.Sprintf("\t{\n\t\t\"geometry\": {\n\t\t\t\"type\": \"Point\",\n\t\t\t\"coordinates\": [%f,%f]\n\t\t},\n\t\t\"type\": \"Feature\",\n\t\t\"properties\": {\n\t\t\t\"popupContent\": \"%s\"\n\t\t},\n\t\t\"id\": %d\n\t},\n", myCoord.Lon, myCoord.Lat, linkStr, idCnt)
			idCnt++
			n, err = f.WriteString(devStr)
			if n < 1 {
				err = errors.New("not able to write to file")
			}
			f.Sync()
			check(err)
		}
	}

	// Write JSON footer
	n, err = f.WriteString("]};")
	if n < 1 {
		err = errors.New("not able to write to file")
	}

	// Write JSON header
	n, err = f.WriteString("\n\nvar transducerCoords = { \"type\": \"FeatureCollection\",\"features\": [\n")
	if n < 1 {
		err = errors.New("not able to write to file")
	}

	//	L.marker([40.44362, -79.94313]).addTo(mymap).bindPopup("<b>Hello world!</b><br />I am a popup.").openPopup();
	for k := range m {
		var myCoord GPScoord
		myCoord = m[k]
		validGPS := false

		if (myCoord.Lat) != 0 && (myCoord.Lon) != 0 {
			validGPS = true
		}

		isPublic := false

		// Default to public if option skipped...
		if len(myCoord.isPublic) == 0 ||
			strings.Contains(strings.ToLower(myCoord.isPublic), "1") == true ||
			strings.Contains(strings.ToLower(myCoord.isPublic), "public") == true ||
			strings.Contains(strings.ToLower(myCoord.isPublic), "true") == true ||
			strings.Contains(strings.ToLower(myCoord.isPublic), "yes") == true {
			isPublic = true
		}

		if (validGPS == true) && isPublic == true && (strings.Contains(strings.ToLower(myCoord.deviceType), "gateway") == false) {
			//devStr := fmt.Sprintf("L.marker([%f,%f]).addTo(mymap).bindPopup(\"%s\");\n", myCoord.Lat, myCoord.Lon, myCoord.deviceName)
			linkStr := fmt.Sprintf("%s: <a href='http://www.openchirp.io/home/device/%s' target='_blank'>%s</a>", myCoord.deviceType, myCoord.deviceID, myCoord.deviceName)
			devStr := fmt.Sprintf("\t{\n\t\t\"geometry\": {\n\t\t\t\"type\": \"Point\",\n\t\t\t\"coordinates\": [%f,%f]\n\t\t},\n\t\t\"type\": \"Feature\",\n\t\t\"properties\": {\n\t\t\t\"popupContent\": \"%s\"\n\t\t},\n\t\t\"id\": %d\n\t},\n", myCoord.Lon, myCoord.Lat, linkStr, idCnt)
			idCnt++
			n, err = f.WriteString(devStr)
			if n < 1 {
				err = errors.New("not able to write to file")
			}
			f.Sync()
			check(err)
		}
	}

	// Write JSON footer
	n, err = f.WriteString("]};")
	if n < 1 {
		err = errors.New("not able to write to file")
	}

	f.Sync()
	check(err)
	return nil
}

// LoadMapTemplate loads the html header and footer into
// memory from the specified files. We use these when serving
// local maps
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

func testrig() {

	m := make(map[string]GPScoord)

	LoadCoords(m, "gpsDB.dat")

	// Define a new device with coordinate
	var myCoord GPScoord

	myCoord.Lat = 40.44436
	myCoord.Lon = -79.91911
	myCoord.Alt = 0.0
	myCoord.deviceID = "59f61c52f230cf7055615d2f"
	myCoord.deviceName = "agr LoRa Gateway"
	myCoord.deviceType = "gateway"
	myCoord.timestamp = time.Now()

	// Add a device to the map
	m[myCoord.deviceID] = myCoord

	GenerateStaticMap(m, "leaflet/index.html")

	//PrintCoord(m["12345"])

	SaveCoords(m, "gpsDB.dat")

}
