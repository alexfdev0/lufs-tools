package main

import (
	"os"
	"strconv"
	"fmt"
)

func main() {
	if len(os.Args) < 7 {
		fmt.Println("Usage: mkfs.lufs <disk image> <drive size (bytes)> <drive name> <file start location> <header start location>")
		os.Exit(1)
	}

	disk := "" 
	drive_size := ""
	drive_name := ""
	drive_files_start := ""
	drive_header_start := ""
	rebuild := false

	for i := 1; i < len(os.Args); i++ {
		arg := os.Args[i]

		switch arg {
		case "-d":
			disk = os.Args[i + 1]
			i++
		case "-s":
			drive_size = os.Args[i + 1]
			i++
		case "-n":
			drive_name = os.Args[i + 1]
			i++
		case "-fst":
			drive_files_start = os.Args[i + 1]
			i++
		case "-hst":
			drive_header_start = os.Args[i + 1]
			i++	
		case "-rebuild-bs":
			rebuild = true
		}
	}	

	contents, err := os.ReadFile(disk)
	if err != nil {
		os.Exit(1)
	}

	svalue, err := strconv.ParseInt(drive_size, 0, 32)
	if err != nil {
		fmt.Println("Error: invalid drive size")
		os.Exit(1)
	}

	if len(drive_name) > 16 {
		fmt.Println("Error: drive name too big (max 16 bytes)")
		os.Exit(1)
	}

	stvalue, err := strconv.ParseInt(drive_files_start, 0, 32)
	if err != nil {
		fmt.Println("Error: invalid file start location")
		os.Exit(1)
	}

	hstvalue_, err := strconv.ParseInt(drive_header_start, 0, 32)
	if err != nil {
		fmt.Println("Error: invalid header start location")
		os.Exit(1)
	}
	hstvalue := int(hstvalue_)


	// LUFS header
	contents[hstvalue] = byte(0x4C)
	contents[hstvalue + 1] = byte(0x55)
	contents[hstvalue + 2] = byte(0x46)
	contents[hstvalue + 3] = byte(0x53)
	
	// DRIVE SIZE
	contents[hstvalue + 4] = byte(svalue >> 24)
	contents[hstvalue + 5] = byte(svalue >> 16)
	contents[hstvalue + 6] = byte(svalue >> 8)
	contents[hstvalue + 7] = byte(svalue & 0xFF)

	// DRIVE NAME
	for i := 0; i < 16; i++ {
		if i < len(drive_name) {
			contents[(hstvalue + 8) + i] = drive_name[i]
		} else {
			contents[(hstvalue + 8) + i] = byte(0x00)
		}
	}

	// START LOCATION
	contents[hstvalue + 24] = byte(stvalue >> 24)
	contents[hstvalue + 25] = byte(stvalue >> 16)
	contents[hstvalue + 26] = byte(stvalue >> 8)
	contents[hstvalue + 27] = byte(stvalue & 0xFF)

	// NEXT FILE LOCATION
	contents[hstvalue + 28] = byte(stvalue >> 24)
	contents[hstvalue + 29] = byte(stvalue >> 16)
	contents[hstvalue + 30] = byte(stvalue >> 8)
	contents[hstvalue + 31] = byte(stvalue & 0xFF)	

	if rebuild == true {	
		contents[492] = byte((hstvalue + 32) >> 8)
		contents[493] = byte((hstvalue + 32) & 0xFF)
	}	

	os.WriteFile(disk, contents, 0644)
}
