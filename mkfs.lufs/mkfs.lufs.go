package main

import (
	"os"
	"strconv"
	"fmt"
	"bufio"
	"time"
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

	hstvalue_, err := strconv.ParseInt(drive_header_start, 0, 32)
	if err != nil {
		fmt.Println("Error: invalid header start location")
		os.Exit(1)
	}
	hstvalue := int(hstvalue_)

	fmt.Println("\033[31mWarning: This tool will overwrite portions of your disk. DO NOT USE if you do not know what you're doing as it can damage your disk data.\033[0m")
	fmt.Println("Press any key to continue...")
	bufio.NewReader(os.Stdin).ReadBytes('\n')

	
	fmt.Printf("Overwriting '" + disk + "' in ")
	for i := 5; i > 0; i-- {
		fmt.Printf("%d...", i)
		time.Sleep(time.Duration(1000) * time.Millisecond)
	}
	fmt.Printf("\n")

	f, err := os.OpenFile(disk, os.O_RDWR | os.O_SYNC, 0)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	defer f.Close()

	contents := make([]byte, 512)
	haddr := (hstvalue / 512) * 512
	_, err = f.ReadAt(contents, int64(haddr))
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	stage1 := make([]byte, 512)
	_, err = f.ReadAt(stage1, int64(0))
	if err != nil {
		fmt.Println(err)
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


	// LUFS header
	contents[0] = byte(0x4C)
	contents[1] = byte(0x55)
	contents[2] = byte(0x46)
	contents[3] = byte(0x53)
	
	// DRIVE SIZE
	contents[4] = byte(svalue >> 24)
	contents[5] = byte(svalue >> 16)
	contents[6] = byte(svalue >> 8)
	contents[7] = byte(svalue & 0xFF)

	// DRIVE NAME
	for i := 0; i < 16; i++ {
		if i < len(drive_name) {
			contents[8 + i] = drive_name[i]
		} else {
			contents[8 + i] = byte(0x00)
		}
	}

	// START LOCATION
	contents[24] = byte(stvalue >> 24)
	contents[25] = byte(stvalue >> 16)
	contents[26] = byte(stvalue >> 8)
	contents[27] = byte(stvalue & 0xFF)

	// NEXT FILE LOCATION
	contents[28] = byte(stvalue >> 24)
	contents[29] = byte(stvalue >> 16)
	contents[30] = byte(stvalue >> 8)
	contents[31] = byte(stvalue & 0xFF)	

	if rebuild == true {	
		stage1[492] = byte((32) >> 8)
		stage1[493] = byte((32) & 0xFF)
	}	

	f.WriteAt(stage1, int64(0))
	f.WriteAt(contents, int64(haddr))
	os.WriteFile(disk, contents, 0644)
}
